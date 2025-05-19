package main

import (
	"fmt"
	"hash/fnv"
	"net"
	"strconv"
	"strings"
	"syscall"

	"github.com/lf-edge/eden/sdn/vm/api"
	"github.com/lf-edge/eden/sdn/vm/pkg/configitems"
	dg "github.com/lf-edge/eve/libs/depgraph"
	"github.com/lf-edge/eve/libs/reconciler"
	log "github.com/sirupsen/logrus"
)

const (
	// Dependency graph modeling current/intended network configuration.
	// *SG are names of sub-graphs.
	configGraphName    = "SDN-Config"
	physicalIfsSG      = "Physical-Interfaces"
	trafficControlSG   = "Traffic-Control"
	hostConnectivitySG = "Host-Connectivity"
	bridgesSG          = "Bridges"
	firewallSG         = "Firewall"
	networkSGPrefix    = "Network-"
	endpointSGPrefix   = "Endpoint-"

	// Iptables chain used to implement firewall rules.
	fwIptablesChain = "firewall"

	// ifNameMaxLen is a limit for interface names in the Linux kernel (IFNAMSIZ).
	ifNameMaxLen = 15

	// Priority for IP rules directing traffic to per-network routing tables.
	networkIPRulePriority = 500
	networkRTBaseIndex    = 500
)

var allIPv4, allIPv6 *net.IPNet

func init() {
	_, allIPv4, _ = net.ParseCIDR("0.0.0.0/0")
	_, allIPv6, _ = net.ParseCIDR("::/0")
}

// Update external items inside the graph with the current state.
func (a *agent) updateCurrentState() (changed bool) {
	if a.currentState == nil {
		graphArgs := dg.InitArgs{Name: configGraphName}
		a.currentState = dg.New(graphArgs)
		changed = true
	}
	currentPhysIfs := dg.New(dg.InitArgs{Name: physicalIfsSG})
	// Port connecting SDN VM with the host.
	if netIf, found := a.macLookup.GetInterfaceByMAC(hostPortMACPrefix, true); found {
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
		if _, found := a.macLookup.GetInterfaceByMAC(mac, false); found {
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
	a.allocNetworkIndexes()
	graphArgs := dg.InitArgs{Name: configGraphName}
	a.intendedState = dg.New(graphArgs)
	a.intendedState.PutSubGraph(a.getIntendedPhysIfs())
	a.intendedState.PutSubGraph(a.getIntendedHostConnectivity())
	a.intendedState.PutSubGraph(a.getIntendedTrafficControl())
	a.intendedState.PutSubGraph(a.getIntendedBridges())
	a.intendedState.PutSubGraph(a.getIntendedFirewall())
	for _, network := range a.netModel.Networks {
		a.intendedState.PutSubGraph(a.getIntendedNetwork(network))
	}
	for _, client := range a.netModel.Endpoints.Clients {
		a.intendedState.PutSubGraph(a.getIntendedClientEp(client))
	}
	for _, dnsSrv := range a.netModel.Endpoints.DNSServers {
		a.intendedState.PutSubGraph(a.getIntendedDNSSrvEp(dnsSrv))
	}
	for _, proxy := range a.netModel.Endpoints.ExplicitProxies {
		a.intendedState.PutSubGraph(a.getIntendedExProxyEp(proxy))
	}
	for _, proxy := range a.netModel.Endpoints.TransparentProxies {
		a.intendedState.PutSubGraph(a.getIntendedTProxyEp(proxy))
	}
	for _, httpSrv := range a.netModel.Endpoints.HTTPServers {
		a.intendedState.PutSubGraph(a.getIntendedHttpSrvEp(httpSrv))
	}

	//nolint:godox
	// TODO: ntp servers, netboot servers
}

func (a *agent) getIntendedPhysIfs() dg.Graph {
	graphArgs := dg.InitArgs{Name: physicalIfsSG}
	intendedCfg := dg.New(graphArgs)
	if netIf, found := a.macLookup.GetInterfaceByMAC(hostPortMACPrefix, true); found {
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
	netIf, found := a.macLookup.GetInterfaceByMAC(hostPortMACPrefix, true)
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
		MTU:     maxMTU,
	}, nil)
	intendedCfg.PutItem(configitems.Sysctl{
		EnableIPv4Forwarding:  true,
		EnableIPv6Forwarding:  true,
		BridgeNfCallIptables:  false,
		BridgeNfCallIp6tables: false,
	}, nil)
	intendedCfg.PutItem(configitems.DhcpClient{
		PhysIf: configitems.PhysIf{
			MAC:          netIf.MAC,
			LogicalLabel: hostPortLogicalLabel,
		},
		LogFile: "/run/dhcpcd.log",
	}, nil)
	intendedCfg.PutItem(configitems.IptablesChain{
		ChainName: "POSTROUTING",
		Table:     "nat",
		ForIPv6:   false,
		Rules: []configitems.IptablesRule{
			{
				Args:        []string{"-o", netIf.IfName, "-j", "MASQUERADE"},
				Description: "S-NAT traffic leaving SDN VM towards the host OS",
			},
		},
	}, nil)
	intendedCfg.PutItem(configitems.IptablesChain{
		ChainName: "POSTROUTING",
		Table:     "nat",
		ForIPv6:   true,
		Rules: []configitems.IptablesRule{
			{
				Args:        []string{"-o", netIf.IfName, "-j", "MASQUERADE"},
				Description: "S-NAT traffic leaving SDN VM towards the host OS",
			},
		},
	}, nil)
	return intendedCfg
}

func (a *agent) getIntendedTrafficControl() dg.Graph {
	graphArgs := dg.InitArgs{Name: trafficControlSG}
	intendedCfg := dg.New(graphArgs)
	emptyTC := api.TrafficControl{}
	for _, port := range a.netModel.Ports {
		if port.TC == emptyTC {
			continue
		}
		// MAC address is already validated
		mac, _ := net.ParseMAC(port.MAC)
		intendedCfg.PutItem(configitems.TrafficControl{
			TrafficControl: port.TC,
			PhysIf: configitems.PhysIf{
				LogicalLabel: port.LogicalLabel,
				MAC:          mac,
			},
		}, nil)
	}
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
			MTU:      maxMTU,
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
			MTU:               maxMTU,
		}, nil)
	}
	for _, bridge := range a.netModel.Bridges {
		vlans := make(map[uint16]struct{})
		labeledItem := a.netModel.items.getItem(api.Bridge{}.ItemType(), bridge.LogicalLabel)
		for refKey, refBy := range labeledItem.referencedBy {
			if strings.HasPrefix(refKey, api.NetworkBridgeRefPrefix) {
				network := a.netModel.items[refBy].LabeledItem
				if vlanID := network.(api.Network).VlanID; vlanID != 0 {
					vlans[vlanID] = struct{}{}
				}
			} else if strings.HasPrefix(refKey, api.EndpointBridgeRefPrefix) {
				endpoint := a.getEndpoint(refBy.logicalLabel)
				if vlanID := endpoint.DirectL2Connect.VlanID; vlanID != 0 {
					vlans[vlanID] = struct{}{}
				}
			}
		}
		var vlanList []uint16
		for vlanID := range vlans {
			vlanList = append(vlanList, vlanID)
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
			VLANs:        vlanList,
			MTU:          maxMTU,
			WithSTP:      bridge.WithSTP,
		}, nil)
	}
	return intendedCfg
}

