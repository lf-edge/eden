package models

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
)

//devModelTypeVBox is model type for VBox
const devModelTypeVBox devModelType = defaults.DefaultVBoxModel

//DevModelVBox is dev model fields
type DevModelVBox struct {
	//physicalIOs is PhysicalIO slice for DevModelVBox
	physicalIOs []*config.PhysicalIO
	//networks is NetworkConfig slice for DevModelVBox
	networks []*config.NetworkConfig
	//adapters is SystemAdapter slice for DevModelVBox
	adapters []*config.SystemAdapter
	//adapterForSwitches is name of adapter for use in switch
	adapterForSwitches []string
}

//Config returns map with config overwrites
func (ctx *DevModelVBox) Config() map[string]interface{} {
	cfg := make(map[string]interface{})
	cfg["eve.serial"] = "*"
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

//SetWiFiParams not implemented for VBox
func (ctx *DevModelVBox) SetWiFiParams(_ string, _ string) {
	log.Warning("not implemented for VBox")
}

//Adapters returns adapters of DevModelVBox
func (ctx *DevModelVBox) Adapters() []*config.SystemAdapter {
	return ctx.adapters
}

//Networks returns networks of DevModelVBox
func (ctx *DevModelVBox) Networks() []*config.NetworkConfig {
	return ctx.networks
}

//PhysicalIOs returns physicalIOs of DevModelVBox
func (ctx *DevModelVBox) PhysicalIOs() []*config.PhysicalIO {
	return ctx.physicalIOs
}

//SetPhysicalIOs sets physicalIOs of devModel
func (ctx *DevModelVBox) SetPhysicalIOs(physicalIOs []*config.PhysicalIO){
	ctx.physicalIOs = physicalIOs
	ctx.adapters = filterSystemAdapters(ctx.adapters, ctx.physicalIOs)
}

//AdapterForSwitches returns adapterForSwitches of DevModelVBox
func (ctx *DevModelVBox) AdapterForSwitches() []string {
	return ctx.adapterForSwitches
}

//DevModelType returns devModelType of DevModelVBox
func (ctx *DevModelVBox) DevModelType() string {
	return string(devModelTypeVBox)
}

func createVBox() (DevModel, error) {
	return &DevModelVBox{
			physicalIOs:        generatePhysicalIOs(2, 0, 4),
			networks:           generateNetworkConfigs(2, 0),
			adapters:           generateSystemAdapters(2, 0),
			adapterForSwitches: []string{"eth1"}},
		nil
}
