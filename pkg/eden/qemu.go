package eden

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

//StartEVEQemu function run EVE in qemu
func StartEVEQemu(qemuARCH, qemuOS, eveImageFile, qemuSMBIOSSerial string, eveTelnetPort int, qemuHostFwd map[string]string, qemuAccel bool, qemuConfigFile, logFile string, pidFile string, foregroud bool) (err error) {
	qemuCommand := ""
	qemuOptions := "-display none -nodefaults -no-user-config "
	qemuOptions += fmt.Sprintf("-serial chardev:char0 -chardev socket,id=char0,port=%d,host=localhost,server,nodelay,nowait,telnet,logfile=%s ", eveTelnetPort, logFile)
	if qemuSMBIOSSerial != "" {
		qemuOptions += fmt.Sprintf("-smbios type=1,serial=%s ", qemuSMBIOSSerial)
	}
	nets, err := utils.GetSubnetsNotUsed(1)
	if err != nil {
		return fmt.Errorf("StartEVEQemu: %s", err)
	}
	offset := 0
	network := nets[0].Subnet
	qemuOptions += fmt.Sprintf("-netdev user,id=eth%d,net=%s,dhcpstart=%s", 0, network, nets[0].FirstAddress)
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
		qemuOptions += fmt.Sprintf(",hostfwd=tcp::%d-:%d", origPort + offset, newPort + offset)
	}
	qemuOptions += fmt.Sprintf(" -device virtio-net-pci,netdev=eth%d ", 0)
	offset += 10

	qemuOptions += fmt.Sprintf("-netdev user,id=eth%d,net=%s,dhcpstart=%s", 1, network, nets[0].SecondAddress)
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
		qemuOptions += fmt.Sprintf(",hostfwd=tcp::%d-:%d", origPort + offset, newPort + offset)
	}
	qemuOptions += fmt.Sprintf(" -device virtio-net-pci,netdev=eth%d ", 1)

	if qemuOS == "" {
		qemuOS = runtime.GOOS
	} else {
		qemuOS = strings.ToLower(qemuOS)
	}
	if qemuOS != "linux" && qemuOS != "darwin" {
		return fmt.Errorf("StartEVEQemu: OS not supported: %s", qemuOS)
	}
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
			qemuOptions += "--cpu SandyBridge "
		}
	case "arm64":
		qemuCommand = "qemu-system-aarch64"
		if qemuAccel {
			qemuOptions += defaults.DefaultQemuAccelLinuxArm64
		}
	default:
		return fmt.Errorf("StartEVEQemu: Arch not supported: %s", qemuARCH)
	}
	qemuOptions += fmt.Sprintf("-drive file=%s,format=qcow2 ", eveImageFile)
	if qemuConfigFile != "" {
		qemuOptions += fmt.Sprintf("-readconfig %s ", qemuConfigFile)
	}
	log.Infof("Start EVE: %s %s", qemuCommand, qemuOptions)
	if foregroud {
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
