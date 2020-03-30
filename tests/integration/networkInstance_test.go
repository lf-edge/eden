package integration

import (
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eve/api/go/config"
	"testing"
)

func TestNetworkInstanceSwitch(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal(err)
	}

	deviceCtx, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal(err)
	}
	niID := "eab8761b-5f89-4e0b-b757-4b87a9fa93e1"

	err = ctx.AddNetworkInstanceConfig(&config.NetworkInstanceConfig{
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

	devUUID := deviceCtx.GetID()
	deviceCtx.SetNetworkInstanceConfig([]string{niID})
	err = ctx.ConfigSync(devUUID)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Process", func(t *testing.T) {
		err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "networkID": niID}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Active", func(t *testing.T) {
		err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "networkID": niID, "activated": "true"}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestNetworkInstanceLocal(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal(err)
	}

	deviceCtx, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal(err)
	}

	niID := "eab8761b-5f89-4e0b-b757-4b87a9fa93e2"

	err = ctx.AddNetworkInstanceConfig(&config.NetworkInstanceConfig{
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

	devUUID := deviceCtx.GetID()
	deviceCtx.SetNetworkInstanceConfig([]string{niID})
	err = ctx.ConfigSync(devUUID)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Process", func(t *testing.T) {
		err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "networkID": niID}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Active", func(t *testing.T) {
		err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "networkID": niID, "activated": "true"}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestNetworkInstanceCloud(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal(err)
	}

	deviceCtx, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal(err)
	}

	niID := "eab8761b-5f89-4e0b-b757-4b87a9fa93e3"

	err = ctx.AddNetworkInstanceConfig(&config.NetworkInstanceConfig{
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

	devUUID := deviceCtx.GetID()
	deviceCtx.SetNetworkInstanceConfig([]string{niID})
	err = ctx.ConfigSync(devUUID)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Process", func(t *testing.T) {
		err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "networkID": niID}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Active", func(t *testing.T) {
		err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "networkID": niID, "activated": "true"}, einfo.ZInfoNetworkInstance, 1000)
		if err != nil {
			t.Fatal(err)
		}
	})
}