func (a *agent) getIntendedNetwork(network api.Network) dg.Graph {
	graphArgs := dg.InitArgs{Name: networkSGPrefix + network.LogicalLabel}
	intendedCfg := dg.New(graphArgs)

	// Process IP configuration.
	ipv4Subnet, gwIPv4, dhcpv4 := a.getNetworkIPConf(network, false)
	ipv6Subnet, gwIPv6, dhcpv6 := a.getNetworkIPConf(network, true)
	gwIPs := filterOutNilAddrs(gwIPv4, gwIPv6)
	vethInIPv4, vethOutIPv4, vethInIPv6, vethOutIPv6 := a.getNetworkRtVethIPs(network)
	vethInIPs := filterOutNilAddrs(vethInIPv4, vethInIPv6)
	vethOutIPs := filterOutNilAddrs(vethOutIPv4, vethOutIPv6)
	var (
		ipv4DNSServers []net.IP
		ipv6DNSServers []net.IP
		allDNSServers  []net.IP
	)
	if ipv4Subnet != nil {
		for _, dnsServer := range dhcpv4.PublicDNS {
			ipv4DNSServers = append(ipv4DNSServers, net.ParseIP(dnsServer))
		}
		for _, dnsServer := range dhcpv4.PrivateDNS {
			ep := a.getEndpoint(dnsServer)
			epIPv4, _ := a.getEndpointIP(ep, false)
			if epIPv4 != nil {
				ipv4DNSServers = append(ipv4DNSServers, epIPv4)
			}
		}
	}
	if ipv6Subnet != nil {
		for _, dnsServer := range dhcpv6.PublicDNS {
			ipv6DNSServers = append(ipv6DNSServers, net.ParseIP(dnsServer))
		}
		for _, dnsServer := range dhcpv6.PrivateDNS {
			ep := a.getEndpoint(dnsServer)
			epIPv6, _ := a.getEndpointIP(ep, true)
			if epIPv6 != nil {
				ipv6DNSServers = append(ipv6DNSServers, epIPv6)
			}
		}
	}
	allDNSServers = append([]net.IP{}, ipv4DNSServers...)
	allDNSServers = append(allDNSServers, ipv6DNSServers...)

	// Network namespace connected with the bridge using veth.
	brVethName, brInIfName, brOutIfName := a.networkBrVethName(network.LogicalLabel)
	nsName := a.networkNsName(network.LogicalLabel)
	netNs := configitems.NetNamespace{
		NsName: nsName,
	}
	intendedCfg.PutItem(netNs, nil)
	intendedCfg.PutItem(configitems.Sysctl{
		NetNamespace:          nsName,
		EnableIPv4Forwarding:  true,
		EnableIPv6Forwarding:  true,
		BridgeNfCallIptables:  true,
		BridgeNfCallIp6tables: true,
	}, nil)
	intendedCfg.PutItem(configitems.Veth{
		VethName: brVethName,
		Peer1: configitems.VethPeer{
			IfName:       brInIfName,
			NetNamespace: nsName,
			IPAddresses:  gwIPs,
			MTU:          network.MTU,
		},
		Peer2: configitems.VethPeer{
			IfName:       brOutIfName,
			NetNamespace: configitems.MainNsName,
			MasterBridge: &configitems.MasterBridge{
				IfName: a.bridgeIfName(network.Bridge),
				VLAN:   network.VlanID,
			},
			MTU: network.MTU,
		},
	}, nil)

	// Another veth used to connect network with the main "router".
	rtVethName, rtInIfName, rtOutIfName := a.networkRtVethName(network.LogicalLabel)
	intendedCfg.PutItem(configitems.Veth{
		VethName: rtVethName,
		Peer1: configitems.VethPeer{
			IfName:       rtInIfName,
			NetNamespace: nsName,
			IPAddresses:  vethInIPs,
			MTU:          network.MTU,
		},
		Peer2: configitems.VethPeer{
			IfName:       rtOutIfName,
			NetNamespace: configitems.MainNsName,
			IPAddresses:  vethOutIPs,
			MTU:          network.MTU,
		},
	}, nil)

	// DHCP server.
	if dhcpv4.Enable || dhcpv6.Enable {
		dhcpIPv4Subnet := ipv4Subnet
		if !dhcpv4.Enable {
			dhcpIPv4Subnet = nil
		}
		dhcpIPv6Subnet := ipv6Subnet
		if !dhcpv6.Enable {
			dhcpIPv6Subnet = nil
		}
		var ipv4Range, ipv6Range configitems.IPRange
		if dhcpIPv4Subnet != nil {
			ipv4Range = a.subnetToHostIPRange(dhcpIPv4Subnet)
			if dhcpv4.IPRange.FromIP != "" {
				ipv4Range.FromIP = net.ParseIP(dhcpv4.IPRange.FromIP)
				ipv4Range.ToIP = net.ParseIP(dhcpv4.IPRange.ToIP)
			}
		}
		if dhcpIPv6Subnet != nil {
			ipv6Range = a.subnetToHostIPRange(dhcpIPv6Subnet)
			if dhcpv6.IPRange.FromIP != "" {
				ipv6Range.FromIP = net.ParseIP(dhcpv6.IPRange.FromIP)
				ipv6Range.ToIP = net.ParseIP(dhcpv6.IPRange.ToIP)
			}
		}
		ipv4NtpServer := dhcpv4.PublicNTP
		if dhcpv4.PrivateNTP != "" {
			ep := a.getEndpoint(dhcpv4.PrivateNTP)
			epIPv4, _ := a.getEndpointIP(ep, false)
			if epIPv4 != nil {
				ipv4NtpServer = epIPv4.String()
			}
		}
		ipv6NtpServer := dhcpv6.PublicNTP
		if dhcpv6.PrivateNTP != "" {
			ep := a.getEndpoint(dhcpv6.PrivateNTP)
			epIPv6, _ := a.getEndpointIP(ep, true)
			if epIPv6 != nil {
				ipv4NtpServer = epIPv6.String()
			}
		}
		var gatewayIPv4 net.IP
		if gwIPv4 != nil && !dhcpv4.WithoutDefaultRoute {
			gatewayIPv4 = gwIPv4.IP
		}
		var staticEntries []configitems.MACToIP
		if dhcpv4.Enable {
			for _, entry := range dhcpv4.StaticEntries {
				mac, _ := net.ParseMAC(entry.MAC)
				staticEntries = append(staticEntries, configitems.MACToIP{
					MAC: mac,
					IP:  net.ParseIP(entry.IP),
				})
			}
		}
		if dhcpv6.Enable {
			for _, entry := range dhcpv6.StaticEntries {
				mac, _ := net.ParseMAC(entry.MAC)
				staticEntries = append(staticEntries, configitems.MACToIP{
					MAC: mac,
					IP:  net.ParseIP(entry.IP),
				})
			}
		}
		// It is already validated that IPv4 and IPv6 config do not define different
		// domain names.
		var domainName string
		if dhcpv4.Enable && dhcpv4.DomainName != "" {
			domainName = dhcpv4.DomainName
		} else if dhcpv6.Enable {
			domainName = dhcpv6.DomainName
		}
		intendedCfg.PutItem(configitems.DhcpServer{
			ServerName:     network.LogicalLabel,
			NetNamespace:   nsName,
			VethName:       brVethName,
			VethPeerIfName: brInIfName,
			IPv4Subnet:     dhcpIPv4Subnet,
			IPv6Subnet:     dhcpIPv6Subnet,
			IPv4Range:      ipv4Range,
			IPv6Range:      ipv6Range,
			StaticEntries:  staticEntries,
			GatewayIPv4:    gatewayIPv4,
			DomainName:     domainName,
			DNSServers:     allDNSServers,
			IPv4NTPServer:  ipv4NtpServer,
			IPv6NTPServer:  ipv6NtpServer,
			WPAD:           dhcpv4.WPAD,
		}, nil)
	}

	// IPv6 router advertisement.
	if ipv6Subnet != nil {
		advManagedFlag := dhcpv6.Enable && dhcpv6.IPRange.FromIP != ""
		advAutonomous := !advManagedFlag
		advOtherConfigFlag := dhcpv6.Enable &&
			(dhcpv6.PrivateNTP != "" || dhcpv6.PublicNTP != "")
		advDNSServers := ipv6DNSServers
		if advManagedFlag || advOtherConfigFlag {
			// If DHCPv6 is being used, do not advertise DNS servers twice.
			advDNSServers = nil
		}
		intendedCfg.PutItem(configitems.Radvd{
			DaemonName:          network.LogicalLabel,
			NetNamespace:        nsName,
			VethName:            brVethName,
			VethPeerIfName:      brInIfName,
			Subnet:              ipv6Subnet,
			MTU:                 network.MTU,
			AdvManagedFlag:      advManagedFlag,
			AdvOtherConfigFlag:  advOtherConfigFlag,
			AdvAutonomous:       advAutonomous,
			DNSServers:          advDNSServers,
			WithoutDefaultRoute: dhcpv6.WithoutDefaultRoute,
		}, nil)
	}

	// Transparent proxy.
	if network.TransparentProxy != "" {
		ep := a.getEndpoint(network.TransparentProxy)
		httpsPorts := []api.ProxyPort{{Port: 443}}
		controllerPort := a.netModel.Host.ControllerPort
		if controllerPort != 443 {
			httpsPorts = append(httpsPorts, api.ProxyPort{Port: controllerPort})
		}
		epIPv4, _ := a.getEndpointIP(ep, false)
		epIPv6, _ := a.getEndpointIP(ep, true)
		if epIPv4 != nil {
			intendedCfg.PutItem(
				a.getIptablesChainForTranspProxy(nsName, epIPv4, httpsPorts), nil)
		}
		if epIPv6 != nil {
			intendedCfg.PutItem(
				a.getIptablesChainForTranspProxy(nsName, epIPv6, httpsPorts), nil)
		}
	}

	// When user is accessing EVE using "sdn fwd" command, the source IP
	// is from the internal IP subnet.
	// Make sure that the IP address is S-NATed before sending packets to EVE.
	// Otherwise, the responses could be routed out via wrong EVE network ports.
	intendedCfg.PutItem(configitems.IptablesChain{
		NetNamespace: nsName,
		ChainName:    "POSTROUTING",
		Table:        "nat",
		ForIPv6:      false,
		RefersVeths:  []string{rtVethName},
		Rules: []configitems.IptablesRule{
			{
				Args: []string{"-o", brInIfName, "-s", internalIPv4Subnet.String(),
					"-j", "MASQUERADE"},
				Description: "S-NAT traffic leaving SDN VM towards EVE with internal source IP",
			},
		},
	}, nil)
	intendedCfg.PutItem(configitems.IptablesChain{
		NetNamespace: nsName,
		ChainName:    "POSTROUTING",
		Table:        "nat",
		ForIPv6:      true,
		RefersVeths:  []string{rtVethName},
		Rules: []configitems.IptablesRule{
			{
				Args: []string{"-o", brInIfName, "-s", internalIPv6Subnet.String(),
					"-j", "MASQUERADE"},
				Description: "S-NAT traffic leaving SDN VM towards EVE with internal source IP",
			},
		},
	}, nil)

	// Add routing configuration.
	a.getIntendedNetworkRouting(network, intendedCfg)
	return intendedCfg
}

