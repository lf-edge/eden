package openevec

import (
	"fmt"
	"path"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/packet"
	log "github.com/sirupsen/logrus"
)

func PacketRun(packetKey, packetProjectName, packetVMName, packetZone, packetMachineType, packetIPXEUrl string, cfg *EdenSetupArgs) error {
	if packetIPXEUrl == "" {
		configPrefix := cfg.ConfigName
		if cfg.ConfigName == defaults.DefaultContext {
			configPrefix = ""
		}
		packetIPXEUrl = fmt.Sprintf("http://%s:%d/%s/ipxe.efi.cfg", cfg.Adam.CertsEVEIP, cfg.Eden.EServer.Port, path.Join("eserver", configPrefix))
		log.Debugf("ipxe-url is empty, will use default one: %s", packetIPXEUrl)
	}

	packetClient, err := packet.NewPacketClient(packetKey, packetProjectName)
	if err != nil {
		return fmt.Errorf("unable to connect to create packet client: %w", err)
	}
	if err := packetClient.CreateInstance(packetVMName, packetZone, packetMachineType, packetIPXEUrl); err != nil {
		return fmt.Errorf("failed to CreateInstance: %w", err)
	}
	return nil
}

func PacketDelete(packetKey, packetProjectName, packetVMName string) error {
	packetClient, err := packet.NewPacketClient(packetKey, packetProjectName)
	if err != nil {
		return fmt.Errorf("unable to connect to create packet client: %w", err)
	}
	if err := packetClient.DeleteInstance(packetVMName); err != nil {
		return fmt.Errorf("DeleteInstance: %w", err)
	}
	return nil
}

func PacketGetIP(packetKey, packetProjectName, packetVMName string) error {
	packetClient, err := packet.NewPacketClient(packetKey, packetProjectName)
	if err != nil {
		return fmt.Errorf("unable to connect to create packet client: %w", err)
	}
	natIP, err := packetClient.GetInstanceNatIP(packetVMName)
	if err != nil {
		return err
	}
	fmt.Println(natIP)
	return nil
}
