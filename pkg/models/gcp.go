package models

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
)

//devModelTypeGCP is model type for GCP
const devModelTypeGCP devModelType = defaults.DefaultGCPModel

//DevModelGCP is dev model fields
type DevModelGCP struct {
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
func (ctx *DevModelGCP) Config() map[string]interface{} {
	cfg := make(map[string]interface{})
	cfg["eve.serial"] = "*"
	cfg["eve.remote"] = true
	cfg["eve.remote-addr"] = ""
	cfg["eve.hostfwd"] = map[string]string{}
	cfg["eve.devmodel"] = ctx.DevModelType()
	return cfg
}

//DiskReadyMessage to show when image is ready
func (ctx *DevModelGCP) DiskReadyMessage() string {
	return "Upload %s to gcp and run"
}

//DiskFormat to use for build image
func (ctx *DevModelGCP) DiskFormat() string {
	return "gcp"
}

//GetPortConfig returns PortConfig overwrite
func (ctx *DevModelGCP) GetPortConfig(_ string, _ string) string {
	return ""
}

//SetWiFiParams not implemented for Qemu
func (ctx *DevModelGCP) SetWiFiParams(_ string, _ string) {
	log.Warning("not implemented for GCP")
}

//Adapters returns adapters of devModel
func (ctx *DevModelGCP) Adapters() []*config.SystemAdapter {
	return ctx.adapters
}

//Networks returns networks of devModel
func (ctx *DevModelGCP) Networks() []*config.NetworkConfig {
	return ctx.networks
}

//PhysicalIOs returns physicalIOs of devModel
func (ctx *DevModelGCP) PhysicalIOs() []*config.PhysicalIO {
	return ctx.physicalIOs
}

//AdapterForSwitches returns adapterForSwitches of devModel
func (ctx *DevModelGCP) AdapterForSwitches() []string {
	return ctx.adapterForSwitches
}

//DevModelType returns devModelType of devModel
func (ctx *DevModelGCP) DevModelType() string {
	return string(devModelTypeGCP)
}

//GetFirstAdapterForSwitches return first adapter available for switch networkInstance
func (ctx *DevModelGCP) GetFirstAdapterForSwitches() string {
	if len(ctx.adapterForSwitches) > 0 {
		return ctx.adapterForSwitches[0]
	}
	return "uplink"
}

func createGCP() (DevModel, error) {
	return &DevModelGCP{
		physicalIOs:        generatePhysicalIOs(1, 0, 0),
		networks:           generateNetworkConfigs(1, 0),
		adapters:           generateSystemAdapters(1, 0),
		adapterForSwitches: []string{},
	}, nil
}