func (a *agent) getIntendedNetworkRouting(network api.Network, intendedCfg dg.Graph) {
	nsName := a.networkNsName(network.LogicalLabel)
	index, hasIndex := a.networkIndex[network.LogicalLabel]
	if !hasIndex {
		log.Fatalf("missing index for network %s", network.LogicalLabel)
	}
	rt := networkRTBaseIndex + index

	ipv4Subnet, _, _ := a.getNetworkIPConf(network, false)
	ipv6Subnet, _, _ := a.getNetworkIPConf(network, true)
	rtVethName, rtInIfName, rtOutIfName := a.networkRtVethName(network.LogicalLabel)
	brVethName, brInIfName, _ := a.networkBrVethName(network.LogicalLabel)
	vethInIPv4, vethOutIPv4, vethInIPv6, vethOutIPv6 := a.getNetworkRtVethIPs(network)

	if ipv4Subnet != nil {
		intendedCfg.PutItem(configitems.IPRule{
			SrcNet:   ipv4Subnet,
			Table:    rt,
			Priority: networkIPRulePriority,
		}, nil)
		intendedCfg.PutItem(configitems.IPRule{
			DstNet:   ipv4Subnet,
			Table:    rt,
			Priority: networkIPRulePriority,
		}, nil)
	}
	if ipv6Subnet != nil {
		intendedCfg.PutItem(configitems.IPRule{
			SrcNet:   ipv6Subnet,
			Table:    rt,
			Priority: networkIPRulePriority,
		}, nil)
		intendedCfg.PutItem(configitems.IPRule{
			DstNet:   ipv6Subnet,
			Table:    rt,
			Priority: networkIPRulePriority,
		}, nil)
	}
	// - default route from inside the network namespace
	if ipv4Subnet != nil {
		intendedCfg.PutItem(configitems.Route{
			NetNamespace: nsName,
			Table:        syscall.RT_TABLE_MAIN,
			DstNet:       allIPv4,
			OutputIf: configitems.RouteOutIf{
				VethName:       rtVethName,
				VethPeerIfName: rtInIfName,
			},
			GwIP: vethOutIPv4.IP,
		}, nil)
	}
	if ipv6Subnet != nil {
		intendedCfg.PutItem(configitems.Route{
			NetNamespace: nsName,
			Table:        syscall.RT_TABLE_MAIN,
			DstNet:       allIPv6,
			OutputIf: configitems.RouteOutIf{
				VethName:       rtVethName,
				VethPeerIfName: rtInIfName,
			},
			GwIP: vethOutIPv6.IP,
		}, nil)
	}
	// - route for every L3-connected endpoint
	epTypename := api.Endpoint{}.ItemType()
	for itemID, item := range a.netModel.items {
		if itemID.typename != epTypename {
			continue
		}
		ep := a.labeledItemToEndpoint(item)
		if ep.DirectL2Connect.Bridge != "" {
			// This endpoint has direct L2 connection to EVE, skip.
			continue
		}
		epIPv4, epIPv4Subnet := a.getEndpointIP(ep, false)
		epIPv6, epIPv6Subnet := a.getEndpointIP(ep, true)
		reachable := network.Router == nil ||
			strListContains(network.Router.ReachableEndpoints, ep.LogicalLabel)
		if reachable {
			epVethName, _, epOutIfName := a.endpointVethName(ep.LogicalLabel)
			if epIPv4Subnet != nil {
				intendedCfg.PutItem(configitems.Route{
					NetNamespace: configitems.MainNsName,
					Table:        rt,
					DstNet:       epIPv4Subnet,
					OutputIf: configitems.RouteOutIf{
						VethName:       epVethName,
						VethPeerIfName: epOutIfName,
					},
					GwIP: epIPv4,
				}, nil)
			}
			if epIPv6Subnet != nil {
				intendedCfg.PutItem(configitems.Route{
					NetNamespace: configitems.MainNsName,
					Table:        rt,
					DstNet:       epIPv6Subnet,
					OutputIf: configitems.RouteOutIf{
						VethName:       epVethName,
						VethPeerIfName: epOutIfName,
					},
					GwIP: epIPv6,
				}, nil)
			}
		} else {
			if epIPv4Subnet != nil {
				intendedCfg.PutItem(configitems.Route{
					NetNamespace: configitems.MainNsName,
					Table:        rt,
					DstNet:       epIPv4Subnet,
				}, nil)
			}
			if epIPv6Subnet != nil {
				intendedCfg.PutItem(configitems.Route{
					NetNamespace: configitems.MainNsName,
					Table:        rt,
					DstNet:       epIPv6Subnet,
				}, nil)
			}
		}
	}
	// - route for every other network (including itself)
	for _, network2 := range a.netModel.Networks {
		net2IPv4Subnet, _, _ := a.getNetworkIPConf(network2, false)
		net2IPv6Subnet, _, _ := a.getNetworkIPConf(network2, true)
		reachable := network.Router == nil ||
			network2.LogicalLabel == network.LogicalLabel ||
			strListContains(network.Router.ReachableNetworks, network2.LogicalLabel)
		if reachable {
			net2VethName, _, net2OutIfName := a.networkRtVethName(network2.LogicalLabel)
			if net2IPv4Subnet != nil {
				net2VethInIPv4, _ := a.genVethIPsForNetwork(network2.LogicalLabel, false)
				intendedCfg.PutItem(configitems.Route{
					NetNamespace: configitems.MainNsName,
					Table:        rt,
					DstNet:       net2IPv4Subnet,
					OutputIf: configitems.RouteOutIf{
						VethName:       net2VethName,
						VethPeerIfName: net2OutIfName,
					},
					GwIP: net2VethInIPv4.IP,
				}, nil)
			}
			if net2IPv6Subnet != nil {
				net2VethInIPv6, _ := a.genVethIPsForNetwork(network2.LogicalLabel, true)
				intendedCfg.PutItem(configitems.Route{
					NetNamespace: configitems.MainNsName,
					Table:        rt,
					DstNet:       net2IPv6Subnet,
					OutputIf: configitems.RouteOutIf{
						VethName:       net2VethName,
						VethPeerIfName: net2OutIfName,
					},
					GwIP: net2VethInIPv6.IP,
				}, nil)
			}
		} else {
			if net2IPv4Subnet != nil {
				intendedCfg.PutItem(configitems.Route{
					NetNamespace: configitems.MainNsName,
					Table:        rt,
					DstNet:       net2IPv4Subnet,
				}, nil)
			}
			if net2IPv6Subnet != nil {
				intendedCfg.PutItem(configitems.Route{
					NetNamespace: configitems.MainNsName,
					Table:        rt,
					DstNet:       net2IPv6Subnet,
				}, nil)
			}
		}
	}
	// - route for the outside world if enabled
	outsideRechability := network.Router == nil || network.Router.OutsideReachability
	hostPort, hostPortfound := a.macLookup.GetInterfaceByMAC(hostPortMACPrefix, true)
	if outsideRechability && hostPortfound {
		hostGwIPv4 := a.getHostGwIP(false)
		if hostGwIPv4 != nil {
			intendedCfg.PutItem(configitems.Route{
				NetNamespace: configitems.MainNsName,
				Table:        rt,
				DstNet:       allIPv4,
				OutputIf: configitems.RouteOutIf{
					PhysIf: configitems.PhysIf{
						MAC:          hostPort.MAC,
						LogicalLabel: hostPortLogicalLabel,
					},
				},
				GwIP: hostGwIPv4,
			}, nil)
		}
		hostGwIPv6 := a.getHostGwIP(true)
		if hostGwIPv6 != nil {
			intendedCfg.PutItem(configitems.Route{
				NetNamespace: configitems.MainNsName,
				Table:        rt,
				DstNet:       allIPv6,
				OutputIf: configitems.RouteOutIf{
					PhysIf: configitems.PhysIf{
						MAC:          hostPort.MAC,
						LogicalLabel: hostPortLogicalLabel,
					},
				},
				GwIP: hostGwIPv6,
			}, nil)
		}
	}
	// - routes towards EVE
	var routesTowardsEVE []api.IPRoute
	if network.Router != nil {
		routesTowardsEVE = network.Router.RoutesTowardsEVE
	}
	for _, route := range routesTowardsEVE {
		_, dstNetwork, _ := net.ParseCIDR(route.DstNetwork)
		ipv4 := dstNetwork.IP.To4() != nil
		gatewayIP := net.ParseIP(route.Gateway)
		intendedCfg.PutItem(configitems.IPRule{
			SrcNet:   dstNetwork,
			Table:    rt,
			Priority: networkIPRulePriority,
		}, nil)
		intendedCfg.PutItem(configitems.IPRule{
			DstNet:   dstNetwork,
			Table:    rt,
			Priority: networkIPRulePriority,
		}, nil)
		intendedCfg.PutItem(configitems.Route{
			NetNamespace: nsName,
			Table:        syscall.RT_TABLE_MAIN,
			DstNet:       dstNetwork,
			OutputIf: configitems.RouteOutIf{
				VethName:       brVethName,
				VethPeerIfName: brInIfName,
			},
			GwIP: gatewayIP,
		}, nil)
		if ipv4 && ipv4Subnet != nil {
			intendedCfg.PutItem(configitems.Route{
				NetNamespace: configitems.MainNsName,
				Table:        rt,
				DstNet:       dstNetwork,
				OutputIf: configitems.RouteOutIf{
					VethName:       rtVethName,
					VethPeerIfName: rtOutIfName,
				},
				GwIP: vethInIPv4.IP,
			}, nil)
		}
		if !ipv4 && ipv6Subnet != nil {
			intendedCfg.PutItem(configitems.Route{
				NetNamespace: configitems.MainNsName,
				Table:        rt,
				DstNet:       dstNetwork,
				OutputIf: configitems.RouteOutIf{
					VethName:       rtVethName,
					VethPeerIfName: rtOutIfName,
				},
				GwIP: vethInIPv6.IP,
			}, nil)
		}
	}
	// - everything else is unreachable
	intendedCfg.PutItem(configitems.Route{
		NetNamespace: configitems.MainNsName,
		Table:        rt,
		DstNet:       allIPv4,
		Metric:       ^uint32(0), // Lowest prio.
	}, nil)
	intendedCfg.PutItem(configitems.Route{
		NetNamespace: configitems.MainNsName,
		Table:        rt,
		DstNet:       allIPv6,
		Metric:       ^uint32(0), // Lowest prio.
	}, nil)
}

