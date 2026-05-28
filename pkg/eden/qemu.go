package eden

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/edensdn"
	"github.com/lf-edge/eden/pkg/utils"
	sdnapi "github.com/lf-edge/eden/sdn/vm/api"
	log "github.com/sirupsen/logrus"
)

// StartSWTPM starts swtpm process and use stateDir as state, log, pid and socket location
func StartSWTPM(stateDir string) error {
	if err := os.MkdirAll(stateDir, 0777); err != nil {
		return err
	}
	command := "swtpm"
	logFile := filepath.Join(stateDir, fmt.Sprintf("%s.log", command))
	pidFile := filepath.Join(stateDir, fmt.Sprintf("%s.pid", command))
	options := fmt.Sprintf("socket --tpmstate dir=%s --ctrl type=unixio,path=%s --log level=20 --tpm2", stateDir, filepath.Join(stateDir, defaults.DefaultSwtpmSockFile))
	if err := utils.RunCommandNohup(command, logFile, pidFile, strings.Fields(options)...); err != nil {
		return fmt.Errorf("StartSWTPM: %s", err)
	}
	return nil
}

// StopSWTPM stops swtpm process using pid from stateDir
func StopSWTPM(stateDir string) error {
	command := "swtpm"
	pidFile := filepath.Join(stateDir, fmt.Sprintf("%s.pid", command))
	return utils.StopCommandWithPid(pidFile)
}

// tcpPortReservation describes a TCP listen that QEMU will attempt at
// spawn time. The host field matches the string passed to QEMU (e.g.
// "localhost" for the chardev/monitor sockets, "0.0.0.0" for hostfwd
// and serial-PCI backends) so a probe with net.Listen sees the same
// conflict QEMU would.
type tcpPortReservation struct {
	host string
	port int
	role string
}

// checkPortFree probes whether the given TCP host:port can be bound
// right now. Returns an actionable error if not. The probe is
// best-effort pre-flight: the port could in principle race closed
// between the probe and QEMU's bind, but in practice this catches
// stuck-process leaks (a stranded QEMU from a prior run holding the
// EVE console port, etc.) before we exec QEMU and wait out the
// onboarding timeout on a VM that never started.
func checkPortFree(host string, port int, role string) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("%s port %s already in use: %v "+
			"(find owner with: sudo ss -tlnp 'sport = :%d')",
			role, addr, err, port)
	}
	_ = ln.Close()
	return nil
}

func startQMPLogger(qmpSockFile string, qmpLogFile string) error {
	shellcmd := fmt.Sprintf(
		"echo '{\"execute\": \"qmp_capabilities\"}' | "+
			"socat -t0 -,ignoreeof UNIX-CONNECT:%s > %s",
		qmpSockFile, qmpLogFile)
	opts := []string{
		"-c", shellcmd,
	}

	var err error

	// Retry a few times if socket is not available yet
	n := 5
	for n > 0 {
		if err = utils.RunCommandNohup("sh", "", "", opts...); err != nil {
			time.Sleep(1 * time.Second)
			n--
			continue
		}
		break
	}
	if err != nil {
		return fmt.Errorf("startQMPLogger: can't connect to the QMP socket, presumably QEMU did not start")
	}

	return nil
}

