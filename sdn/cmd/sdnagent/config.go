package main

import (
	"net"
	"strings"

	"github.com/lf-edge/eden/sdn/api"
	"github.com/lf-edge/eden/sdn/pkg/configitems"
	dg "github.com/lf-edge/eve/libs/depgraph"
	"github.com/lf-edge/eve/libs/reconciler"
)

const (
	// Dependency graph modeling current/intended network configuration.
	// *SG are names of sub-graphs.
	configGraphName    = "SDN-Config"
	physicalIfsSG      = "Physical-Interfaces"
	hostConnectivitySG = "Host-Connectivity"
	bridgesSG          = "Bridges"
)

// Update external items inside the graph with the current state.
func (a *agent) updateCurrentState() (changed bool) {
	if a.currentState == nil {
		graphArgs := dg.InitArgs{Name: configGraphName}
		a.currentState = dg.New(graphArgs)
		changed = true
	}
	currentPhysIfs := dg.New(dg.InitArgs{Name: physicalIfsSG})
	// Port connecting SDN VM with the host.
	if netIf, found := a.findHostInterface(); found {
		currentPhysIfs.PutItem(configitems.PhysIf{
			LogicalLabel: hostPortLogicalLabel,
			MAC:          netIf.MAC,
		}, &reconciler.ItemStateData{
			State:         reconciler.ItemStateCreated,
			LastOperation: reconciler.OperationCreate,
		})
	}
	// Ports to be connected with EVE VM(s).
	for _, port := range a.netModel.Ports {
		// MAC address is already validated
		mac, _ := net.ParseMAC(port.MAC)
		if _, found := a.netMonitor.LookupInterfaceByMAC(mac); found {
			currentPhysIfs.PutItem(configitems.PhysIf{
				LogicalLabel: port.LogicalLabel,
				MAC:          mac,
			}, &reconciler.ItemStateData{
				State:         reconciler.ItemStateCreated,
				LastOperation: reconciler.OperationCreate,
			})
		}
	}
	// Is there any actual change?
	prevSG := a.currentState.SubGraph(physicalIfsSG)
	if prevSG == nil || len(prevSG.DiffItems(currentPhysIfs)) > 0 {
		a.currentState.PutSubGraph(currentPhysIfs)
		changed = true
	}
	return changed
}

// Update graph with the intended state based on the network model stored in a.netModel
func (a *agent) updateIntendedState() {
	graphArgs := dg.InitArgs{Name: configGraphName}
	a.intendedState = dg.New(graphArgs)
	a.intendedState.PutSubGraph(a.getIntendedPhysIfs())
	a.intendedState.PutSubGraph(a.getIntendedHostConnectivity())
	a.intendedState.PutSubGraph(a.getIntendedBridges())
	// TODO ...
}

func (a *agent) getIntendedPhysIfs() dg.Graph {
	graphArgs := dg.InitArgs{Name: physicalIfsSG}
	intendedCfg := dg.New(graphArgs)
	if netIf, found := a.findHostInterface(); found {
		intendedCfg.PutItem(configitems.PhysIf{
			LogicalLabel: hostPortLogicalLabel,
			MAC:          netIf.MAC,
		}, nil)
	}
	for _, port := range a.netModel.Ports {
		// MAC address is already validated
		mac, _ := net.ParseMAC(port.MAC)
		intendedCfg.PutItem(configitems.PhysIf{
			LogicalLabel: port.LogicalLabel,
			MAC:          mac,
		}, nil)
	}
	return intendedCfg
}

func (a *agent) getIntendedHostConnectivity() dg.Graph {
	graphArgs := dg.InitArgs{Name: hostConnectivitySG}
	intendedCfg := dg.New(graphArgs)
	netIf, found := a.findHostInterface()
	if !found {
		// Without interface connecting SDN with the host it is clearly
		// not possible to establish host connectivity.
		return intendedCfg
	}
	intendedCfg.PutItem(configitems.NetNamespace{
		NsName: configitems.MainNsName,
	}, nil)
	intendedCfg.PutItem(configitems.IfHandle{
		PhysIf: configitems.PhysIf{
			MAC:          netIf.MAC,
			LogicalLabel: hostPortLogicalLabel,
		},
		Usage:   configitems.IfUsageL3,
		AdminUP: true,
	}, nil)
	intendedCfg.PutItem(configitems.IPForwarding{
		EnableForIPv4: true,
	}, nil)
	intendedCfg.PutItem(configitems.Dhcpcd{
		PhysIfLL:  hostPortLogicalLabel,
		PhysIfMAC: netIf.MAC,
		LogFile:   "/run/dhcpcd.log",
	}, nil)
	return intendedCfg
}

