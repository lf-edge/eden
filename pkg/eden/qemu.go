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

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/edensdn"
	"github.com/lf-edge/eden/pkg/utils"
	sdnapi "github.com/lf-edge/eden/sdn/vm/api"
	log "github.com/sirupsen/logrus"
)

//StartSWTPM starts swtpm process and use stateDir as state, log, pid and socket location
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

//StopSWTPM stops swtpm process using pid from stateDir
func StopSWTPM(stateDir string) error {
	command := "swtpm"
	pidFile := filepath.Join(stateDir, fmt.Sprintf("%s.pid", command))
	return utils.StopCommandWithPid(pidFile)
}

//StartEVEQemu function run EVE in qemu
func StartEVEQemu(qemuARCH, qemuOS, eveImageFile, imageFormat string, isInstaller bool,
	qemuSMBIOSSerial string, eveTelnetPort, qemuMonitorPort, netDevBasePort int,
	qemuHostFwd map[string]string, qemuAccel bool, qemuConfigFile, logFile, pidFile string,
	netModel sdnapi.NetworkModel, withSDN bool, tapInterface, usbImagePath string,
	swtpm, foreground bool) (err error) {
	var qemuCommand, qemuOptions string
	qemuOptions += "-nodefaults -no-user-config "
	netDev := "e1000"
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
			}
		} else {
			qemuOptions += defaults.DefaultQemuAmd64
		}
	case "arm64":
		qemuCommand = "qemu-system-aarch64"
		if qemuAccel {
			qemuOptions += defaults.DefaultQemuAccelArm64
		} else {
			qemuOptions += defaults.DefaultQemuArm64
		}
		netDev = "virtio-net-pci"
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
	}
	return nil
}

//StopEVEQemu function stop EVE
func StopEVEQemu(pidFile string) (err error) {
	return utils.StopCommandWithPid(pidFile)
}

//StatusEVEQemu function get status of EVE
func StatusEVEQemu(pidFile string) (status string, err error) {
	return utils.StatusCommandWithPid(pidFile)
}

//SetLinkStateQemu changes the link state of the given interface.
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

//GetLinkStatesQemu returns link states for the given set of EVE interfaces.
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
