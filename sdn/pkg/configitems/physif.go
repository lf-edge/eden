package configitems

import (
	"fmt"
	"net"

	"github.com/lf-edge/eve/libs/depgraph"
)

// PhysIf : physical network interface.
// External item used to represent a presence (or lack) of a NIC.
type PhysIf struct {
	// IfName : Interface name assigned by the OS.
	IfName string
	// MAC address assigned by Eden.
	MAC net.HardwareAddr
	// LogicalLabel : label used within the network model.
	LogicalLabel string
}

// Name
func (p PhysIf) Name() string {
	return p.MAC.String()
}

// Label
func (p PhysIf) Label() string {
	return p.LogicalLabel
}

// Type
func (p PhysIf) Type() string {
	return PhysIfTypename
}

// Equal is a comparison method for two PhysIf instances.
func (p PhysIf) Equal(other depgraph.Item) bool {
	p2 := other.(PhysIf)
	return p.IfName == p2.IfName && p.LogicalLabel == p2.LogicalLabel
}

// External returns true because we learn about a presence of a physical interface
// through netlink API.
func (p PhysIf) External() bool {
	return true
}

// String describes the interface.
func (p PhysIf) String() string {
	return fmt.Sprintf("Physical Network Interface: %#+v", p)
}

// Dependencies returns nothing (external item).
func (p PhysIf) Dependencies() (deps []depgraph.Dependency) {
	return nil
}
