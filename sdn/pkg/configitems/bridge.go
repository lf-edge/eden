package configitems

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/lf-edge/eden/sdn/pkg/netmonitor"
	"github.com/lf-edge/eve/libs/depgraph"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// Bridge : Linux bridge.
type Bridge struct {
	// IfName : name of the Bridge in the OS.
	IfName string
	// LogicalLabel : logical name used for reference.
	LogicalLabel string
	// PhysIfs : physical interfaces to put under the bridge.
	PhysIfs []PhysIf
	// BondIfs : *interface names* of bonds to put under the bridge.
	BondIfs []string
	// VLANs : list of VLANs used with this bridge.
	// If empty then this bridge is used without VLAN filtering.
	VLANs []uint16
}

// Name
func (b Bridge) Name() string {
	return b.IfName
}

// Label
func (b Bridge) Label() string {
	return b.LogicalLabel + " (bridge)"
}

// Type
func (b Bridge) Type() string {
	return BridgeTypename
}

// Equal is a comparison method for two equally-named Bridge instances.
func (b Bridge) Equal(other depgraph.Item) bool {
	b2 := other.(Bridge)
	return b.LogicalLabel == b2.LogicalLabel &&
		reflect.DeepEqual(b.PhysIfs, b2.PhysIfs) &&
		reflect.DeepEqual(b.BondIfs, b2.BondIfs) &&
		reflect.DeepEqual(b.VLANs, b2.VLANs)
}

// External returns false.
func (b Bridge) External() bool {
	return false
}

// String describes Bridge.
func (b Bridge) String() string {
	return fmt.Sprintf("Bridge: %#+v", b)
}

// Dependencies lists all bridged interfaces as dependencies.
func (b Bridge) Dependencies() (deps []depgraph.Dependency) {
	for _, physIf := range b.PhysIfs {
		deps = append(deps, depgraph.Dependency{
			RequiredItem: depgraph.ItemRef{
				ItemType: IfHandleTypename,
				ItemName: physIf.MAC.String(),
			},
			// Requires exclusive access to the physical interface.
			MustSatisfy: func(item depgraph.Item) bool {
				ioHandle := item.(IfHandle)
				return ioHandle.Usage == IfUsageBridged &&
					ioHandle.ParentLL == b.LogicalLabel
			},
			Description: "Bridged physical interface must exist",
		})
	}
	for _, bondIfName := range b.BondIfs {
		deps = append(deps, depgraph.Dependency{
			RequiredItem: depgraph.ItemRef{
				ItemType: BondTypename,
				ItemName: bondIfName,
			},
			Description: "Bridged bond interface must exist",
		})
	}
	return deps
}

// BridgeConfigurator implements Configurator interface for bond interfaces.
type BridgeConfigurator struct {
	netMonitor *netmonitor.NetworkMonitor
}

// Create adds new Bridge.
func (c *BridgeConfigurator) Create(ctx context.Context, item depgraph.Item) error {
	bridgeCfg := item.(Bridge)
	attrs := netlink.NewLinkAttrs()
	attrs.Name = bridgeCfg.IfName
	bridge := &netlink.Bridge{LinkAttrs: attrs}
	if err := netlink.LinkAdd(bridge); err != nil {
		err = fmt.Errorf("failed to add bridge %s: %v", bridgeCfg.IfName, err)
		log.Error(err)
		return err
	}
	if err := netlink.LinkSetUp(bridge); err != nil {
		err = fmt.Errorf("failed to set bridge %s UP: %v", bridgeCfg.IfName, err)
		log.Error(err)
		return err
	}
	// Put interface under the bridge using the Modify handler.
	emptyBridge := Bridge{
		IfName:       bridgeCfg.IfName,
		LogicalLabel: bridgeCfg.LogicalLabel,
	}
	return c.handleModify(emptyBridge, bridgeCfg)
}

