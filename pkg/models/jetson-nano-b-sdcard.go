package models

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
)

//devModelTypeRaspberry is model type for rpi
const devModelTypeJetsonNanoB devModelType = defaults.DefaultJetsonNanoBModel

//DevModelJetsonNanoB is dev model fields
type DevModelJetsonNanoB struct {
	//physicalIOs is PhysicalIO slice for DevModel
	physicalIOs []*config.PhysicalIO
	//networks is NetworkConfig slice for DevModel
	networks []*config.NetworkConfig
	//adapters is SystemAdapter slice for DevModel
	adapters []*config.SystemAdapter
}

//Config returns map with config overwrites
func (ctx *DevModelJetsonNanoB) Config() map[string]interface{} {
	cfg := make(map[string]interface{})
	cfg["eve.serial"] = "*"
	cfg["eve.remote"] = true
	cfg["eve.remote-addr"] = ""
	cfg["eve.arch"] = "arm64"
	cfg["eve.hostfwd"] = map[string]string{}
	cfg["eve.devmodel"] = ctx.DevModelType()
	return cfg
}

//DiskReadyMessage to show when image is ready
func (ctx *DevModelJetsonNanoB) DiskReadyMessage() string {
	return "Write file %s to sd (it is in raw format)"
}

//DiskFormat to use for build image
func (ctx *DevModelJetsonNanoB) DiskFormat() string {
	return "raw"
}

//GetPortConfig returns PortConfig overwrite
func (ctx *DevModelJetsonNanoB) GetPortConfig(_ string, _ string) string {
	return ""
}

//Adapters returns adapters of devModel
func (ctx *DevModelJetsonNanoB) Adapters() []*config.SystemAdapter {
	return ctx.adapters
}

//Networks returns networks of devModel
func (ctx *DevModelJetsonNanoB) Networks() []*config.NetworkConfig {
	return ctx.networks
}

//SetWiFiParams not implemented for jetson-nano-b
func (ctx *DevModelJetsonNanoB) SetWiFiParams(_ string, _ string) {
	log.Warning("not implemented for jetson-nano-b")
}

//PhysicalIOs returns physicalIOs of devModel
func (ctx *DevModelJetsonNanoB) PhysicalIOs() []*config.PhysicalIO {
	return ctx.physicalIOs
}

//AdapterForSwitches returns adapterForSwitches of devModel
func (ctx *DevModelJetsonNanoB) AdapterForSwitches() []string {
	return nil
}

//DevModelType returns devModelType of devModel
func (ctx *DevModelJetsonNanoB) DevModelType() string {
	return string(devModelTypeJetsonNanoB)
}

//GetFirstAdapterForSwitches return first adapter available for switch networkInstance
func (ctx *DevModelJetsonNanoB) GetFirstAdapterForSwitches() string {
	return "uplink"
}

//GetTarget not used for DevModelJetson
func (ctx *DevModelJetsonNanoB) GetTarget() string {
	return ctx.DevModelType()
}

func createJetsonNanoB() (DevModel, error) {
	return &DevModelJetsonNanoB{
		physicalIOs: generatePhysicalIOs(1, 0, 4),
		networks:    generateNetworkConfigs(1, 0),
		adapters:    generateSystemAdapters(1, 0),
	}, nil
}