func (a *agent) getIntendedBridges() dg.Graph {
	graphArgs := dg.InitArgs{Name: bridgesSG}
	intendedCfg := dg.New(graphArgs)
	for _, port := range a.netModel.Ports {
		labeledItem := a.netModel.items.getItem(api.Port{}.ItemType(), port.LogicalLabel)
		masterID, hasMaster := labeledItem.referencedBy[api.PortMasterRef]
		if !hasMaster {
			// Port is not really used.
			continue
		}
		mac, _ := net.ParseMAC(port.MAC) // already validated
		var usage configitems.IfUsage
		switch masterID.typename {
		case api.Bridge{}.ItemType():
			usage = configitems.IfUsageBridged
		case api.Bond{}.ItemType():
			usage = configitems.IfUsageAggregated
		}
		intendedCfg.PutItem(configitems.IfHandle{
			PhysIf: configitems.PhysIf{
				MAC:          mac,
				LogicalLabel: port.LogicalLabel,
			},
			ParentLL: masterID.logicalLabel,
			Usage:    usage,
			AdminUP:  port.AdminUP,
			MTU:      port.MTU,
		}, nil)
	}
	for _, bond := range a.netModel.Bonds {
		labeledItem := a.netModel.items.getItem(api.Bond{}.ItemType(), bond.LogicalLabel)
		var aggrPhysIfs []configitems.PhysIf
		for _, ref := range labeledItem.referencing {
			if ref.refKey == api.PortMasterRef {
				port := a.netModel.items[ref.itemID].LabeledItem
				mac, _ := net.ParseMAC(port.(api.Port).MAC)
				aggrPhysIfs = append(aggrPhysIfs, configitems.PhysIf{
					MAC:          mac,
					LogicalLabel: ref.logicalLabel,
				})
			}
		}
		intendedCfg.PutItem(configitems.Bond{
			Bond:              bond,
			IfName:            a.bondIfName(bond.LogicalLabel),
			AggregatedPhysIfs: aggrPhysIfs,
		}, nil)
	}
	for _, bridge := range a.netModel.Bridges {
		var vlans []uint16
		labeledItem := a.netModel.items.getItem(api.Bridge{}.ItemType(), bridge.LogicalLabel)
		for refKey, refBy := range labeledItem.referencedBy {
			if strings.HasPrefix(refKey, api.NetworkBridgeRefPrefix) {
				network := a.netModel.items[refBy].LabeledItem
				if vlanID := network.(api.Network).VlanID; vlanID != 0 {
					vlans = append(vlans, vlanID)
				}
			}
		}
		var physIfs []configitems.PhysIf
		var bonds []string
		for _, ref := range labeledItem.referencing {
			if ref.refKey != api.PortMasterRef {
				continue
			}
			switch ref.typename {
			case api.Port{}.ItemType():
				port := a.netModel.items[ref.itemID].LabeledItem
				mac, _ := net.ParseMAC(port.(api.Port).MAC)
				physIfs = append(physIfs, configitems.PhysIf{
					MAC:          mac,
					LogicalLabel: ref.logicalLabel,
				})
			case api.Bond{}.ItemType():
				bonds = append(bonds, a.bondIfName(ref.logicalLabel))
			}
		}
		intendedCfg.PutItem(configitems.Bridge{
			IfName:       a.bridgeIfName(bridge.LogicalLabel),
			LogicalLabel: bridge.LogicalLabel,
			PhysIfs:      physIfs,
			BondIfs:      bonds,
			VLANs:        vlans,
		}, nil)
	}
	return intendedCfg
}

func (a *agent) bondIfName(logicalLabel string) string {
	// TODO: make sure it is not longer than 15 characters
	return "bond-" + logicalLabel
}

func (a *agent) bridgeIfName(logicalLabel string) string {
	// TODO: make sure it is not longer than 15 characters
	return "br-" + logicalLabel
}
