package edensdn

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"net"

	model "github.com/lf-edge/eden/sdn/api"
)

// SdnVMRunner is implemented for every virtualization technology on which Eden-SDN
// is supported. Currently only QEMU is supported.
type SdnVMRunner interface {
	// Start Eden-SDN VM.
	Start() error
	// Stop Eden-SDN VM.
	Stop() error
	// RequiresVmRestart should return true if going from oldModel to newModel requires
	// to restart EVE VM and SDN VM.
	RequiresVmRestart(oldModel, newModel model.NetworkModel) bool
}

// SdnVMConfig : configuration for Eden-SDN VM.
type SdnVMConfig struct {
	Architecture   string
	Acceleration   bool
	HostOS         string // darwin, linux, etc.
	ImagePath      string
	ConfigDir      string
	CPU            int
	RAM            int // in MB
	Firmware       []string
	NetModel       model.NetworkModel
	TelnetPort     uint16
	SSHPort        uint16
	SSHKeyPath     string
	MgmtPort       uint16
	MgmtSubnet     SdnMgmtSubnet
	NetDevBasePort uint16 // QEMU-specific
	PidFile        string
	ConsoleLogFile string
}

// SdnMgmtSubnet : IP configuration for Eden-SDN management network.
type SdnMgmtSubnet struct {
	*net.IPNet
	DHCPStart net.IP
}

// GetSdnVMRunner returns SdnVMRunner for a given device model type.
func GetSdnVMRunner(devModelType string, config SdnVMConfig) (SdnVMRunner, error) {
	switch devModelType {
	case defaults.DefaultQemuModel:
		return NewSdnVMQemuRunner(config), nil
	}
	return nil, fmt.Errorf("not implemented for type: %s", devModelType)
}
