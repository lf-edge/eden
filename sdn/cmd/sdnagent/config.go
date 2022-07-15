package main

import (
	"net"

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
			IfName:       netIf.IfName,
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
		if netIf, found := a.netMonitor.LookupInterfaceByMAC(mac); found {
			currentPhysIfs.PutItem(configitems.PhysIf{
				LogicalLabel: port.LogicalLabel,
				IfName:       netIf.IfName,
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
		PhysIfLL:  hostPortLogicalLabel,
		PhysIfMAC: netIf.MAC,
		Usage:     configitems.IfUsageL3,
		AdminUP:   true,
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