func (c *BridgeConfigurator) putIfUnderBridge(bridge *netlink.Bridge, ifName string) error {
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return err
	}
	err = netlink.LinkSetDown(link)
	if err != nil {
		return err
	}
	err = netlink.LinkSetMaster(link, bridge)
	if err != nil {
		return err
	}
	err = netlink.LinkSetUp(link)
	if err != nil {
		return err
	}
	return nil
}

func (c *BridgeConfigurator) delIfFromBridge(ifName string) error {
	aggrLink, err := netlink.LinkByName(ifName)
	if err != nil {
		return err
	}
	err = netlink.LinkSetNoMaster(aggrLink)
	if err != nil {
		return err
	}
	// Releasing interface from the master causes it be automatically
	// brought down - we need to bring it back up.
	err = netlink.LinkSetUp(aggrLink)
	if err != nil {
		return err
	}
	return nil
}

// Bridge ports (going towards EVE) are all trunks
// (VETHs are used as access ports for networks).
func (c *BridgeConfigurator) updateVLANs(ifName string, prevVLANs, newVLANs []uint16) error {
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		err = fmt.Errorf("failed to get link %s for VLAN update: %w",
			ifName, err)
		return err
	}
	for _, vlanID := range prevVLANs {
		var keepVLAN bool
		for _, vlanID2 := range newVLANs {
			if vlanID == vlanID2 {
				keepVLAN = true
				break
			}
		}
		if !keepVLAN {
			err = netlink.BridgeVlanDel(link, vlanID, false, false, false, false)
			if err != nil {
				err = fmt.Errorf("failed to remove VLAN (%d) from (trunk) port '%s': %w",
					vlanID, ifName, err)
				return err
			}
		}
	}
	for _, vlanID := range newVLANs {
		var alreadyAdded bool
		for _, vlanID2 := range prevVLANs {
			if vlanID == vlanID2 {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			err = netlink.BridgeVlanAdd(link, vlanID, false, false, false, false)
			if err != nil {
				err = fmt.Errorf("failed to add VLAN (%d) to (trunk) port '%s': %w",
					vlanID, ifName, err)
				return err
			}
		}
	}
	return nil
}

// Modify is able to change the set of bridged interfaces.
func (c *BridgeConfigurator) Modify(ctx context.Context, oldItem, newItem depgraph.Item) (err error) {
	oldBridgeCfg := oldItem.(Bridge)
	newBridgeCfg := newItem.(Bridge)
	return c.handleModify(oldBridgeCfg, newBridgeCfg)
}