// StartEVEQemu function run EVE in qemu
func StartEVEQemu(qemuARCH, qemuOS, eveImageFile, imageFormat string, isInstaller bool,
	qemuSMBIOSSerial string, eveTelnetPort, qemuMonitorPort, netDevBasePort int,
	qemuHostFwd map[string]string, qemuAccel bool, qemuConfigFile, logFile, pidFile string,
	netModel sdnapi.NetworkModel, withSDN bool, tapInterface, usbImagePath string,
	swtpm, can, serialPCI, foreground bool) (err error) {
	var qemuCommand, qemuOptions string
	qemuOptions += "-nodefaults -no-user-config "
	netDev := "virtio-net-pci"
	tpmDev := "tpm-tis"
	if qemuARCH == "" {
		qemuARCH = runtime.GOARCH
	} else {
		qemuARCH = strings.ToLower(qemuARCH)
	}
	switch qemuARCH {
	case "amd64":
		qemuCommand = "qemu-system-x86_64"
		if qemuAccel {
			if qemuOS == "darwin" {
				qemuOptions += defaults.DefaultQemuAccelDarwin
			} else {
				qemuOptions += defaults.DefaultQemuAccelLinuxAmd64
				// to support pass-through of virtio-net-pci
				netDev = fmt.Sprintf("%s,disable-legacy=on,disable-modern=off,iommu_platform=on", netDev)
			}
		} else {
			qemuOptions += defaults.DefaultQemuAmd64
		}
	case "arm64":
		qemuCommand = "qemu-system-aarch64"
		if qemuAccel {
			if qemuOS == "darwin" {
				qemuOptions += defaults.DefaultQemuAccelDarwinArm64
			} else {
				qemuOptions += defaults.DefaultQemuAccelArm64
			}
		} else {
			qemuOptions += defaults.DefaultQemuArm64
		}
		tpmDev = "tpm-tis-device"
	default:
		return fmt.Errorf("StartEVEQemu: Arch not supported: %s", qemuARCH)
	}
	if qemuSMBIOSSerial != "" {
		qemuOptions += fmt.Sprintf("-smbios type=1,serial=%s ", qemuSMBIOSSerial)
	}
	if qemuMonitorPort != 0 {
		qemuOptions += fmt.Sprintf("-monitor tcp:localhost:%d,server,nowait  ", qemuMonitorPort)
	}

	if withSDN {
		// Ports connecting SDN VM with EVE VM.
		socketPort := netDevBasePort
		for i, port := range netModel.Ports {
			qemuOptions += fmt.Sprintf("-netdev socket,id=eth%d,connect=:%d", i, socketPort)
			qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d,mac=%s ", netDev, i,
				port.EVEConnect.MAC)
			socketPort++
		}
	} else {
		// Use SLIRP networking to connect QEMU VM with the host.
		nets, err := utils.GetSubnetsNotUsed(1)
		if err != nil {
			return fmt.Errorf("StartEVEQemu: %s", err)
		}
		network := nets[0].Subnet
		var ip net.IP
		for i, port := range netModel.Ports {
			switch i {
			case 0:
				ip = nets[0].FirstAddress
			case 1:
				ip = nets[0].SecondAddress
			default:
				return fmt.Errorf("unexpected number of ports (in non-SDN mode): %d",
					len(netModel.Ports))
			}
			qemuOptions += fmt.Sprintf("-netdev user,id=eth%d,net=%s,dhcpstart=%s,ipv6=off",
				i, network, ip)
			for k, v := range qemuHostFwd {
				origPort, err := strconv.Atoi(k)
				if err != nil {
					log.Errorf("Failed converting %s to Integer", k)
					break
				}
				newPort, err := strconv.Atoi(v)
				if err != nil {
					log.Errorf("Failed converting %s to Integer", v)
					break
				}
				qemuOptions += fmt.Sprintf(",hostfwd=tcp::%d-:%d", origPort+(i*10), newPort+(i*10))
			}
			qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d,mac=%s ", netDev, i,
				port.EVEConnect.MAC)
		}
	}

	if tapInterface != "" {
		tapIdx := len(netModel.Ports)
		qemuOptions += fmt.Sprintf("-netdev tap,id=eth%d,ifname=%s", tapIdx, tapInterface)
		qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d ", netDev, tapIdx)
	}

	if swtpm {
		tpmSocket := filepath.Join(filepath.Dir(eveImageFile), "swtpm", defaults.DefaultSwtpmSockFile)
		qemuOptions += fmt.Sprintf("-chardev socket,id=chrtpm,path=%s -tpmdev emulator,id=tpm0,chardev=chrtpm -device %s,tpmdev=tpm0 ", tpmSocket, tpmDev)
	}
	if can {
		qemuOptions += "-object can-bus,id=canbus0 -device kvaser_pci,canbus=canbus0 "
	}
	if serialPCI {
		qemuOptions += "-chardev socket,id=serial_backend1,port=4444,host=0.0.0.0,server=on,wait=off " +
			"-chardev socket,id=serial_backend2,port=4445,host=0.0.0.0,server=on,wait=off " +
			"-device pci-serial-2x,chardev1=serial_backend1,chardev2=serial_backend2 "
	}
	if qemuOS == "" {
		qemuOS = runtime.GOOS
	} else {
		qemuOS = strings.ToLower(qemuOS)
	}
	if qemuOS != "linux" && qemuOS != "darwin" {
		return fmt.Errorf("StartEVEQemu: OS not supported: %s", qemuOS)
	}
	qemuOptions += "-watchdog-action reset "

	if isInstaller {
		// Run EVE installer, then start EVE VM again but without the installer image.
		consoleOpts := "-serial stdio "
		installerOptions := consoleOpts + qemuOptions
		installerOptions += fmt.Sprintf("-drive file=%s,format=%s ",
			eveImageFile, imageFormat)
		if qemuConfigFile != "" {
			installerOptions += fmt.Sprintf("-readconfig %s ", qemuConfigFile)
		}
		log.Infof("Start EVE installer: %s %s", qemuCommand, installerOptions)
		if err := utils.RunCommandForeground(qemuCommand, strings.Fields(installerOptions)...); err != nil {
			return fmt.Errorf("StartEVEQemu: %s", err)
		}
		// TODO: create a file in dist to mark EVE as installed to avoid running installer on restart
		// (with "eden eve stop && eden eve start)
	}

	consoleOps := "-display none "
	consoleOps += fmt.Sprintf("-serial chardev:char0 -chardev socket,id=char0,port=%d,"+
		"host=localhost,server,nodelay,nowait,telnet,logappend=on,logfile=%s ",
		eveTelnetPort, logFile)
	qemuOptions = consoleOps + qemuOptions
	if !isInstaller {
		qemuOptions += fmt.Sprintf("-drive file=%s,format=%s ", eveImageFile, imageFormat)
	}
	if usbImagePath != "" {
		qemuOptions += fmt.Sprintf("-drive format=raw,file=%s ", usbImagePath)
	}

	// keep readconfig after -drive as we locate additional disks in qemuConfigFile
	if qemuConfigFile != "" {
		qemuOptions += fmt.Sprintf("-readconfig %s ", qemuConfigFile)
	}

	context, err := utils.ContextLoad()
	if err != nil {
		return fmt.Errorf("StartEVEQemu: load context error: %w", err)
	}

	qmpSockFile := fmt.Sprintf("%s-qmp.sock", strings.ToLower(context.Current))
	qmpLogFile := fmt.Sprintf("%s-qmp.log", strings.ToLower(context.Current))

	qmpSockFile = filepath.Join(filepath.Dir(pidFile), qmpSockFile)
	qmpLogFile = filepath.Join(filepath.Dir(pidFile), qmpLogFile)

	// QMP sock
	qemuOptions += fmt.Sprintf("-qmp unix:%s,server,wait=off", qmpSockFile)

	// Pre-flight: probe every TCP port QEMU is about to listen on. A
	// stranded process holding any of them (most commonly the EVE
	// console port from a leaked prior-run QEMU) makes the QEMU bind
	// fail several seconds in, after which "eden eve onboard" waits
	// out the full onboarding timeout on a VM that never started. The
	// probe surfaces the real cause immediately, with the offending
	// port and a hint for finding the owner.
	portChecks := []tcpPortReservation{
		{host: "localhost", port: eveTelnetPort, role: "EVE console (telnet)"},
	}
	if qemuMonitorPort != 0 {
		portChecks = append(portChecks, tcpPortReservation{
			host: "localhost", port: qemuMonitorPort, role: "QEMU monitor",
		})
	}
	if !withSDN {
		for i := range netModel.Ports {
			for k, v := range qemuHostFwd {
				origPort, err1 := strconv.Atoi(k)
				newPort, err2 := strconv.Atoi(v)
				if err1 != nil || err2 != nil {
					continue
				}
				portChecks = append(portChecks, tcpPortReservation{
					host: "0.0.0.0",
					port: origPort + i*10,
					role: fmt.Sprintf("hostfwd %d->guest:%d",
						origPort+i*10, newPort+i*10),
				})
			}
		}
	}
	if serialPCI {
		portChecks = append(portChecks,
			tcpPortReservation{host: "0.0.0.0", port: 4444, role: "serial PCI 1"},
			tcpPortReservation{host: "0.0.0.0", port: 4445, role: "serial PCI 2"},
		)
	}
	for _, c := range portChecks {
		if err := checkPortFree(c.host, c.port, c.role); err != nil {
			return fmt.Errorf("StartEVEQemu: %w", err)
		}
	}

	log.Infof("Start EVE: %s %s", qemuCommand, qemuOptions)
	if foreground {
		if err := utils.RunCommandForeground(qemuCommand, strings.Fields(qemuOptions)...); err != nil {
			return fmt.Errorf("StartEVEQemu: %s", err)
		}
	} else {
		log.Infof("With pid: %s ; log: %s", pidFile, logFile)
		if err := utils.RunCommandNohup(qemuCommand, logFile, pidFile, strings.Fields(qemuOptions)...); err != nil {
			return fmt.Errorf("StartEVEQemu: %s", err)
		}
		err = startQMPLogger(qmpSockFile, qmpLogFile)
		if err != nil {
			// Not critical, so just print and continue
			log.Errorf("%v", err)
		}
	}
	return nil
}

