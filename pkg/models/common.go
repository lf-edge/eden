package models

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/evecommon"
)

func generateNetworkConfigs(ethCount, wifiCount uint) []*config.NetworkConfig {
	var networkConfigs []*config.NetworkConfig
	if ethCount > 0 {
		networkConfigs = append(networkConfigs,
			&config.NetworkConfig{
				Id:   defaults.NetDHCPID,
				Type: config.NetworkType_V4,
				Ip: &config.Ipspec{
					Dhcp:      config.DHCPType_Client,
					DhcpRange: &config.IpRange{},
				},
				Wireless: nil,
			})
		if ethCount > 1 {
			networkConfigs = append(networkConfigs,
				&config.NetworkConfig{
					Id:   defaults.NetNoDHCPID,
					Type: config.NetworkType_V4,
					Ip: &config.Ipspec{
						Dhcp:      config.DHCPType_Client,
						DhcpRange: &config.IpRange{},
					},
					Wireless: nil,
				})
		}
		if ethCount > 2 {
			networkConfigs = append(networkConfigs,
				&config.NetworkConfig{
					Id:   defaults.NetSwitch,
					Type: config.NetworkType_V4,
					Ip: &config.Ipspec{
						Dhcp:      config.DHCPType_DHCPNone,
						DhcpRange: &config.IpRange{},
					},
					Wireless: nil,
				})
		}
	}
	if wifiCount > 0 {
		networkConfigs = append(networkConfigs,
			&config.NetworkConfig{
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
			})
	}

	return networkConfigs
}

func generateSystemAdapters(ethCount, wifiCount uint) []*config.SystemAdapter {
	var adapters []*config.SystemAdapter
	for i := uint(0); i < ethCount; i++ {
		name := fmt.Sprintf("eth%d", i)
		uplink := true
		networkUUID := defaults.NetDHCPID
		if i > 0 {
			uplink = false
		}
		if i == 1 {
			networkUUID = defaults.NetNoDHCPID
		}
		if i == 2 {
			networkUUID = defaults.NetSwitch
		}
		adapters = append(adapters, &config.SystemAdapter{
			Name:        name,
			Uplink:      uplink,
			NetworkUUID: networkUUID,
		})
	}
	for i := uint(0); i < wifiCount; i++ {
		name := fmt.Sprintf("wlan%d", i)
		adapters = append(adapters, &config.SystemAdapter{
			Name:        name,
			NetworkUUID: defaults.NetWiFiID,
		})
	}
	return adapters
}

func generatePhysicalIOs(ethCount, wifiCount, usbCount uint) []*config.PhysicalIO {
	var physicalIOs []*config.PhysicalIO
	for i := uint(0); i < ethCount; i++ {
		name := fmt.Sprintf("eth%d", i)
		usage := evecommon.PhyIoMemberUsage_PhyIoUsageMgmtAndApps
		if i > 0 {
			usage = evecommon.PhyIoMemberUsage_PhyIoUsageShared
		}
		physicalIOs = append(physicalIOs, &config.PhysicalIO{
			Ptype:        evecommon.PhyIoType_PhyIoNetEth,
			Phylabel:     name,
			Logicallabel: name,
			Assigngrp:    name,
			Phyaddrs:     map[string]string{"Ifname": name},
			Usage:        usage,
			UsagePolicy: &config.PhyIOUsagePolicy{
				FreeUplink: true,
			},
		})
	}
	for i := uint(0); i < wifiCount; i++ {
		name := fmt.Sprintf("wlan%d", i)
		physicalIOs = append(physicalIOs, &config.PhysicalIO{
			Ptype:        evecommon.PhyIoType_PhyIoNetWLAN,
			Phylabel:     name,
			Logicallabel: name,
			Assigngrp:    name,
			Phyaddrs:     map[string]string{"Ifname": name},
			Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDisabled,
			UsagePolicy: &config.PhyIOUsagePolicy{
				FreeUplink: false,
			},
		})
	}
	usbGroup := 0
	for i := uint(0); i < usbCount; i++ {
		for j := uint(1); j < 4; j++ {
			num := fmt.Sprintf("%d:%d", i, j)
			name := fmt.Sprintf("USB%s", num)
			physicalIOs = append(physicalIOs, &config.PhysicalIO{
				Ptype:        evecommon.PhyIoType_PhyIoUSB,
				Phylabel:     name,
				Logicallabel: name,
				Assigngrp:    fmt.Sprintf("USB%d", usbGroup),
				Phyaddrs:     map[string]string{"UsbAddr": num},
				Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageDedicated,
			})
			usbGroup++
		}
	}
	return physicalIOs
}
