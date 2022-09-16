package edensdn

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	model "github.com/lf-edge/eden/sdn/api"
	log "github.com/sirupsen/logrus"
)

// SdnVMQemuRunner implements Eden-SDN VM runner using QEMU.
type SdnVMQemuRunner struct {
	SdnVMConfig
}

// NewSdnVMQemuRunner is constructor for SdnVMQemuRunner.
func NewSdnVMQemuRunner(config SdnVMConfig) *SdnVMQemuRunner {
	return &SdnVMQemuRunner{SdnVMConfig: config}
}

// Start Eden-SDN VM using QEMU.
func (vm *SdnVMQemuRunner) Start() error {
	var qemuCommand string
	qemuOptions := "-display none -nodefaults -no-user-config "
	qemuOptions += fmt.Sprintf("-serial chardev:char0 -chardev socket,id=char0,port=%d,"+
		"host=localhost,server,nodelay,nowait,telnet,logfile=%s ",
		vm.TelnetPort, vm.ConsoleLogFile)
	netDev := "e1000"
	hostOS := strings.ToLower(vm.HostOS)
	if hostOS == "" {
		hostOS = runtime.GOOS
	}
	if hostOS != "linux" && hostOS != "darwin" {
		return fmt.Errorf("host OS not supported for SDN VM: %s", hostOS)
	}
	qemuArch := strings.ToLower(vm.Architecture)
	if qemuArch == "" {
		qemuArch = runtime.GOARCH
	}
	switch qemuArch {
	case "amd64":
		qemuCommand = "qemu-system-x86_64"
		if vm.Acceleration {
			if hostOS == "darwin" {
				qemuOptions += defaults.DefaultQemuAccelDarwin
			} else {
				qemuOptions += defaults.DefaultQemuAccelLinuxAmd64
			}
		} else {
			qemuOptions += defaults.DefaultQemuAmd64
		}
	case "arm64":
		qemuCommand = "qemu-system-aarch64"
		if vm.Acceleration {
			qemuOptions += defaults.DefaultQemuAccelArm64
		} else {
			qemuOptions += defaults.DefaultQemuArm64
		}
		netDev = "virtio-net-pci"
	default:
		return fmt.Errorf("architecture not supported for SDN VM: %s", qemuArch)
	}

	// Ports connecting SDN VM with EVE VM.
	socketPort := vm.NetDevBasePort
	for i, port := range vm.NetModel.Ports {
		qemuOptions += fmt.Sprintf("-netdev socket,id=eth%d,listen=:%d", i, socketPort)
		qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d,mac=%s ", netDev, i, port.MAC)
		socketPort++
	}

	// Management port.
	qemuOptions += fmt.Sprintf("-netdev user,id=eth%d,net=%s,dhcpstart=%s,ipv6=off,"+
		"hostfwd=tcp::%d-:22,hostfwd=tcp::%d-:6666", len(vm.NetModel.Ports), vm.MgmtSubnet.String(),
		vm.MgmtSubnet.DHCPStart.String(), vm.SSHPort, vm.MgmtPort)
	qemuOptions += fmt.Sprintf(" -device %s,netdev=eth%d,mac=%s ", netDev,
		len(vm.NetModel.Ports), GenerateSdnMgmtMAC())
	_ = os.Chmod(vm.SSHKeyPath, 0600)

	// Image
	qemuOptions += fmt.Sprintf("-drive file=%s,format=qcow2 ", vm.ImagePath)

	// Watchdog
	qemuOptions += "-watchdog-action reset "

	// QEMU config
	qemuConfigPath := filepath.Join(vm.ConfigDir, "qemu.conf")
	settings := utils.QemuSettings{
		Firmware: vm.Firmware,
		MemoryMB: vm.RAM,
		CPUs:     vm.CPU,
	}
	conf, err := settings.GenerateQemuConfig()
	if err != nil {
		fmt.Errorf("failed to generate QEMU config: %v", err)
	}
	err = os.WriteFile(qemuConfigPath, conf, 0664)
	if err != nil {
		return fmt.Errorf("failed to write QEMU config file: %v", err)
	}
	qemuOptions += fmt.Sprintf("-readconfig %s ", qemuConfigPath)
	log.Infof("Start SDN: %s %s", qemuCommand, qemuOptions)
	log.Infof("With pid: %s ; console log: %s", vm.PidFile, vm.ConsoleLogFile)
	return utils.RunCommandNohup(qemuCommand, vm.ConsoleLogFile, vm.PidFile,
		strings.Fields(qemuOptions)...)
}

// Stop Eden-SDN VM running in QEMU.
func (vm *SdnVMQemuRunner) Stop() (err error) {
	if err = utils.StopCommandWithPid(vm.PidFile); err != nil {
		err = fmt.Errorf("failed to stop SDN: %v", err)
	}
	return nil
}

// RequiresVmRestart returns true if the set of ports has changed.
func (vm *SdnVMQemuRunner) RequiresVmRestart(oldModel, newModel model.NetworkModel) bool {
	for _, oldPort := range oldModel.Ports {
		newPort := newModel.GetPortByMAC(oldPort.MAC)
		if newPort == nil || oldPort.EVEConnect != newPort.EVEConnect {
			return true
		}
	}
	for _, newPort := range newModel.Ports {
		oldPort := oldModel.GetPortByMAC(newPort.MAC)
		if oldPort == nil || oldPort.EVEConnect != newPort.EVEConnect {
			return true
		}
	}
	return false
}
