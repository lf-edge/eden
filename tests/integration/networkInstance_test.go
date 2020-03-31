package integration

import (
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/pkg/errors"
	"testing"
)

const cloudOConfig = `{ "VpnRole": "onPremClient",
  "VpnGatewayIpAddr": "192.168.254.51",
  "VpnSubnetBlock": "20.1.0.0/24",
  "ClientConfigList": [{"IpAddr": "%any", "PreSharedKey": "0sjVzONCF02ncsgiSlmIXeqhGN", "SubnetBlock": "30.1.0.0/24"}]
}`

func prepareNetworkInstance(ctx controller.Cloud, networkInstanceID string, networkInstanceName string, networkInstanceType config.ZNetworkInstType) error {
	uid := config.UUIDandVersion{
		Uuid:    networkInstanceID,
		Version: "4",
	}
	networkInstance := config.NetworkInstanceConfig{
		Uuidandversion: &uid,
		Displayname:    networkInstanceName,
		InstType:       networkInstanceType,
		Activate:       true,
		Port:           nil,
		Cfg:            nil,
		IpType:         config.AddressType_IPV4,
		Ip:             nil,
	}
	switch networkInstanceType {
	case config.ZNetworkInstType_ZnetInstSwitch:
		networkInstance.Port = &config.Adapter{
			Type: config.PhyIoType_PhyIoNoop,
			Name: "uplink",
		}
		networkInstance.Ip = &config.Ipspec{
			Dhcp:    config.DHCPType_DHCPNoop,
			Subnet:  "10.1.0.0/24",
			Gateway: "10.1.0.1",
			Domain:  "",
			Ntp:     "",
			Dns:     []string{"10.1.0.1"},
			DhcpRange: &config.IpRange{
				Start: "10.1.0.2",
				End:   "10.1.0.254",
			},
		}
		networkInstance.Cfg = &config.NetworkInstanceOpaqueConfig{}
	case config.ZNetworkInstType_ZnetInstLocal:
		networkInstance.Port = &config.Adapter{
			Type: config.PhyIoType_PhyIoNoop,
			Name: "uplink",
		}
		networkInstance.Ip = &config.Ipspec{
			Dhcp:    config.DHCPType_DHCPNoop,
			Subnet:  "20.1.0.0/24",
			Gateway: "20.1.0.1",
			Domain:  "",
			Ntp:     "",
			Dns:     []string{"20.1.0.1"},
			DhcpRange: &config.IpRange{
				Start: "20.1.0.2",
				End:   "20.1.0.254",
			},
		}
		networkInstance.Cfg = &config.NetworkInstanceOpaqueConfig{}
	case config.ZNetworkInstType_ZnetInstCloud:
		networkInstance.Port = &config.Adapter{
			Type: config.PhyIoType_PhyIoNoop,
			Name: "uplink",
		}
		networkInstance.Ip = &config.Ipspec{
			Dhcp:    config.DHCPType_DHCPNoop,
			Subnet:  "30.1.0.0/24",
			Gateway: "30.1.0.1",
			Domain:  "",
			Ntp:     "",
			Dns:     []string{"30.1.0.1"},
			DhcpRange: &config.IpRange{
				Start: "30.1.0.2",
				End:   "30.1.0.254",
			},
		}
		networkInstance.Cfg = &config.NetworkInstanceOpaqueConfig{
			Oconfig:    cloudOConfig,
			LispConfig: nil,
			Type:       config.ZNetworkOpaqueConfigType_ZNetOConfigVPN,
		}
	default:
		return errors.New("not implemented type")
	}
	return ctx.AddNetworkInstanceConfig(&networkInstance)
}

func TestNetworkInstance(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal("Fail in controller prepare: ", err)
	}

	deviceCtx, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}
	var networkInstanceTests = []struct {
		networkInstanceID   string
		networkInstanceName string
		networkInstanceType config.ZNetworkInstType
	}{
		{"eab8761b-5f89-4e0b-b757-4b87a9fa93e1",

			"testLocal",

			config.ZNetworkInstType_ZnetInstLocal,
		},
		{"eab8761b-5f89-4e0b-b757-4b87a9fa93e2",

			"testSwitch",

			config.ZNetworkInstType_ZnetInstSwitch,
		},
		{"eab8761b-5f89-4e0b-b757-4b87a9fa93e3",

			"testCloud",

			config.ZNetworkInstType_ZnetInstCloud,
		},
	}
	for _, tt := range networkInstanceTests {
		t.Run(tt.networkInstanceName, func(t *testing.T) {

			err = prepareNetworkInstance(ctx, tt.networkInstanceID, tt.networkInstanceName, tt.networkInstanceType)
			if err != nil {
				t.Fatal("Fail in prepare network instance: ", err)
			}

			devUUID := deviceCtx.GetID()
			deviceCtx.SetNetworkInstanceConfig([]string{tt.networkInstanceID})
			err = ctx.ConfigSync(devUUID)
			if err != nil {
				t.Fatal("Fail in sync config with controller: ", err)
			}
			t.Run("Process", func(t *testing.T) {
				err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "networkID": tt.networkInstanceID}, einfo.ZInfoNetworkInstance, 300)
				if err != nil {
					t.Fatal("Fail in waiting for process start from info: ", err)
				}
			})
			t.Run("Active", func(t *testing.T) {
				err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "networkID": tt.networkInstanceID, "activated": "true"}, einfo.ZInfoNetworkInstance, 600)
				if err != nil {
					t.Fatal("Fail in waiting for activated state from info: ", err)
				}
			})
		})
	}
}