// Returns chain with iptables rules to transparently redirect traffic into a proxy.
func (a *agent) getIptablesChainForTranspProxy(
	nsName string, epIP net.IP, httpsPorts []api.ProxyPort) configitems.IptablesChain {
	dnatRules := []configitems.IptablesRule{
		{
			Args: []string{"-p", "tcp", "--dport", "80", "-j", "DNAT",
				"--to-destination", epIP.String()},
			Description: "Send HTTP traffic into the proxy",
		},
	}
	for _, httpsPort := range httpsPorts {
		dnatRules = append(dnatRules, configitems.IptablesRule{
			Args: []string{"-p", "tcp", "--dport", strconv.Itoa(int(httpsPort.Port)),
				"-j", "DNAT", "--to-destination", epIP.String()},
			Description: fmt.Sprintf("Send HTTPS traffic (port %d) into the proxy",
				httpsPort.Port),
		})
	}
	return configitems.IptablesChain{
		NetNamespace: nsName,
		ChainName:    "PREROUTING",
		Table:        "nat",
		ForIPv6:      false,
		Rules:        dnatRules,
	}
}

func (a *agent) getIntendedFirewall() dg.Graph {
	graphArgs := dg.InitArgs{Name: firewallSG}
	intendedCfg := dg.New(graphArgs)
	iptablesRules := make([]configitems.IptablesRule, 0, 2+len(a.netModel.Firewall.Rules))
	ip6tablesRules := make([]configitems.IptablesRule, 0, 2+len(a.netModel.Firewall.Rules))
	// Allow any subsequent traffic that results from an already allowed connection.
	matchAlreadyAllowed := configitems.IptablesRule{
		Args: []string{"-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
	}
	iptablesRules = append(iptablesRules, matchAlreadyAllowed)
	ip6tablesRules = append(ip6tablesRules, matchAlreadyAllowed)
	// Add explicitly configured firewall rules.
	for _, rule := range a.netModel.Firewall.Rules {
		iptablesRule, ip6tablesRule := a.getIntendedFwRule(rule)
		if len(iptablesRule.Args) != 0 {
			iptablesRules = append(iptablesRules, iptablesRule)
		}
		if len(ip6tablesRule.Args) != 0 {
			ip6tablesRules = append(ip6tablesRules, ip6tablesRule)
		}
	}
	// Implicitly allow everything not matched by the rules above.
	allowTheRest := api.FwRule{Action: api.FwAllow}
	iptablesRule, ip6tablesRule := a.getIntendedFwRule(allowTheRest)
	iptablesRules = append(iptablesRules, iptablesRule)
	ip6tablesRules = append(ip6tablesRules, ip6tablesRule)
	intendedCfg.PutItem(configitems.IptablesChain{
		NetNamespace: configitems.MainNsName,
		ChainName:    fwIptablesChain,
		Table:        "filter",
		ForIPv6:      false,
		Rules:        iptablesRules,
	}, nil)
	intendedCfg.PutItem(configitems.IptablesChain{
		NetNamespace: configitems.MainNsName,
		ChainName:    fwIptablesChain,
		Table:        "filter",
		ForIPv6:      true,
		Rules:        ip6tablesRules,
	}, nil)
	// Link the firewall chain with every network and endpoint (outside) interface.
	veths := make([]string, 0, len(a.netModel.Networks)+len(a.netModel.Endpoints.GetAll()))
	iptablesRules = nil
	for _, network := range a.netModel.Networks {
		rtVethName, _, rtOutIfName := a.networkRtVethName(network.LogicalLabel)
		veths = append(veths, rtVethName)
		iptablesRules = append(iptablesRules, configitems.IptablesRule{
			Args: []string{"-i", rtOutIfName, "-j", fwIptablesChain},
		})
	}
	for _, ep := range a.netModel.Endpoints.GetAll() {
		epVethName, _, epOutIfName := a.endpointVethName(ep.LogicalLabel)
		veths = append(veths, epVethName)
		iptablesRules = append(iptablesRules, configitems.IptablesRule{
			Args: []string{"-i", epOutIfName, "-j", fwIptablesChain},
		})
	}
	intendedCfg.PutItem(configitems.IptablesChain{
		NetNamespace: configitems.MainNsName,
		ChainName:    "FORWARD",
		Table:        "filter",
		ForIPv6:      false,
		Rules:        iptablesRules,
		RefersVeths:  veths,
		RefersChains: []string{fwIptablesChain},
	}, nil)
	intendedCfg.PutItem(configitems.IptablesChain{
		NetNamespace: configitems.MainNsName,
		ChainName:    "FORWARD",
		Table:        "filter",
		ForIPv6:      true,
		Rules:        iptablesRules,
		RefersVeths:  veths,
		RefersChains: []string{fwIptablesChain},
	}, nil)
	return intendedCfg
}

func (a *agent) getIntendedFwRule(
	rule api.FwRule) (iptablesRule, ip6tablesRule configitems.IptablesRule) {
	var ipv4RuleArgs, ipv6RuleArgs []string
	ipv4 := true
	ipv6 := true
	if rule.SrcSubnet != "" {
		_, subnet, _ := net.ParseCIDR(rule.SrcSubnet)
		if subnet.IP.To4() != nil {
			ipv4RuleArgs = append(ipv4RuleArgs, "-s", rule.SrcSubnet)
			ipv6 = false
		} else {
			ipv6RuleArgs = append(ipv6RuleArgs, "-s", rule.SrcSubnet)
			ipv4 = false
		}
	}
	if rule.DstSubnet != "" {
		_, subnet, _ := net.ParseCIDR(rule.DstSubnet)
		if subnet.IP.To4() != nil {
			ipv4RuleArgs = append(ipv4RuleArgs, "-d", rule.DstSubnet)
			ipv6 = false
		} else {
			ipv6RuleArgs = append(ipv6RuleArgs, "-d", rule.DstSubnet)
			ipv4 = false
		}
	}
	switch rule.Protocol {
	case api.AnyProto:
		ipv4RuleArgs = append(ipv4RuleArgs, "-p", "all")
		ipv6RuleArgs = append(ipv6RuleArgs, "-p", "all")
	case api.ICMP:
		ipv4RuleArgs = append(ipv4RuleArgs, "-p", "icmp")
		ipv6RuleArgs = append(ipv6RuleArgs, "-p", "icmpv6")
	case api.TCP:
		ipv4RuleArgs = append(ipv4RuleArgs, "-p", "tcp")
		ipv6RuleArgs = append(ipv6RuleArgs, "-p", "tcp")
	case api.UDP:
		ipv4RuleArgs = append(ipv4RuleArgs, "-p", "udp")
		ipv6RuleArgs = append(ipv6RuleArgs, "-p", "udp")
	}
	if len(rule.Ports) > 0 {
		var ports []string
		for _, port := range rule.Ports {
			ports = append(ports, strconv.Itoa(int(port)))
		}
		matchPorts := []string{"--match", "multiport", "--dport", strings.Join(ports, ",")}
		ipv4RuleArgs = append(ipv4RuleArgs, matchPorts...)
		ipv6RuleArgs = append(ipv6RuleArgs, matchPorts...)
	}
	switch rule.Action {
	case api.FwAllow:
		ipv4RuleArgs = append(ipv4RuleArgs, "-j", "ACCEPT")
		ipv6RuleArgs = append(ipv6RuleArgs, "-j", "ACCEPT")
	case api.FwReject:
		ipv4RuleArgs = append(ipv4RuleArgs, "-j", "REJECT")
		ipv6RuleArgs = append(ipv6RuleArgs, "-j", "REJECT")
	case api.FwDrop:
		ipv4RuleArgs = append(ipv4RuleArgs, "-j", "DROP")
		ipv6RuleArgs = append(ipv6RuleArgs, "-j", "DROP")
	}
	if ipv4 {
		iptablesRule = configitems.IptablesRule{
			Args: ipv4RuleArgs,
		}
	}
	if ipv6 {
		ip6tablesRule = configitems.IptablesRule{
			Args: ipv6RuleArgs,
		}
	}
	return iptablesRule, ip6tablesRule
}

func (a *agent) getIntendedClientEp(client api.Client) dg.Graph {
	graphArgs := dg.InitArgs{Name: endpointSGPrefix + client.LogicalLabel}
	intendedCfg := dg.New(graphArgs)
	a.putEpCommonConfig(intendedCfg, client.Endpoint, nil)
	// Nothing running inside...
	return intendedCfg
}

func (a *agent) getIntendedDNSSrvEp(dnsSrv api.DNSServer) dg.Graph {
	graphArgs := dg.InitArgs{Name: endpointSGPrefix + dnsSrv.LogicalLabel}
	intendedCfg := dg.New(graphArgs)
	a.putEpCommonConfig(intendedCfg, dnsSrv.Endpoint, nil)
	var (
		upstreamServers []net.IP
		staticEntries   []configitems.DnsEntry
	)
	nsName := a.endpointNsName(dnsSrv.LogicalLabel)
	vethName, inIfName, _ := a.endpointVethName(dnsSrv.LogicalLabel)
	for _, upstreamServer := range dnsSrv.UpstreamServers {
		upstreamServers = append(upstreamServers, net.ParseIP(upstreamServer))
	}
	for _, staticEntry := range dnsSrv.StaticEntries {
		var fqdn string
		var ipv4, ipv6 net.IP
		switch {
		case strings.HasPrefix(staticEntry.FQDN, api.EndpointFQDNRefPrefix):
			epLL := strings.TrimPrefix(staticEntry.FQDN, api.EndpointFQDNRefPrefix)
			ep := a.getEndpoint(epLL)
			fqdn = ep.FQDN
		default:
			fqdn = staticEntry.FQDN
		}
		switch {
		case staticEntry.IP == api.AdamIPRef:
			if a.netModel.hostIPv4 != nil {
				ipv4 = a.netModel.hostIPv4
			}
			if a.netModel.hostIPv6 != nil {
				ipv6 = a.netModel.hostIPv6
			}
		case staticEntry.IP == api.AdamIPv4Ref:
			ipv4 = a.netModel.hostIPv4
		case staticEntry.IP == api.AdamIPv6Ref:
			ipv6 = a.netModel.hostIPv6
		case strings.HasPrefix(staticEntry.IP, api.EndpointIPRefPrefix):
			epLL := strings.TrimPrefix(staticEntry.IP, api.EndpointIPRefPrefix)
			ep := a.getEndpoint(epLL)
			ipv4, _ = a.getEndpointIP(ep, false)
			ipv6, _ = a.getEndpointIP(ep, true)
		default:
			ip := net.ParseIP(staticEntry.IP)
			if ip != nil {
				if ip.To4() == nil {
					ipv6 = ip
				} else {
					ipv4 = ip
				}
			}
		}
		if ipv4 != nil {
			staticEntries = append(staticEntries, configitems.DnsEntry{
				FQDN: fqdn,
				IP:   ipv4,
			})
		}
		if ipv6 != nil {
			staticEntries = append(staticEntries, configitems.DnsEntry{
				FQDN: fqdn,
				IP:   ipv6,
			})
		}
	}
	intendedCfg.PutItem(configitems.DnsServer{
		ServerName:      dnsSrv.LogicalLabel,
		NetNamespace:    nsName,
		VethName:        vethName,
		VethPeerIfName:  inIfName,
		StaticEntries:   staticEntries,
		UpstreamServers: upstreamServers,
	}, nil)
	return intendedCfg
}

func (a *agent) getIntendedExProxyEp(proxy api.ExplicitProxy) dg.Graph {
	graphArgs := dg.InitArgs{Name: endpointSGPrefix + proxy.LogicalLabel}
	intendedCfg := dg.New(graphArgs)
	a.putEpCommonConfig(intendedCfg, proxy.Endpoint, &proxy.DNSClientConfig)
	nsName := a.endpointNsName(proxy.LogicalLabel)
	vethName, _, _ := a.endpointVethName(proxy.LogicalLabel)
	epIPs := a.getEndpointAllIPs(proxy.Endpoint)
	var httpsPorts []api.ProxyPort
	if proxy.HTTPSProxy.Port != 0 {
		httpsPorts = append(httpsPorts, proxy.HTTPSProxy)
	}
	httpPort := proxy.HTTPProxy
	intendedCfg.PutItem(configitems.HttpProxy{
		Proxy:        proxy.Proxy,
		ProxyName:    proxy.LogicalLabel,
		NetNamespace: nsName,
		VethName:     vethName,
		ListenIPs:    epIPs,
		Hostname:     proxy.FQDN,
		HTTPPort:     httpPort,
		HTTPSPorts:   httpsPorts,
		Users:        proxy.Users,
	}, nil)
	return intendedCfg
}

func (a *agent) getIntendedTProxyEp(proxy api.TransparentProxy) dg.Graph {
	graphArgs := dg.InitArgs{Name: endpointSGPrefix + proxy.LogicalLabel}
	intendedCfg := dg.New(graphArgs)
	a.putEpCommonConfig(intendedCfg, proxy.Endpoint, &proxy.DNSClientConfig)
	nsName := a.endpointNsName(proxy.LogicalLabel)
	vethName, _, _ := a.endpointVethName(proxy.LogicalLabel)
	epIPs := a.getEndpointAllIPs(proxy.Endpoint)
	httpsPorts := []api.ProxyPort{{Port: 443}}
	controllerPort := a.netModel.Host.ControllerPort
	if controllerPort != 443 {
		httpsPorts = append(httpsPorts, api.ProxyPort{Port: controllerPort})
	}
	intendedCfg.PutItem(configitems.HttpProxy{
		Proxy:        proxy.Proxy,
		ProxyName:    proxy.LogicalLabel,
		NetNamespace: nsName,
		VethName:     vethName,
		ListenIPs:    epIPs,
		HTTPPort:     api.ProxyPort{Port: 80},
		HTTPSPorts:   httpsPorts,
		Transparent:  true,
	}, nil)
	return intendedCfg
}

func (a *agent) getIntendedHttpSrvEp(httpSrv api.HTTPServer) dg.Graph {
	graphArgs := dg.InitArgs{Name: endpointSGPrefix + httpSrv.LogicalLabel}
	intendedCfg := dg.New(graphArgs)
	a.putEpCommonConfig(intendedCfg, httpSrv.Endpoint, &httpSrv.DNSClientConfig)
	nsName := a.endpointNsName(httpSrv.LogicalLabel)
	vethName, _, _ := a.endpointVethName(httpSrv.LogicalLabel)
	epIPs := a.getEndpointAllIPs(httpSrv.Endpoint)
	intendedCfg.PutItem(configitems.HttpServer{
		ServerName:   httpSrv.LogicalLabel,
		NetNamespace: nsName,
		VethName:     vethName,
		ListenIPs:    epIPs,
		HTTPPort:     httpSrv.HTTPPort,
		HTTPSPort:    httpSrv.HTTPSPort,
		CertPEM:      httpSrv.CertPEM,
		KeyPEM:       httpSrv.KeyPEM,
		Paths:        httpSrv.Paths,
	}, nil)
	return intendedCfg
}

func (a *agent) putEpCommonConfig(graph dg.Graph, ep api.Endpoint, dnsClient *api.DNSClientConfig) {
	vethName, inIfName, outIfName := a.endpointVethName(ep.LogicalLabel)
	nsName := a.endpointNsName(ep.LogicalLabel)
	netNs := configitems.NetNamespace{
		NsName: nsName,
	}
	if dnsClient != nil {
		var dnsServers []net.IP
		for _, dnsServer := range dnsClient.PublicDNS {
			dnsServers = append(dnsServers, net.ParseIP(dnsServer))
		}
		for _, dnsServer := range dnsClient.PrivateDNS {
			ep := a.getEndpoint(dnsServer)
			serverIPv4, _ := a.getEndpointIP(ep, false)
			if serverIPv4 != nil {
				dnsServers = append(dnsServers, serverIPv4)
			}
			serverIPv6, _ := a.getEndpointIP(ep, true)
			if serverIPv6 != nil {
				dnsServers = append(dnsServers, serverIPv6)
			}
		}
		netNs.ResolvConf = configitems.ResolvConf{
			Create:     true,
			DNSServers: dnsServers,
		}
	}
	graph.PutItem(netNs, nil)
	graph.PutItem(configitems.Sysctl{
		NetNamespace:          nsName,
		EnableIPv4Forwarding:  true,
		EnableIPv6Forwarding:  true,
		BridgeNfCallIptables:  true,
		BridgeNfCallIp6tables: true,
	}, nil)
	// Prepare IP config.
	l2Direct := ep.DirectL2Connect.Bridge != ""
	epIPv4, ipv4Subnet := a.getEndpointIP(ep, false)
	epIPv6, ipv6Subnet := a.getEndpointIP(ep, true)
	var epIPs []*net.IPNet
	if epIPv4 != nil {
		epIPs = append(epIPs, &net.IPNet{IP: epIPv4, Mask: ipv4Subnet.Mask})
	}
	if epIPv6 != nil {
		epIPs = append(epIPs, &net.IPNet{IP: epIPv6, Mask: ipv6Subnet.Mask})
	}
	var gwIPv4, gwIPv6 *net.IPNet
	var gwIPs []*net.IPNet
	if !l2Direct && ipv4Subnet != nil {
		gwIPv4 = a.genEndpointGwIP(ipv4Subnet, epIPv4)
		gwIPs = append(gwIPs, gwIPv4)
	}
	if !l2Direct && ipv6Subnet != nil {
		gwIPv6 = a.genEndpointGwIP(ipv6Subnet, epIPv6)
		gwIPs = append(gwIPs, gwIPv6)
	}
	// Connect endpoint using a VETH.
	var masterBridge *configitems.MasterBridge
	if l2Direct {
		masterBridge = &configitems.MasterBridge{
			IfName: a.bridgeIfName(ep.DirectL2Connect.Bridge),
			VLAN:   ep.DirectL2Connect.VlanID,
		}
	}
	graph.PutItem(configitems.Veth{
		VethName: vethName,
		Peer1: configitems.VethPeer{
			IfName:       inIfName,
			NetNamespace: nsName,
			IPAddresses:  epIPs,
			MTU:          ep.MTU,
		},
		Peer2: configitems.VethPeer{
			IfName:       outIfName,
			NetNamespace: configitems.MainNsName,
			IPAddresses:  gwIPs,
			MTU:          ep.MTU,
			MasterBridge: masterBridge,
		},
	}, nil)
	// Configure default route(s).
	if !l2Direct && ipv4Subnet != nil {
		graph.PutItem(configitems.Route{
			NetNamespace: nsName,
			DstNet:       allIPv4,
			OutputIf: configitems.RouteOutIf{
				VethName:       vethName,
				VethPeerIfName: inIfName,
			},
			GwIP: gwIPv4.IP,
		}, nil)
	}
	if !l2Direct && ipv6Subnet != nil {
		graph.PutItem(configitems.Route{
			NetNamespace: nsName,
			DstNet:       allIPv6,
			OutputIf: configitems.RouteOutIf{
				VethName:       vethName,
				VethPeerIfName: inIfName,
			},
			GwIP: gwIPv6.IP,
		}, nil)
	}
}

func (a *agent) bondIfName(logicalLabel string) string {
	return a.genIfName("bond-", logicalLabel)
}

func (a *agent) bridgeIfName(logicalLabel string) string {
	return a.genIfName("br-", logicalLabel)
}

func (a *agent) networkNsName(logicalLabel string) string {
	return "network-" + logicalLabel
}

func (a *agent) endpointNsName(logicalLabel string) string {
	return "endpoint-" + logicalLabel
}

func (a *agent) networkBrVethName(logicalLabel string) (
	vethName, inIfName, outIfName string) {
	vethName = "net-br-" + logicalLabel
	inIfName = a.genIfName("net-br-in-", logicalLabel)
	outIfName = a.genIfName("net-br-out-", logicalLabel)
	return
}

func (a *agent) networkRtVethName(logicalLabel string) (
	vethName, inIfName, outIfName string) {
	vethName = "net-rt-" + logicalLabel
	inIfName = a.genIfName("net-rt-in-", logicalLabel)
	outIfName = a.genIfName("net-rt-out-", logicalLabel)
	return
}

func (a *agent) endpointVethName(logicalLabel string) (
	vethName, inIfName, outIfName string) {
	vethName = "ep-" + logicalLabel
	inIfName = a.genIfName("ep-in-", logicalLabel)
	outIfName = a.genIfName("ep-out-", logicalLabel)
	return
}

func (a *agent) getNetwork(logicalLabel string) api.Network {
	item := a.netModel.items.getItem(api.Network{}.ItemType(), logicalLabel)
	return item.LabeledItem.(api.Network)
}

func (a *agent) getEndpoint(logicalLabel string) api.Endpoint {
	item := a.netModel.items.getItem(api.Endpoint{}.ItemType(), logicalLabel)
	return a.labeledItemToEndpoint(item)
}

func (a *agent) getNetworkIPConf(
	network api.Network, forIPv6 bool) (subnet, gwIP *net.IPNet, dhcpConf api.DHCP) {
	if network.IsDualStack() {
		if forIPv6 {
			_, subnet, _ = net.ParseCIDR(network.DualStack.IPv6.Subnet)
			gwIP = &net.IPNet{IP: net.ParseIP(network.DualStack.IPv6.GwIP),
				Mask: subnet.Mask}
			dhcpConf = network.DualStack.IPv6.DHCP
			return
		}
		_, subnet, _ = net.ParseCIDR(network.DualStack.IPv4.Subnet)
		gwIP = &net.IPNet{IP: net.ParseIP(network.DualStack.IPv4.GwIP),
			Mask: subnet.Mask}
		dhcpConf = network.DualStack.IPv4.DHCP
		return
	}
	// IPv4 or IPv6 single-stack
	_, subnet, _ = net.ParseCIDR(network.Subnet)
	isIPv6 := subnet.IP.To4() == nil
	if forIPv6 != isIPv6 {
		return nil, nil, api.DHCP{}
	}
	gwIP = &net.IPNet{IP: net.ParseIP(network.GwIP), Mask: subnet.Mask}
	dhcpConf = network.DHCP
	return
}

// Returns IP addresses assigned to VETHs connecting Network with the Router namespace.
func (a *agent) getNetworkRtVethIPs(
	network api.Network) (vethInIPv4, vethOutIPv4, vethInIPv6, vethOutIPv6 *net.IPNet) {
	if network.HasIPv4Subnet() {
		vethInIPv4, vethOutIPv4 = a.genVethIPsForNetwork(network.LogicalLabel, false)
	}
	if network.HasIPv6Subnet() {
		vethInIPv6, vethOutIPv6 = a.genVethIPsForNetwork(network.LogicalLabel, true)
	}
	return
}

func (a *agent) getEndpointIP(
	ep api.Endpoint, forIPv6 bool) (ip net.IP, subnet *net.IPNet) {
	if ep.IsDualStack() {
		if forIPv6 {
			ip = net.ParseIP(ep.DualStack.IPv6.IP)
			_, subnet, _ = net.ParseCIDR(ep.DualStack.IPv6.Subnet)
			if ip == nil || subnet == nil {
				return nil, nil
			}
			return ip.To16(), subnet
		}
		ip = net.ParseIP(ep.DualStack.IPv4.IP)
		_, subnet, _ = net.ParseCIDR(ep.DualStack.IPv4.Subnet)
		if ip == nil || subnet == nil {
			return nil, nil
		}
		return ip.To4(), subnet
	}
	ip = net.ParseIP(ep.IP)
	_, subnet, _ = net.ParseCIDR(ep.Subnet)
	if ip == nil || subnet == nil {
		return nil, nil
	}
	isIPv6 := ip.To4() == nil
	if forIPv6 != isIPv6 {
		return nil, nil
	}
	if forIPv6 {
		return ip.To16(), subnet
	}
	return ip.To4(), subnet
}

func (a *agent) getEndpointAllIPs(ep api.Endpoint) (epIPs []net.IP) {
	epIPv4, _ := a.getEndpointIP(ep, false)
	if epIPv4 != nil {
		epIPs = append(epIPs, epIPv4)
	}
	epIPv6, _ := a.getEndpointIP(ep, true)
	if epIPv6 != nil {
		epIPs = append(epIPs, epIPv6)
	}
	return epIPs
}

// XXX With Go 1.18 and generics we can do better.
func (a *agent) labeledItemToEndpoint(item *labeledItem) api.Endpoint {
	switch item.category {
	case api.Client{}.ItemCategory():
		return item.LabeledItem.(api.Client).Endpoint
	case api.DNSServer{}.ItemCategory():
		return item.LabeledItem.(api.DNSServer).Endpoint
	case api.NTPServer{}.ItemCategory():
		return item.LabeledItem.(api.NTPServer).Endpoint
	case api.HTTPServer{}.ItemCategory():
		return item.LabeledItem.(api.HTTPServer).Endpoint
	case api.ExplicitProxy{}.ItemCategory():
		return item.LabeledItem.(api.ExplicitProxy).Endpoint
	case api.TransparentProxy{}.ItemCategory():
		return item.LabeledItem.(api.TransparentProxy).Endpoint
	case api.NetbootServer{}.ItemCategory():
		return item.LabeledItem.(api.NetbootServer).Endpoint
	default:
		log.Fatalf("Unexpected endpoint category: %s", item.category)
	}
	return api.Endpoint{} // unreachable
}

func (a *agent) genIfName(prefix, logicalLabel string) string {
	ifNameLen := len(prefix) + len(logicalLabel)
	if ifNameLen <= ifNameMaxLen {
		return prefix + logicalLabel
	}
	hashLen := ifNameMaxLen - len(prefix)
	if hashLen < 3 {
		log.Fatalf("interface name prefix too long: %s", prefix)
	}
	if hashLen > 6 {
		hashLen = 6
	}
	return prefix + hashString(logicalLabel, hashLen)
}

const (
	// 32 letters (5 bits to fit single one)
	letters5b = "abcdefghijklmnopqrstuvwxyzABCDEF"
)

// hashString returns a hash of an arbitrarily long string.
// The hash will have <len> characters (shouldn't be more than 7).
func hashString(str string, len int) string {
	h := fnv.New32a()
	h.Write([]byte(str))
	hn := h.Sum32()
	var hash string
	bitMask5b := uint32((1 << 5) - 1)
	for i := 0; i < len; i++ {
		hash = string(letters5b[int(hn&bitMask5b)]) + hash
		hn >>= 5
	}
	return hash
}

func strListContains(list []string, item string) bool {
	for i := range list {
		if item == list[i] {
			return true
		}
	}
	return false
}

func filterOutNilAddrs(addrs ...*net.IPNet) (filtered []*net.IPNet) {
	for _, addr := range addrs {
		if addr != nil {
			filtered = append(filtered, addr)
		}
	}
	return filtered
}
