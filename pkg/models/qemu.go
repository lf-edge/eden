package models

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve-api/go/config"
	log "github.com/sirupsen/logrus"
)

// devModelTypeQemu is model type for qemu
const devModelTypeQemu devModelType = defaults.DefaultQemuModel

// DevModelQemu is dev model fields
type DevModelQemu struct {
	//physicalIOs is PhysicalIO slice for DevModel
	physicalIOs []*config.PhysicalIO
	//networks is NetworkConfig slice for DevModel
	networks []*config.NetworkConfig
	//adapters is SystemAdapter slice for DevModel
	adapters     []*config.SystemAdapter
	vlanAdapters []*config.VlanAdapter
	bondAdapters []*config.BondAdapter
	//adapterForSwitches is name of adapter for use in switch
	adapterForSwitches []string
}

// Config returns map with config overwrites
func (ctx *DevModelQemu) Config() map[string]interface{} {
	return nil
}

// DiskReadyMessage to show when image is ready
func (ctx *DevModelQemu) DiskReadyMessage() string {
	return "EVE image ready: %s"
}

// DiskFormat to use for build image
func (ctx *DevModelQemu) DiskFormat() string {
	return "qcow2"
}

// GetPortConfig returns PortConfig overwrite
func (ctx *DevModelQemu) GetPortConfig(_ string, _ string) string {
	return ""
}

// SetWiFiParams not implemented for Qemu
func (ctx *DevModelQemu) SetWiFiParams(_ string, _ string) {
	log.Warning("not implemented for Qemu")
}

// Adapters returns adapters of devModel
func (ctx *DevModelQemu) Adapters() []*config.SystemAdapter {
	return ctx.adapters
}

// SetAdapters sets systems adapters of devModel
func (ctx *DevModelQemu) SetAdapters(adapters []*config.SystemAdapter) {
	ctx.adapters = adapters
}

// Networks returns networks of devModel
func (ctx *DevModelQemu) Networks() []*config.NetworkConfig {
	return ctx.networks
}

// SetNetworks sets networks of devModel
func (ctx *DevModelQemu) SetNetworks(networks []*config.NetworkConfig) {
	ctx.networks = networks
}

// PhysicalIOs returns physicalIOs of devModel
func (ctx *DevModelQemu) PhysicalIOs() []*config.PhysicalIO {
	return ctx.physicalIOs
}

// SetPhysicalIOs sets physicalIOs of devModel
func (ctx *DevModelQemu) SetPhysicalIOs(physicalIOs []*config.PhysicalIO) {
	ctx.physicalIOs = physicalIOs
}

// VlanAdapters returns Vlan adapters of devModel
func (ctx *DevModelQemu) VlanAdapters() []*config.VlanAdapter {
	return ctx.vlanAdapters
}

// SetVlanAdapters sets Vlan adapters of devModel
func (ctx *DevModelQemu) SetVlanAdapters(vlans []*config.VlanAdapter) {
	ctx.vlanAdapters = vlans
}

// BondAdapters returns Bond adapters of devModel
func (ctx *DevModelQemu) BondAdapters() []*config.BondAdapter {
	return ctx.bondAdapters
}

// SetBondAdapters sets Bond adapters of devModel
func (ctx *DevModelQemu) SetBondAdapters(bonds []*config.BondAdapter) {
	ctx.bondAdapters = bonds
}

// AdapterForSwitches returns adapterForSwitches of devModel
func (ctx *DevModelQemu) AdapterForSwitches() []string {
	return ctx.adapterForSwitches
}

// DevModelType returns devModelType of devModel
func (ctx *DevModelQemu) DevModelType() string {
	return string(devModelTypeQemu)
}

func createQemu() (DevModel, error) {
	return &DevModelQemu{
			physicalIOs:        generatePhysicalIOs(3, 0, 4),
			networks:           generateNetworkConfigs(3, 0),
			adapters:           generateSystemAdapters(3, 0),
			adapterForSwitches: []string{"eth2"}},
		nil
}
