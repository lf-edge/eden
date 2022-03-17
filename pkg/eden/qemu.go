package eden

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

//StartEVEQemu function run EVE in qemu
func StartEVEQemu(qemuARCH, qemuOS, eveImageFile, qemuSMBIOSSerial string, eveTelnetPort,
	qemuMonitorPort, qemuNetdevSocketPort int, qemuHostFwd map[string]string, qemuAccel bool,
	qemuConfigFile, logFile, pidFile string, tapInterface string, ethLoops int, foreground bool) (err error) {
	qemuCommand := ""
	qemuOptions := "-display none -nodefaults -no-user-config "
	qemuOptions += fmt.Sprintf("-serial chardev:char0 -chardev socket,id=char0,port=%d,host=localhost,server,nodelay,nowait,telnet,logfile=%s ", eveTelnetPort, logFile)
	netDev := "e1000"
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
			qemuOptions += defaults.DefaultQemulAmd64
		}
	case "arm64":
		qemuCommand = "qemu-system-aarch64"
		if qemuAccel {
			qemuOptions += defaults.DefaultQemuAccelArm64
		} else {
			qemuOptions += defaults.DefaultQemulArm64
		}
		netDev = "virtio-net-pci"
	default:
		return fmt.Errorf("StartEVEQemu: Arch not supported: %s", qemuARCH)
	}
	if qemuSMBIOSSerial != "" {
		qemuOptions += fmt.Sprintf("-smbios type=1,serial=%s ", qemuSMBIOSSerial)
	}
	if qemuMonitorPort != 0 {
		qemuOptions += fmt.Sprintf("-monitor tcp:localhost:%d,server,nowait  ", qemuMonitorPort)
	}
	nets, err := utils.GetSubnetsNotUsed(1)
	if err != nil {
		return fmt.Errorf("StartEVEQemu: %s", err)
	}
	offset := 0
	network := nets[0].Subnet
	var ethIndex int
	qemuOptions += fmt.Sprintf("-netdev user,id=eth%d,net=%s,dhcpstart=%s,ipv6=off",
		ethIndex, network, nets[0].FirstAddress)
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
		qemuOptions += fmt.Sprintf(",hostfwd=tcp::%d-:%d", origPort+offset, newPort+offset)
	}
	qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d ", netDev, ethIndex)
	offset += 10
	ethIndex++

	qemuOptions += fmt.Sprintf("-netdev user,id=eth%d,net=%s,dhcpstart=%s,ipv6=off",
		ethIndex, network, nets[0].SecondAddress)
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
		qemuOptions += fmt.Sprintf(",hostfwd=tcp::%d-:%d", origPort+offset, newPort+offset)
	}
	qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d ", netDev, ethIndex)
	ethIndex++

	if tapInterface != "" {
		qemuOptions += fmt.Sprintf("-netdev tap,id=eth%d,ifname=%s", ethIndex, tapInterface)
		qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d ", netDev, ethIndex)
		ethIndex++
	}

	for i := 0; i < ethLoops; i++ {
		qemuOptions += fmt.Sprintf("-netdev socket,id=eth%d,listen=:%d",
			ethIndex, qemuNetdevSocketPort)
		qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d ", netDev, ethIndex)
		ethIndex++
		qemuOptions += fmt.Sprintf("-netdev socket,id=eth%d,connect=:%d",
			ethIndex, qemuNetdevSocketPort)
		qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d ", netDev, ethIndex)
		ethIndex++
		qemuNetdevSocketPort++
	}

	if qemuOS == "" {
		qemuOS = runtime.GOOS
	} else {
		qemuOS = strings.ToLower(qemuOS)
	}
	if qemuOS != "linux" && qemuOS != "darwin" {
		return fmt.Errorf("StartEVEQemu: OS not supported: %s", qemuOS)
	}
	qemuOptions += fmt.Sprintf("-drive file=%s,format=qcow2 ", eveImageFile)
	qemuOptions += "-watchdog-action reset "
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
//If interface name is undefined, the function changes the link state of every uplink interface.
func SetLinkStateQemu(qemuMonitorPort int, ifName string, up bool) error {
	if ifName == "" {
		if err := setLinkStateQemu(qemuMonitorPort, "eth0", up); err != nil {
			return err
		}
		return setLinkStateQemu(qemuMonitorPort, "eth1", up)
	}
	return setLinkStateQemu(qemuMonitorPort, ifName, up)
}

func setLinkStateQemu(qemuMonitorPort int, ifName string, up bool) error {
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

//GetLinkStateQemu returns the link state of the interface.
//If interface name is undefined, link state of all interfaces is returned.
func GetLinkStateQemu(qemuMonitorPort int, ifName string) (linkStates []LinkState, err error) {
	// Unfortunately QEMU Monitor doesn't provide command to obtain
	// the current link state of interfaces.
	// All we can do is to traverse through the command history,
	// find the last invocation of set_link command for every interface and assume
	// that it succeeded.
	var linkStateMap = map[string]bool{"eth0": true, "eth1": true} // initial state
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
		if ifName != "" && ifName != nicName {
			continue
		}
		linkStates = append(linkStates, LinkState{InterfaceName: nicName, IsUP: isUP})
	}
	return linkStates, nil
}
