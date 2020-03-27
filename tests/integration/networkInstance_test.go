package integration

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/cloud"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/einfo"
	"github.com/lf-edge/eve/api/go/config"
	"testing"
)

func TestNetworkInstanceSwitch(t *testing.T) {
	ctx, devUUID, err := adamPrepare()
	if err != nil {
		t.Fatal(err)
	}

	niID := "eab8761b-5f89-4e0b-b757-4b87a9fa93e1"

	cloudCxt := &cloud.Ctx{}
	err = cloudCxt.AddNetworkInstanceConfig(&config.NetworkInstanceConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    niID,
			Version: "4",
		},
		Displayname: "testSwitch",
		InstType:    config.ZNetworkInstType_ZnetInstSwitch,
		Activate:    true,
		Port: &config.Adapter{
			Type: config.PhyIoType_PhyIoNoop,
			Name: "uplink",
		},
		Cfg:    &config.NetworkInstanceOpaqueConfig{},
		IpType: config.AddressType_IPV4,
		Ip: &config.Ipspec{
			Dhcp:    config.DHCPType_DHCPNoop,
			Subnet:  "10.1.0.0/16",
			Gateway: "10.1.0.1",
			Domain:  "",
			Ntp:     "",
			Dns:     []string{"10.1.0.1"},
			DhcpRange: &config.IpRange{
				Start: "10.1.0.2",
				End:   "10.1.255.254",
			},
		},
		Dns: nil,
	})
	if err != nil {
		t.Fatal(err)
	}

	deviceCtx := device.CreateWithBaseConfig(*devUUID, cloudCxt)
	deviceCtx.SetNetworkInstanceConfig([]string{niID})
	b, err := deviceCtx.GenerateJSONBytes()
	if err != nil {
		t.Fatal(err)
	}
	configToSet := fmt.Sprintf("%s", string(b))
	t.Log(configToSet)
	err = ctx.ConfigSet(devUUID.String(), configToSet)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Process", func(t *testing.T) {
		err = einfo.InfoChecker(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String(), "networkID": niID}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Active", func(t *testing.T) {
		err = einfo.InfoChecker(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String(), "networkID": niID, "activated": "true"}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestNetworkInstanceLocal(t *testing.T) {
	ctx, devUUID, err := adamPrepare()
	if err != nil {
		t.Fatal(err)
	}

	niID := "eab8761b-5f89-4e0b-b757-4b87a9fa93e2"

	cloudCxt := &cloud.Ctx{}
	err = cloudCxt.AddNetworkInstanceConfig(&config.NetworkInstanceConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    niID,
			Version: "4",
		},
		Displayname: "testLocal",
		InstType:    config.ZNetworkInstType_ZnetInstLocal,
		Activate:    true,
		Port: &config.Adapter{
			Type: config.PhyIoType_PhyIoNoop,
			Name: "uplink",
		},
		Cfg:    &config.NetworkInstanceOpaqueConfig{},
		IpType: config.AddressType_IPV4,
		Ip: &config.Ipspec{
			Dhcp:    config.DHCPType_DHCPNoop,
			Subnet:  "10.1.0.0/16",
			Gateway: "10.1.0.1",
			Domain:  "",
			Ntp:     "",
			Dns:     []string{"10.1.0.1"},
			DhcpRange: &config.IpRange{
				Start: "10.1.0.2",
				End:   "10.1.255.254",
			},
		},
		Dns: nil,
	})
	if err != nil {
		t.Fatal(err)
	}

	deviceCtx := device.CreateWithBaseConfig(*devUUID, cloudCxt)
	deviceCtx.SetNetworkInstanceConfig([]string{niID})
	b, err := deviceCtx.GenerateJSONBytes()
	if err != nil {
		t.Fatal(err)
	}
	configToSet := fmt.Sprintf("%s", string(b))
	t.Log(configToSet)
	err = ctx.ConfigSet(devUUID.String(), configToSet)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Process", func(t *testing.T) {
		err = einfo.InfoChecker(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String(), "networkID": niID}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Active", func(t *testing.T) {
		err = einfo.InfoChecker(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String(), "networkID": niID, "activated": "true"}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestNetworkInstanceCloud(t *testing.T) {
	ctx, devUUID, err := adamPrepare()
	if err != nil {
		t.Fatal(err)
	}

	niID := "eab8761b-5f89-4e0b-b757-4b87a9fa93e3"

	cloudCxt := &cloud.Ctx{}
	err = cloudCxt.AddNetworkInstanceConfig(&config.NetworkInstanceConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    niID,
			Version: "4",
		},
		Displayname: "testCloud",
		InstType:    config.ZNetworkInstType_ZnetInstCloud,
		Activate:    true,
		Port: &config.Adapter{
			Type: config.PhyIoType_PhyIoNoop,
			Name: "uplink",
		},
		Cfg:    &config.NetworkInstanceOpaqueConfig{},
		IpType: config.AddressType_IPV4,
		Ip: &config.Ipspec{
			Dhcp:    config.DHCPType_DHCPNoop,
			Subnet:  "10.1.0.0/16",
			Gateway: "10.1.0.1",
			Domain:  "",
			Ntp:     "",
			Dns:     []string{"10.1.0.1"},
			DhcpRange: &config.IpRange{
				Start: "10.1.0.2",
				End:   "10.1.255.254",
			},
		},
		Dns: nil,
	})
	if err != nil {
		t.Fatal(err)
	}

	deviceCtx := device.CreateWithBaseConfig(*devUUID, cloudCxt)
	deviceCtx.SetNetworkInstanceConfig([]string{niID})
	b, err := deviceCtx.GenerateJSONBytes()
	if err != nil {
		t.Fatal(err)
	}
	configToSet := fmt.Sprintf("%s", string(b))
	t.Log(configToSet)
	err = ctx.ConfigSet(devUUID.String(), configToSet)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Process", func(t *testing.T) {
		err = einfo.InfoChecker(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String(), "networkID": niID}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Active", func(t *testing.T) {
		err = einfo.InfoChecker(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String(), "networkID": niID, "activated": "true"}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
}
