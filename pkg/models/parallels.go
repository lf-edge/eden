package models

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve-api/go/config"
	log "github.com/sirupsen/logrus"
)

// devModelTypeParallels is model type for parallels
const devModelTypeParallels devModelType = defaults.DefaultParallelsModel

// DevModelParallels is dev model fields
type DevModelParallels struct {
	//physicalIOs is PhysicalIO slice for DevModel
	physicalIOs []*config.PhysicalIO
	//networks is NetworkConfig slice for DevModel
	networks []*config.NetworkConfig
	//adapters is SystemAdapter slice for DevModel
	adapters           []*config.SystemAdapter
	vlanAdapters       []*config.VlanAdapter
	bondAdapters       []*config.BondAdapter
	adapterForSwitches []string
}

// SetWiFiParams not implemented for parallels
func (ctx *DevModelParallels) SetWiFiParams(_ string, _ string) {
	log.Warning("not implemented for parallels")
}

// GetPortConfig not implemented for parallels
func (ctx *DevModelParallels) GetPortConfig(_ string, _ string) string {
	return ""
}

// DiskReadyMessage ready message
func (ctx *DevModelParallels) DiskReadyMessage() string {
	return "Upload %s to parallels and run"
}

// Config returns map with config overwrites
func (ctx *DevModelParallels) Config() map[string]interface{} {
	cfg := make(map[string]interface{})
	cfg["eve.serial"] = "*"
	cfg["eve.devmodel"] = ctx.DevModelType()
	return cfg
}

// DiskFormat to use for build image
func (ctx *DevModelParallels) DiskFormat() string {
	return "parallels"
}

// Adapters returns adapters of devModel
func (ctx *DevModelParallels) Adapters() []*config.SystemAdapter {
	return ctx.adapters
}

// SetAdapters sets systems adapters of devModel
func (ctx *DevModelParallels) SetAdapters(adapters []*config.SystemAdapter) {
	ctx.adapters = adapters
}

// Networks returns networks of devModel
func (ctx *DevModelParallels) Networks() []*config.NetworkConfig {
	return ctx.networks
}

// SetNetworks sets networks of devModel
func (ctx *DevModelParallels) SetNetworks(networks []*config.NetworkConfig) {
	ctx.networks = networks
}

// PhysicalIOs returns physicalIOs of devModel
func (ctx *DevModelParallels) PhysicalIOs() []*config.PhysicalIO {
	return ctx.physicalIOs
}

// SetPhysicalIOs sets physicalIOs of devModel
func (ctx *DevModelParallels) SetPhysicalIOs(physicalIOs []*config.PhysicalIO) {
	ctx.physicalIOs = physicalIOs
}

// VlanAdapters returns Vlan adapters of devModel
func (ctx *DevModelParallels) VlanAdapters() []*config.VlanAdapter {
	return ctx.vlanAdapters
}

// SetVlanAdapters sets Vlan adapters of devModel
func (ctx *DevModelParallels) SetVlanAdapters(vlans []*config.VlanAdapter) {
	ctx.vlanAdapters = vlans
}

// BondAdapters returns Bond adapters of devModel
func (ctx *DevModelParallels) BondAdapters() []*config.BondAdapter {
	return ctx.bondAdapters
}

// SetBondAdapters sets Bond adapters of devModel
func (ctx *DevModelParallels) SetBondAdapters(bonds []*config.BondAdapter) {
	ctx.bondAdapters = bonds
}

// AdapterForSwitches returns adapterForSwitches of devModel
func (ctx *DevModelParallels) AdapterForSwitches() []string {
	return nil
}

// DevModelType returns devModelType of devModel
func (ctx *DevModelParallels) DevModelType() string {
	return string(devModelTypeParallels)
}

func createParallels() (DevModel, error) {
	return &DevModelParallels{
		physicalIOs:        generatePhysicalIOs(2, 0, 4),
		networks:           generateNetworkConfigs(2, 0),
		adapters:           generateSystemAdapters(2, 0),
		adapterForSwitches: []string{"eth1"},
	}, nil
}
