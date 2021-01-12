package models

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
)

//devModelTypeQemu is model type for qemu
const devModelTypeVBox devModelType = defaults.DefaultVBoxModel

//DevModelQemu is dev model fields
type DevModelVBox struct {
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
func (ctx *DevModelVBox) Config() map[string]interface{} {
	cfg := make(map[string]interface{})
	cfg["eve.devmodel"] = ctx.DevModelType()
	return cfg
}

//DiskReadyMessage to show when image is ready
func (ctx *DevModelVBox) DiskReadyMessage() string {
	return "EVE image ready: %s"
}

//DiskFormat to use for build image
func (ctx *DevModelVBox) DiskFormat() string {
	return "vdi"
}

//GetPortConfig returns PortConfig overwrite
func (ctx *DevModelVBox) GetPortConfig(_ string, _ string) string {
	return ""
}

//SetWiFiParams not implemented for Qemu
func (ctx *DevModelVBox) SetWiFiParams(_ string, _ string) {
	log.Warning("not implemented for VBox")
}

//Adapters returns adapters of devModel
func (ctx *DevModelVBox) Adapters() []*config.SystemAdapter {
	return ctx.adapters
}

//Networks returns networks of devModel
func (ctx *DevModelVBox) Networks() []*config.NetworkConfig {
	return ctx.networks
}

//PhysicalIOs returns physicalIOs of devModel
func (ctx *DevModelVBox) PhysicalIOs() []*config.PhysicalIO {
	return ctx.physicalIOs
}

//AdapterForSwitches returns adapterForSwitches of devModel
func (ctx *DevModelVBox) AdapterForSwitches() []string {
	return ctx.adapterForSwitches
}

//DevModelType returns devModelType of devModel
func (ctx *DevModelVBox) DevModelType() string {
	return string(devModelTypeVBox)
}

//GetFirstAdapterForSwitches return first adapter available for switch networkInstance
func (ctx *DevModelVBox) GetFirstAdapterForSwitches() string {
	if len(ctx.adapterForSwitches) > 0 {
		return ctx.adapterForSwitches[0]
	}
	return "uplink"
}

func createVBox() (DevModel, error) {
	return &DevModelVBox{
			physicalIOs:        generatePhysicalIOs(2, 0, 4),
			networks:           generateNetworkConfigs(2, 0),
			adapters:           generateSystemAdapters(2, 0),
			adapterForSwitches: []string{"eth1"}},
		nil
}
