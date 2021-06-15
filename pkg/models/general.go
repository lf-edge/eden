package models

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
)

//devModelTypeGeneral is model type for genera device
const devModelTypeGeneral devModelType = defaults.DefaultGeneralModel

//DevModelGeneral is dev model fields
type DevModelGeneral struct {
	//physicalIOs is PhysicalIO slice for DevModel
	physicalIOs []*config.PhysicalIO
	//networks is NetworkConfig slice for DevModel
	networks []*config.NetworkConfig
	//adapters is SystemAdapter slice for DevModel
	adapters []*config.SystemAdapter
	//adapterForSwitches is name of adapter for use in switch
	adapterForSwitches []string
}

//Config returns map with config overwrites
func (ctx *DevModelGeneral) Config() map[string]interface{} {
	cfg := make(map[string]interface{})
	cfg["eve.serial"] = "*"
	cfg["eve.remote"] = true
	cfg["eve.remote-addr"] = ""
	cfg["eve.hostfwd"] = map[string]string{}
	cfg["eve.devmodel"] = ctx.DevModelType()
	return cfg
}

//DiskReadyMessage to show when image is ready
func (ctx *DevModelGeneral) DiskReadyMessage() string {
	return "EVE image ready: %s"
}

//DiskFormat to use for build image
func (ctx *DevModelGeneral) DiskFormat() string {
	return "raw"
}

//GetPortConfig returns PortConfig overwrite
func (ctx *DevModelGeneral) GetPortConfig(_ string, _ string) string {
	return ""
}

//SetWiFiParams not implemented for Qemu
func (ctx *DevModelGeneral) SetWiFiParams(_ string, _ string) {
	log.Warning("not implemented for Qemu")
}

//Adapters returns adapters of devModel
func (ctx *DevModelGeneral) Adapters() []*config.SystemAdapter {
	return ctx.adapters
}

//Networks returns networks of devModel
func (ctx *DevModelGeneral) Networks() []*config.NetworkConfig {
	return ctx.networks
}

//PhysicalIOs returns physicalIOs of devModel
func (ctx *DevModelGeneral) PhysicalIOs() []*config.PhysicalIO {
	return ctx.physicalIOs
}

//SetPhysicalIOs sets physicalIOs of devModel
func (ctx *DevModelGeneral) SetPhysicalIOs(physicalIOs []*config.PhysicalIO){
	ctx.physicalIOs = physicalIOs
	ctx.adapters = filterSystemAdapters(ctx.adapters, ctx.physicalIOs)
}

//AdapterForSwitches returns adapterForSwitches of devModel
func (ctx *DevModelGeneral) AdapterForSwitches() []string {
	return ctx.adapterForSwitches
}

//DevModelType returns devModelType of devModel
func (ctx *DevModelGeneral) DevModelType() string {
	return string(devModelTypeGeneral)
}

func createGeneral() (DevModel, error) {
	return &DevModelGeneral{
		physicalIOs:        generatePhysicalIOs(2, 0, 0),
		networks:           generateNetworkConfigs(2, 0),
		adapters:           generateSystemAdapters(2, 0),
		adapterForSwitches: []string{"eth1"},
	}, nil
}