func (c *BridgeConfigurator) handleModify(oldBridgeCfg, newBridgeCfg Bridge) (err error) {
	ifName := oldBridgeCfg.IfName
	bridgeLink, err := netlink.LinkByName(ifName)
	if err != nil {
		log.Error(err)
		return err
	}
	if bridgeLink.Type() != "bridge" {
		err = fmt.Errorf("interface %s is not Bridge", ifName)
		log.Error(err)
		return err
	}
	bridge := bridgeLink.(*netlink.Bridge)
	// Update VLAN filering.
	vlanFiltering := len(newBridgeCfg.VLANs) > 0
	if *bridge.VlanFiltering != vlanFiltering {
		if err := netlink.BridgeSetVlanFiltering(bridge, vlanFiltering); err != nil {
			err = fmt.Errorf("failed to set VLAN filtering to %t for bridge %s: %v",
				vlanFiltering, ifName, err)
			log.Error(err)
			return err
		}
	}
	// Remove interfaces which are no longer configured to be under the Bridge.
	for _, oldPhysIf := range oldBridgeCfg.PhysIfs {
		var keepBridged bool
		for _, newPhysIf := range newBridgeCfg.PhysIfs {
			if bytes.Equal(oldPhysIf.MAC, newPhysIf.MAC) {
				keepBridged = true
				break
			}
		}
		if !keepBridged {
			netIf, found := c.netMonitor.LookupInterfaceByMAC(oldPhysIf.MAC)
			if !found {
				err := fmt.Errorf("failed to get physical interface with MAC %v",
					oldPhysIf.MAC)
				log.Error(err)
				return err
			}
			err := c.delIfFromBridge(netIf.IfName)
			if err != nil {
				err = fmt.Errorf("failed to release interface %s from bridge %s: %v",
					netIf.IfName, ifName, err)
				log.Error(err)
				return err
			}
		}
	}
	for _, oldBondIf := range oldBridgeCfg.BondIfs {
		var keepBridged bool
		for _, newBondIf := range newBridgeCfg.BondIfs {
			if oldBondIf == newBondIf {
				keepBridged = true
				break
			}
		}
		if !keepBridged {
			err := c.delIfFromBridge(oldBondIf)
			if err != nil {
				err = fmt.Errorf("failed to release bond %s from bridge %s: %v",
					oldBondIf, ifName, err)
				log.Error(err)
				return err
			}
		}
	}
	// Add interfaces newly configured to be under this Bridge.
	for _, newPhysIf := range newBridgeCfg.PhysIfs {
		netIf, foundPhysIf := c.netMonitor.LookupInterfaceByMAC(newPhysIf.MAC)
		if !foundPhysIf {
			err := fmt.Errorf("failed to get physical interface with MAC %v",
				newPhysIf.MAC)
			log.Error(err)
			return err
		}
		var alreadyBridged bool
		for _, oldPhysIf := range oldBridgeCfg.PhysIfs {
			if bytes.Equal(oldPhysIf.MAC, newPhysIf.MAC) {
				alreadyBridged = true
				break
			}
		}
		if !alreadyBridged {
			err := c.putIfUnderBridge(bridge, netIf.IfName)
			if err != nil {
				err = fmt.Errorf("failed to put interface %s under bridge %s: %v",
					netIf.IfName, ifName, err)
				log.Error(err)
				return err
			}
		}
		err := c.updateVLANs(netIf.IfName, oldBridgeCfg.VLANs, newBridgeCfg.VLANs)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	for _, newBondIf := range newBridgeCfg.BondIfs {
		var alreadyBridged bool
		for _, oldBondIf := range oldBridgeCfg.BondIfs {
			if oldBondIf == newBondIf {
				alreadyBridged = true
				break
			}
		}
		if !alreadyBridged {
			err := c.putIfUnderBridge(bridge, newBondIf)
			if err != nil {
				err = fmt.Errorf("failed to put bond %s under bridge %s: %v",
					newBondIf, ifName, err)
				log.Error(err)
				return err
			}
		}
		err := c.updateVLANs(newBondIf, oldBridgeCfg.VLANs, newBridgeCfg.VLANs)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

// Delete removes bridge.
func (c *BridgeConfigurator) Delete(ctx context.Context, item depgraph.Item) error {
	bridgeCfg := item.(Bridge)
	// Remove all interfaces from under the bridge using the Modify handler.
	emptyBridge := Bridge{
		IfName:       bridgeCfg.IfName,
		LogicalLabel: bridgeCfg.LogicalLabel,
	}
	if err := c.handleModify(bridgeCfg, emptyBridge); err != nil {
		return err
	}
	bridge, err := netlink.LinkByName(bridgeCfg.IfName)
	if err != nil {
		err = fmt.Errorf("failed to select bridge %s for removal: %v",
			bridgeCfg.IfName, err)
		log.Error(err)
		return err
	}
	err = netlink.LinkDel(bridge)
	if err != nil {
		err = fmt.Errorf("failed to delete bond %s: %v", bridgeCfg.IfName, err)
		log.Error(err)
		return err
	}
	return nil
}

// NeedsRecreate returns false.
// The set of bridged interfaces can be changed without recreating bridge.
func (c *BridgeConfigurator) NeedsRecreate(oldItem, newItem depgraph.Item) (recreate bool) {
	return false
}
