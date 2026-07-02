package openevec

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eve-api/go/config"
	"github.com/lf-edge/eve-api/go/evecommon"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// AddWirelessPort injects a WiFi device port carrying ENCRYPTED credentials into
// the device config: a wireless PhysicalIO, a NetworkConfig whose WifiConfig holds
// the credentials encrypted (ECDH) against the device's cert, and a NON-management
// SystemAdapter binding them. EVE decrypts the credentials at device-config ingest
// (nim/zedagent) using /persist/certs/ecdh.*, independent of the app pipeline and
// of whether a radio is physically present — which is what lets the F9 restore test
// prove decryption works from the RESTORED ecdh cert.
//
// The QEMU device model (ZedVirtual-4G) has no wireless adapter, so this synthesizes
// one. portName is the phy/logical label + interface name (e.g. "wlan0"); ssid is
// the WiFi SSID; username/password are the (to-be-encrypted) EAP identity / PSK.
// useEncryptCert selects the controller encrypt cert (CONTROLLER_ECDH_EXCHANGE).
func (openEVEC *OpenEVEC) AddWirelessPort(controllerMode, portName, ssid, username, password string, useEncryptCert bool) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig: %w", err)
	}

	// Encrypt the credentials against the device ECDH cert (registers a
	// CipherContext on dev so it is pushed with the config).
	encBlock := &evecommon.EncryptionBlock{WifiUserName: username, WifiPassword: password}
	cipherBlock, err := expect.EncryptForDevice(ctrl, dev, encBlock, useEncryptCert)
	if err != nil {
		return fmt.Errorf("EncryptForDevice: %w", err)
	}
	if cipherBlock == nil {
		return fmt.Errorf("encryption produced no cipher block (device ECDH cert available?)")
	}

	// Wireless PhysicalIO (synthesized WLAN adapter).
	pioID, err := uuid.NewV4()
	if err != nil {
		return err
	}
	physio := &config.PhysicalIO{
		Ptype:        evecommon.PhyIoType_PhyIoNetWLAN,
		Phylabel:     portName,
		Logicallabel: portName,
		Phyaddrs:     map[string]string{"ifname": portName},
		Assigngrp:    portName,
		Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageShared,
	}
	if err := ctrl.AddPhysicalIO(pioID.String(), physio); err != nil {
		return fmt.Errorf("AddPhysicalIO: %w", err)
	}
	dev.SetPhysicalIOConfig(append(dev.GetPhysicalIOs(), pioID.String()))

	// NetworkConfig carrying the encrypted WiFi credentials.
	netID, err := uuid.NewV4()
	if err != nil {
		return err
	}
	netcfg := &config.NetworkConfig{
		Id:   netID.String(),
		Type: evecommon.NetworkType_V4,
		Wireless: &config.WirelessConfig{
			Type: evecommon.WirelessType_WiFi,
			WifiCfg: []*config.WifiConfig{{
				WifiSSID:   ssid,
				KeyScheme:  evecommon.WiFiKeyScheme_WPAPSK,
				CipherData: cipherBlock,
			}},
		},
	}
	if err := ctrl.AddNetworkConfig(netcfg); err != nil {
		return fmt.Errorf("AddNetworkConfig: %w", err)
	}
	dev.SetNetworkConfig(append(dev.GetNetworks(), netID.String()))

	// NON-management SystemAdapter (uplink=false, high cost) bound to the port+network.
	saID, err := uuid.NewV4()
	if err != nil {
		return err
	}
	adapter := &config.SystemAdapter{
		Name:           portName,
		LowerLayerName: portName,
		NetworkUUID:    netID.String(),
		Uplink:         false,
		Cost:           255,
	}
	if err := ctrl.AddSystemAdapter(saID.String(), adapter); err != nil {
		return fmt.Errorf("AddSystemAdapter: %w", err)
	}
	dev.SetSystemAdaptersConfig(append(dev.GetSystemAdapters(), saID.String()))

	if err := changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %w", err)
	}
	log.Infof("added encrypted WiFi port %q (ssid=%q) as non-mgmt adapter", portName, ssid)
	return nil
}
