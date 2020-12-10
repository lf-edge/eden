package controller

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/evecommon"
	log "github.com/sirupsen/logrus"
)

//DevModelType is type of dev model
type DevModelType string

//DevModel is dev model fields
type DevModel struct {
	//physicalIOs is PhysicalIO slice for DevModel
	physicalIOs []*config.PhysicalIO
	//networks is NetworkConfig slice for DevModel
	networks []*config.NetworkConfig
	//adapters is SystemAdapter slice for DevModel
	adapters []*config.SystemAdapter
	//adapterForSwitches is name of adapter for use in switch
	adapterForSwitches []string
	devModelType       DevModelType
}

//GetFirstAdapterForSwitches return first adapter available for switch networkInstance
func (ctx *DevModel) GetFirstAdapterForSwitches() string {
	if len(ctx.adapterForSwitches) > 0 {
		return ctx.adapterForSwitches[0]
	}
	return "uplink"
}

//GetNetDHCPID return netDHCPID id
func (ctx *DevModel) GetNetDHCPID() string {
	return defaults.NetDHCPID
}

//GetNetNoDHCPID return netNoDHCPID id
func (ctx *DevModel) GetNetNoDHCPID() string {
	return defaults.NetNoDHCPID
}

//SetWiFiParams set ssid and psk for RPI
func (ctx *DevModel) SetWiFiParams(ssid string, psk string) {
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
				el.UsagePolicy = &config.PhyIOUsagePolicy{
					FreeUplink: false,
				}
				for _, adapter := range ctx.adapters {
					if adapter.Name == el.Phylabel {
						adapter.Uplink = false
						break
					}
				}
			case evecommon.PhyIoType_PhyIoNetWLAN:
				el.Usage = evecommon.PhyIoMemberUsage_PhyIoUsageMgmtAndApps
				el.UsagePolicy = &config.PhyIOUsagePolicy{
					FreeUplink: true,
				}
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

//GetPortConfig returns PortConfig overwrite
func GetPortConfig(devModel string, ssid string, psk string) string {
	switch devModel {
	case defaults.DefaultRPIModel:
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
	default:
		return ""
	}
}

//DevModelTypeEmpty is empty model type
const DevModelTypeEmpty DevModelType = "Empty"

//DevModelTypeQemu is model type for qemu
const DevModelTypeQemu DevModelType = defaults.DefaultQemuModel

//DevModelTypeRaspberry is model type for Raspberry
const DevModelTypeRaspberry DevModelType = defaults.DefaultRPIModel

//DevModelTypeGCP is model type for GCP
const DevModelTypeGCP DevModelType = defaults.DefaultGCPModel

//DevModelTypeGeneral is model type for general EVE device
const DevModelTypeGeneral DevModelType = defaults.DefaultGeneralModel

//CreateDevModel create manual DevModel with provided params
func (cloud *CloudCtx) CreateDevModel(PhysicalIOs []*config.PhysicalIO, Networks []*config.NetworkConfig, Adapters []*config.SystemAdapter, AdapterForSwitches []string, modelType DevModelType) *DevModel {
	devModel := &DevModel{adapterForSwitches: AdapterForSwitches, physicalIOs: PhysicalIOs, networks: Networks, adapters: Adapters, devModelType: modelType}
	if cloud.devModels == nil {
		cloud.devModels = make(map[DevModelType]*DevModel)
	}
	cloud.devModels[modelType] = devModel
	return devModel
}

//GetDevModelByName return DevModel object by DevModelType string
func (cloud *CloudCtx) GetDevModelByName(devModelType string) (*DevModel, error) {
	return cloud.GetDevModel(DevModelType(devModelType))
}

//GetDevModel return DevModel object by DevModelType
func (cloud *CloudCtx) GetDevModel(devModelType DevModelType) (*DevModel, error) {
	switch devModelType {
	case DevModelTypeEmpty:
		return cloud.CreateDevModel(nil, nil, nil, nil, DevModelTypeEmpty), nil
	case DevModelTypeQemu, DevModelTypeGCP, DevModelTypeGeneral:
		return cloud.CreateDevModel(
				[]*config.PhysicalIO{{
					Ptype:        evecommon.PhyIoType_PhyIoNetEth,
					Phylabel:     "eth0",
					Logicallabel: "eth0",
					Assigngrp:    "eth0",
					Phyaddrs:     map[string]string{"Ifname": "eth0"},
					Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageMgmtAndApps,
					UsagePolicy: &config.PhyIOUsagePolicy{
						FreeUplink: true,
					},
				}, {
					Ptype:        evecommon.PhyIoType_PhyIoNetEth,
					Phylabel:     "eth1",
					Logicallabel: "eth1",
					Assigngrp:    "eth1",
					Phyaddrs:     map[string]string{"Ifname": "eth1"},
					Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageShared,
					UsagePolicy: &config.PhyIOUsagePolicy{
						FreeUplink: false,
					},
				}, {
					Ptype:        evecommon.PhyIoType_PhyIoUSB,
					Phylabel:     "USB1:1",
					Logicallabel: "USB1:1",
					Assigngrp:    "USB0",
					Phyaddrs:     map[string]string{"UsbAddr": "1:1"},
					Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDedicated,
				},
					{
						Ptype:        evecommon.PhyIoType_PhyIoUSB,
						Phylabel:     "USB2:1",
						Logicallabel: "USB2:1",
						Assigngrp:    "USB1",
						Phyaddrs:     map[string]string{"UsbAddr": "2:1"},
						Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDedicated,
					},
					{
						Ptype:        evecommon.PhyIoType_PhyIoUSB,
						Phylabel:     "USB3:1",
						Logicallabel: "USB3:1",
						Assigngrp:    "USB2",
						Phyaddrs:     map[string]string{"UsbAddr": "3:1"},
						Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDedicated,
					},
					{
						Ptype:        evecommon.PhyIoType_PhyIoUSB,
						Phylabel:     "USB3:2",
						Logicallabel: "USB3:2",
						Assigngrp:    "USB3",
						Phyaddrs:     map[string]string{"UsbAddr": "3:2"},
						Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDedicated,
					},
					{
						Ptype:        evecommon.PhyIoType_PhyIoUSB,
						Phylabel:     "USB3:3",
						Logicallabel: "USB3:3",
						Assigngrp:    "USB4",
						Phyaddrs:     map[string]string{"UsbAddr": "3:3"},
						Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDedicated,
					},
					{
						Ptype:        evecommon.PhyIoType_PhyIoUSB,
						Phylabel:     "USB3:4",
						Logicallabel: "USB3:4",
						Assigngrp:    "USB5",
						Phyaddrs:     map[string]string{"UsbAddr": "3:4"},
						Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDedicated,
					},
					{
						Ptype:        evecommon.PhyIoType_PhyIoUSB,
						Phylabel:     "USB3:5",
						Logicallabel: "USB3:5",
						Assigngrp:    "USB6",
						Phyaddrs:     map[string]string{"UsbAddr": "3:5"},
						Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDedicated,
					},
					{
						Ptype:        evecommon.PhyIoType_PhyIoUSB,
						Phylabel:     "USB3:6",
						Logicallabel: "USB3:6",
						Assigngrp:    "USB7",
						Phyaddrs:     map[string]string{"UsbAddr": "3:6"},
						Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDedicated,
					},
					{
						Ptype:        evecommon.PhyIoType_PhyIoUSB,
						Phylabel:     "USB4:1",
						Logicallabel: "USB4:1",
						Assigngrp:    "USB8",
						Phyaddrs:     map[string]string{"UsbAddr": "4:1"},
						Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDedicated,
					},
				},
				[]*config.NetworkConfig{
					{
						Id:   defaults.NetDHCPID,
						Type: config.NetworkType_V4,
						Ip: &config.Ipspec{
							Dhcp:      config.DHCPType_Client,
							DhcpRange: &config.IpRange{},
						},
						Wireless: nil,
					},
					{
						Id:   defaults.NetNoDHCPID,
						Type: config.NetworkType_V4,
						Ip: &config.Ipspec{
							Dhcp:      config.DHCPType_DHCPNone,
							DhcpRange: &config.IpRange{},
						},
						Wireless: nil,
					},
				},
				[]*config.SystemAdapter{
					{
						Name:        "eth0",
						Uplink:      true,
						NetworkUUID: defaults.NetDHCPID,
					},
					{
						Name:        "eth1",
						NetworkUUID: defaults.NetNoDHCPID,
					},
				},
				[]string{"eth1"},
				devModelType),
			nil
	case DevModelTypeRaspberry:
		return cloud.CreateDevModel(
				[]*config.PhysicalIO{{
					Ptype:        evecommon.PhyIoType_PhyIoNetEth,
					Phylabel:     "eth0",
					Logicallabel: "eth0",
					Assigngrp:    "eth0",
					Phyaddrs:     map[string]string{"Ifname": "eth0"},
					Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageMgmtAndApps,
					UsagePolicy: &config.PhyIOUsagePolicy{
						FreeUplink: true,
					},
				}, {
					Ptype:        evecommon.PhyIoType_PhyIoNetWLAN,
					Phylabel:     "wlan0",
					Logicallabel: "wlan0",
					Assigngrp:    "wlan0",
					Phyaddrs:     map[string]string{"Ifname": "wlan0"},
					Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDisabled,
					UsagePolicy: &config.PhyIOUsagePolicy{
						FreeUplink: false,
					},
				},
				},
				[]*config.NetworkConfig{
					{
						Id:   defaults.NetDHCPID,
						Type: config.NetworkType_V4,
						Ip: &config.Ipspec{
							Dhcp:      config.DHCPType_Client,
							DhcpRange: &config.IpRange{},
						},
						Wireless: nil,
					},
					{
						Id:   defaults.NetWiFiID,
						Type: config.NetworkType_V4,
						Ip: &config.Ipspec{
							Dhcp:      config.DHCPType_Client,
							DhcpRange: &config.IpRange{},
						},
						Wireless: &config.WirelessConfig{
							Type:        config.WirelessType_WiFi,
							CellularCfg: nil,
							WifiCfg:     nil,
						},
					},
				},
				[]*config.SystemAdapter{
					{
						Name:        "eth0",
						Uplink:      true,
						NetworkUUID: defaults.NetDHCPID,
					},
					{
						Name:        "wlan0",
						NetworkUUID: defaults.NetWiFiID,
					},
				},
				nil,
				DevModelTypeRaspberry),
			nil

	}
	return nil, fmt.Errorf("not implemented type: %s", devModelType)
}
