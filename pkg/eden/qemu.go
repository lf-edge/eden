package eden

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	sdnapi "github.com/lf-edge/eden/sdn/api"
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
func StartEVEQemu(qemuARCH, qemuOS, eveImageFile, qemuSMBIOSSerial string,
	eveTelnetPort, qemuMonitorPort, netDevBasePort int,
	qemuAccel bool, qemuConfigFile, logFile, pidFile string,
	netModel sdnapi.NetworkModel, tapInterface string, swtpm, foreground bool) (err error) {
	qemuCommand := ""
	qemuOptions := "-display none -nodefaults -no-user-config "
	qemuOptions += fmt.Sprintf("-serial chardev:char0 -chardev socket,id=char0,port=%d,host=localhost,server,nodelay,nowait,telnet,logappend=on,logfile=%s ", eveTelnetPort, logFile)
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

	// Ports connecting SDN VM with EVE VM.
	socketPort := netDevBasePort
	for i, port := range netModel.Ports {
		qemuOptions += fmt.Sprintf("-netdev socket,id=eth%d,connect=:%d", i, socketPort)
		qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d,mac=%s ", netDev, i,
			port.EVEConnect.MAC)
		socketPort++
	}

	if tapInterface != "" {
		tapIdx := len(netModel.Ports)
		qemuOptions += fmt.Sprintf("-netdev tap,id=eth%d,ifname=%s",	tapIdx, tapInterface)
		qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d ", netDev, tapIdx)
	}

	// TODO: allow to test usb.json
	//qemuOptions += fmt.Sprintf(" -drive format=raw,file=./sdn/usb.img ")

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