// StopEVEQemu function stop EVE
func StopEVEQemu(pidFile string) (err error) {
	return utils.StopCommandWithPid(pidFile)
}

// StatusEVEQemu function get status of EVE
func StatusEVEQemu(pidFile string) (status string, err error) {
	return utils.StatusCommandWithPid(pidFile)
}

// SetLinkStateQemu changes the link state of the given interface.
func SetLinkStateQemu(qemuMonitorPort int, ifName string, up bool) error {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", qemuMonitorPort))
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	}
	linkState := "on"
	if !up {
		linkState = "off"
	}
	cmd := fmt.Sprintf("set_link %s %s", ifName, linkState)
	_, err = conn.Write([]byte(cmd + "\n"))
	if err == nil {
		err = conn.CloseWrite()
	}
	if err != nil {
		return fmt.Errorf("failed to send '%s' command to qemu: %v", cmd, err)
	}
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		// read output from the QEMU monitor command prompt
		line := scanner.Text()
		if strings.HasPrefix(line, "QEMU") || strings.HasPrefix(line, "(qemu)") {
			continue
		}
		// anything else must be an error message
		return errors.New(line)
	}
	if scanner.Err() != nil {
		return fmt.Errorf("failed to read response from QEMU monitor: %v", scanner.Err())
	}
	return nil
}

