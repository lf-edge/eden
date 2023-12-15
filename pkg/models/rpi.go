package models

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve-api/go/config"
	"github.com/lf-edge/eve-api/go/evecommon"
	log "github.com/sirupsen/logrus"
)

// devModelTypeRaspberry is model type for rpi
const devModelTypeRaspberry devModelType = defaults.DefaultRPIModel

// DevModelRpi is dev model fields
type DevModelRpi struct {
	//physicalIOs is PhysicalIO slice for DevModel
	physicalIOs []*config.PhysicalIO
	//networks is NetworkConfig slice for DevModel
	networks []*config.NetworkConfig
	//adapters is SystemAdapter slice for DevModel
	adapters     []*config.SystemAdapter
	vlanAdapters []*config.VlanAdapter
	bondAdapters []*config.BondAdapter
}

// Config returns map with config overwrites
func (ctx *DevModelRpi) Config() map[string]interface{} {
	cfg := make(map[string]interface{})
	cfg["eve.serial"] = "*"
	cfg["eve.remote"] = true
	cfg["eve.remote-addr"] = ""
	cfg["eve.arch"] = "arm64"
	cfg["eve.hostfwd"] = map[string]string{}
	cfg["eve.devmodel"] = ctx.DevModelType()
	return cfg
}

// DiskReadyMessage to show when image is ready
func (ctx *DevModelRpi) DiskReadyMessage() string {
	return "Write file %s to sd (it is in raw format)"
}

// DiskFormat to use for build image
func (ctx *DevModelRpi) DiskFormat() string {
	return "raw"
}

// GetPortConfig returns PortConfig overwrite
func (ctx *DevModelRpi) GetPortConfig(ssid string, psk string) string {
	return fmt.Sprintf(`{
	"Version": 1,
	"Ports": [{
			"Dhcp": 4,
			"Free": false,
			"IfName": "eth0",
			"Name": "Management1",
			"IsMgmt": false
		},
		{
			"Dhcp": 4,
			"Free": true,
			"IfName": "wlan0",
			"Name": "Management",
			"IsMgmt": true,
			"WirelessCfg": {
				"WType": 2,
				"Wifi": [{
					"KeyScheme": 1,
					"SSID": "%s",
					"Password": "%s"
				}]
			}
		}
	]
}`, ssid, psk)
}

// SetWiFiParams set ssid and psk for RPI
func (ctx *DevModelRpi) SetWiFiParams(ssid string, psk string) {
	if ssid != "" {
		log.Debugf("will set params for ssid %s", ssid)
		for _, el := range ctx.networks {
			if el.Wireless != nil {
				el.Wireless.WifiCfg = []*config.WifiConfig{{
					WifiSSID:  ssid,
					KeyScheme: config.WiFiKeyScheme_WPAPSK,
					Password:  psk,
				}}
			}
		}
		for _, el := range ctx.physicalIOs {
			switch el.Ptype {
			case evecommon.PhyIoType_PhyIoNetEth:
				el.Usage = evecommon.PhyIoMemberUsage_PhyIoUsageDisabled
				el.UsagePolicy = &config.PhyIOUsagePolicy{}
				for _, adapter := range ctx.adapters {
					if adapter.Name == el.Phylabel {
						adapter.Uplink = false
						break
					}
				}
			case evecommon.PhyIoType_PhyIoNetWLAN:
				el.Usage = evecommon.PhyIoMemberUsage_PhyIoUsageMgmtAndApps
				el.UsagePolicy = &config.PhyIOUsagePolicy{}
				for _, adapter := range ctx.adapters {
					if adapter.Name == el.Phylabel {
						adapter.Uplink = true
						break
					}
				}
			}
		}
	}
}

// Adapters returns adapters of devModel
func (ctx *DevModelRpi) Adapters() []*config.SystemAdapter {
	return ctx.adapters
}

// SetAdapters sets systems adapters of devModel
func (ctx *DevModelRpi) SetAdapters(adapters []*config.SystemAdapter) {
	ctx.adapters = adapters
}

// Networks returns networks of devModel
func (ctx *DevModelRpi) Networks() []*config.NetworkConfig {
	return ctx.networks
}

// SetNetworks sets networks of devModel
func (ctx *DevModelRpi) SetNetworks(networks []*config.NetworkConfig) {
	ctx.networks = networks
}

// PhysicalIOs returns physicalIOs of devModel
func (ctx *DevModelRpi) PhysicalIOs() []*config.PhysicalIO {
	return ctx.physicalIOs
}

// SetPhysicalIOs sets physicalIOs of devModel
func (ctx *DevModelRpi) SetPhysicalIOs(physicalIOs []*config.PhysicalIO) {
	ctx.physicalIOs = physicalIOs
}

// VlanAdapters returns Vlan adapters of devModel
func (ctx *DevModelRpi) VlanAdapters() []*config.VlanAdapter {
	return ctx.vlanAdapters
}

// SetVlanAdapters sets Vlan adapters of devModel
func (ctx *DevModelRpi) SetVlanAdapters(vlans []*config.VlanAdapter) {
	ctx.vlanAdapters = vlans
}

// BondAdapters returns Bond adapters of devModel
func (ctx *DevModelRpi) BondAdapters() []*config.BondAdapter {
	return ctx.bondAdapters
}

// SetBondAdapters sets Bond adapters of devModel
func (ctx *DevModelRpi) SetBondAdapters(bonds []*config.BondAdapter) {
	ctx.bondAdapters = bonds
}

// AdapterForSwitches returns adapterForSwitches of devModel
func (ctx *DevModelRpi) AdapterForSwitches() []string {
	return nil
}

// DevModelType returns devModelType of devModel
func (ctx *DevModelRpi) DevModelType() string {
	return string(devModelTypeRaspberry)
}

func createRpi() (DevModel, error) {
	return &DevModelRpi{
		physicalIOs: generatePhysicalIOs(1, 1, 0),
		networks:    generateNetworkConfigs(1, 1),
		adapters:    generateSystemAdapters(1, 1),
	}, nil
}
