package configitems

import (
	"context"
	"fmt"
	"net"

	"github.com/lf-edge/eden/sdn/pkg/netmonitor"
	"github.com/lf-edge/eve/libs/depgraph"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// IfUsage : how a network interface is being used.
type IfUsage uint8

const (
	// IfUsageUnspecified : not specified how a network interface is being used.
	IfUsageUnspecified IfUsage = iota
	// IfUsageL3 : network interface is used in the L3 mode.
	IfUsageL3
	// IfUsageBridged : network interface is bridged.
	IfUsageBridged
	// IfUsageAggregated : network interface is aggregated by Bond interface.
	IfUsageAggregated
)

// IfHandle : an item representing *exclusive* allocation and use of a physical interface.
type IfHandle struct {
	// PhysIfLL : logical label of the physical interface associated with this handle.
	PhysIfLL string
	// PhysIfMAC : MAC address of the physical interface associated with this handle.
	PhysIfMAC net.HardwareAddr
	// Usage : How is the physical network interface being used.
	Usage IfUsage
	// ParentLL : Logical label of the parent bridge or bond if the physical interface
	// is bridged or aggregated, respectively.
	// Leave empty for L3 interfaces.
	ParentLL string
	// AdminUP : enable to put the physical interface administratively UP.
	AdminUP bool
}

// Name
func (h IfHandle) Name() string {
	return h.PhysIfMAC.String()
}

// Label
func (h IfHandle) Label() string {
	return h.PhysIfLL + " (handle)"
}

// Type
func (h IfHandle) Type() string {
	return IfHandleTypename
}

// Equal is a comparison method for two equally-named IfHandle instances.
func (h IfHandle) Equal(other depgraph.Item) bool {
	h2 := other.(IfHandle)
	return h.Usage == h2.Usage &&
		h.ParentLL == h2.ParentLL &&
		h.AdminUP == h2.AdminUP
}

// External returns false.
func (h IfHandle) External() bool {
	return false
}

// String describes the handle.
func (h IfHandle) String() string {
	return fmt.Sprintf("Physical Network Interface Handle: %#+v", h)
}

// Dependencies returns the physical interface as the only dependency.
func (h IfHandle) Dependencies() (deps []depgraph.Dependency) {
	return []depgraph.Dependency{
		{
			RequiredItem: depgraph.ItemRef{
				ItemType: PhysIfTypename,
				ItemName: h.PhysIfMAC.String(),
			},
			Description: "Underlying physical network interface must exist",
		},
	}
}

// IfHandleConfigurator implements Configurator interface for IfHandle.
type IfHandleConfigurator struct {
	netMonitor *netmonitor.NetworkMonitor
}

// Create sets interface admin state.
func (c *IfHandleConfigurator) Create(ctx context.Context, item depgraph.Item) error {
	ifHandle := item.(IfHandle)
	return c.setAdminState(ifHandle.PhysIfMAC, ifHandle.AdminUP)
}

// Modify is able to change interface admin status.
func (c *IfHandleConfigurator) Modify(ctx context.Context, oldItem, newItem depgraph.Item) (err error) {
	ifHandle := newItem.(IfHandle)
	return c.setAdminState(ifHandle.PhysIfMAC, ifHandle.AdminUP)
}

func (c *IfHandleConfigurator) setAdminState(mac net.HardwareAddr, up bool) error {
	netIf, found := c.netMonitor.LookupInterfaceByMAC(mac)
	if !found {
		err := fmt.Errorf("failed to get physical interface with MAC %v", mac)
		log.Error(err)
		return err
	}
	link, err := netlink.LinkByName(netIf.IfName)
	if err != nil {
		err = fmt.Errorf("netlink.LinkByName(%s) failed: %v", netIf.IfName, err)
		log.Error(err)
		return err
	}
	if up {
		err = netlink.LinkSetUp(link)
		if err != nil {
			err = fmt.Errorf("netlink.LinkSetUp(%s) failed: %v", link.Attrs().Name, err)
			log.Error(err)
		}
	} else {
		err = netlink.LinkSetDown(link)
		if err != nil {
			err = fmt.Errorf("netlink.LinkSetDown(%s) failed: %v", link.Attrs().Name, err)
			log.Error(err)
		}
	}
	return err
}

// Delete sets interface DOWN.
func (c *IfHandleConfigurator) Delete(ctx context.Context, item depgraph.Item) error {
	ifHandle := item.(IfHandle)
	return c.setAdminState(ifHandle.PhysIfMAC, false)
}

// NeedsRecreate returns true if the usage of PhysIf changed.
// This triggers recreate which cascades up through the graph of dependencies.
func (c *IfHandleConfigurator) NeedsRecreate(oldItem, newItem depgraph.Item) (recreate bool) {
	oldIfHandle := oldItem.(IfHandle)
	newIfHandle := newItem.(IfHandle)
	if oldIfHandle.Usage != newIfHandle.Usage || oldIfHandle.ParentLL != newIfHandle.ParentLL {
		return true
	}
	return false
}
