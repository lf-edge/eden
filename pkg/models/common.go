package models

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/evecommon"
)

func generateNetworkConfigs(ethCount, wifiCount uint) []*config.NetworkConfig {
	var networkConfigs []*config.NetworkConfig
	proxyCert := "-----BEGIN CERTIFICATE-----\nMIIFIjCCAwqgAwIBAgIILnV2rF6rRj8wDQYJKoZIhvcNAQEMBQAwLzELMAkGA1UE\nBhMCVVMxEDAOBgNVBAoTB2xmLWVkZ2UxDjAMBgNVBAMTBXByb3h5MB4XDTIyMDcw\nODE0MTY0MloXDTMyMDcwNTE0MTY0MlowLzELMAkGA1UEBhMCVVMxEDAOBgNVBAoT\nB2xmLWVkZ2UxDjAMBgNVBAMTBXByb3h5MIICIjANBgkqhkiG9w0BAQEFAAOCAg8A\nMIICCgKCAgEAmz4kI8FwvqQKZ+bcXB9Elme3B1hG6fo0gU7Ej1JpR0grfkiea1Kn\ng06RiGYjUgl5zQ3MmyE9FQs6SSqbohoWfZv5FabnbqWYy6zjHz4cNeFvfV5kHfe4\nUnNUNwLYAYni1InP3iqdVKhCKHS6+5FjvB8iwN5SesBf6yqHKli8+Lm54YIqZRFx\n9yMJyM3qCquuqiQJiKibx+76UIUWuf9Whf64p1NLaAlpbq3tbNmzV32BCzn8Otf9\nMv+wnGvxzQDsPRTfgBskptsPF2K28932iSLMudZTnuXfl6ydaHpYNK6SI/3GmkJa\nViZzIGNsjAz31QTqd/06VTAVL3597fSIwBnXSG3NryjKe1qhulk+7hhXiVui32c0\njvkwgTSrWb1FGuzkgUqWXfdUIiDTT/0rDBIbRq23rlonYOnMPJI9G4PQwmOZTFXi\nkLg8qq28pz/jHTpn8VqKF/XMDUxf/0EFc/vejk8cgzDDAlqkqkvEGee483PyuPi2\netBX9/+ngFoYDSZqCnPgAShYqq9qroIjtg7/cbdr8KiMO4Dj6yHM2OzTXQ6THjCq\nV0As2i64YGDDaMhsavvwB2geznjBXf/extQiVshLEm08RQ8HViRzm0G1p+7nHQDb\nVsY25yg9ETnCRMukiBzUXF6uV8z3JauSj1eIZGqgL0wQ+bIpcX1BaNkCAwEAAaNC\nMEAwDwYDVR0TAQH/BAUwAwEB/zAOBgNVHQ8BAf8EBAMCAQYwHQYDVR0OBBYEFGVF\n4X8j3OGihtnFiC3BZ33zqvfjMA0GCSqGSIb3DQEBDAUAA4ICAQATNvl+IgAwz3Gx\ni+6WHiqsVwRWKMifZKY952lcusviq0m6Aa/48ifZ4fc7nOTJ/pEXHDJKF/0ObYYH\n83j8AenkAp5lHEHZXfX9138QEhmMaBFqSS7IH1Vt8rvr3ZUJdq9rRNLLbnmnewdK\nC+YKwFyuqbdjdPMRQJiBWb2WyiWLydn/fMvU8Tsdcriyn4bcJdu2+4iPsb4e1YvA\n/ubGBS5Yt/v7iulNGEu9jp5nxfaBaQrc63HXcvyM4f9Q4kMNuOe+hJ/fDtUVSX67\nGTfXQGwEWUUaT4nI8mbk55KDdCy1Li4Ky5YweVR7rPeBLZ7Z+LW6pN51JYUzD/LQ\nYPhwM/mgEE+cum5jlMOBKkDwqrLpwerThfWJ2Ry+eCeLPb6qxjO7jWZcsL9yhQFX\nyBiq6F+zUKN6kRp4kOKvEM52aN0lpY4WY9xvRfehog1NS6YADQqm2zNdiwzu1RGg\nIwLzYJF6lJfTS8vPK1DNeZS9rvchc+v1ABfaLlyo4eQ2xtl6T4ynDDuRnFGppq7u\n5ZK/cM21U+CmMLs3l1yAuXoCoD+XT1P5kzJPtyjKaImSvNJHA9nKiWamrwmeTNPS\n0J7/B8Zv4EKn8mYTe4Okzn2GmOUF8djEyxWuOyfPdROJG5/oNhOJQtOd16xhnX+4\ni5dPDNhKEZtP3KeY2vQRoldysRSbFA==\n-----END CERTIFICATE-----"
	if ethCount > 0 {
		networkConfigs = append(networkConfigs,
			&config.NetworkConfig{
				Id:   defaults.NetDHCPID,
				Type: config.NetworkType_V4,
				Ip: &config.Ipspec{
					Dhcp:      config.DHCPType_Client,
					DhcpRange: &config.IpRange{},
				},
				EntProxy: &config.ProxyConfig{
					Proxies: []*config.ProxyServer{
						{
							Server: "192.168.120.1",
							Port:   8080,
							Proto:  config.ProxyProto_PROXY_HTTP,
						},
						{
							Server: "192.168.120.1",
							Port:   8080,
							Proto:  config.ProxyProto_PROXY_HTTPS,
						},
					},
					ProxyCertPEM: [][]byte{
						[]byte(proxyCert),
					},
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
					EntProxy: &config.ProxyConfig{
						Proxies: []*config.ProxyServer{
							{
								Server: "192.168.120.1",
								Port:   8080,
								Proto:  config.ProxyProto_PROXY_HTTP,
							},
							{
								Server: "192.168.120.1",
								Port:   8080,
								Proto:  config.ProxyProto_PROXY_HTTPS,
							},
						},
						ProxyCertPEM: [][]byte{
							[]byte(proxyCert),
						},
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