// GetLinkStatesQemu returns link states for the given set of EVE interfaces.
func GetLinkStatesQemu(qemuMonitorPort int, ifNames []string) (linkStates []edensdn.LinkState, err error) {
	// Unfortunately QEMU Monitor doesn't provide command to obtain
	// the current link state of interfaces.
	// All we can do is to traverse through the command history,
	// find the last invocation of set_link command for every interface and assume
	// that it succeeded.
	var linkStateMap = make(map[string]bool)
	for _, ifName := range ifNames {
		// initial state
		linkStateMap[ifName] = true
	}
	tcpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", qemuMonitorPort))
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	cmd := "info history"
	_, err = conn.Write([]byte(cmd + "\n"))
	if err == nil {
		err = conn.CloseWrite()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to send '%s' command to qemu: %v", cmd, err)
	}
	scanner := bufio.NewScanner(conn)
	setLinkCmdReg := regexp.MustCompile(`'set_link (\S+) (on|off)'`)
	for scanner.Scan() {
		// read output from the QEMU monitor command prompt
		line := scanner.Text()
		match := setLinkCmdReg.FindStringSubmatch(line)
		if len(match) == 3 {
			nicName := match[1]
			isUp := match[2] == "on"
			if _, knownNic := linkStateMap[nicName]; knownNic {
				linkStateMap[nicName] = isUp
			}
		}
	}
	if scanner.Err() != nil {
		return nil, fmt.Errorf("failed to read response from QEMU monitor: %v", scanner.Err())
	}
	for nicName, isUP := range linkStateMap {
		linkStates = append(linkStates, edensdn.LinkState{EveIfName: nicName, IsUP: isUP})
	}
	return linkStates, nil
}
