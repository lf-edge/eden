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

	// TODO (ntp servers, netboot servers)
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
			MTU:          maxMTU,
		}, nil)
	}
	return intendedCfg
}

func (a *agent) getIntendedNetwork(network api.Network) dg.Graph {
	index, hasIndex := a.networkIndex[network.LogicalLabel]
	if !hasIndex {
		log.Fatalf("missing index for network %s", network.LogicalLabel)
	}
	graphArgs := dg.InitArgs{Name: networkSGPrefix + network.LogicalLabel}
	intendedCfg := dg.New(graphArgs)

	// Network namespace connected with the bridge using veth.
	brVethName, brInIfName, brOutIfName := a.networkBrVethName(network.LogicalLabel)
	_, subnet, _ := net.ParseCIDR(network.Subnet) // already validated
	gwIP := &net.IPNet{IP: net.ParseIP(network.GwIP), Mask: subnet.Mask}
	nsName := a.networkNsName(network.LogicalLabel)
	netNs := configitems.NetNamespace{
		NsName: nsName,
	}
	intendedCfg.PutItem(netNs, nil)
	intendedCfg.PutItem(configitems.Veth{
		VethName: brVethName,
		Peer1: configitems.VethPeer{
			IfName:       brInIfName,
			NetNamespace: nsName,
			IPAddresses:  []*net.IPNet{gwIP},
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
	isIPv6 := len(subnet.IP) == net.IPv6len
	inIP, outIP := a.genVethIPsForNetwork(network.LogicalLabel, isIPv6)
	intendedCfg.PutItem(configitems.Veth{
		VethName: rtVethName,
		Peer1: configitems.VethPeer{
			IfName:       rtInIfName,
			NetNamespace: nsName,
			IPAddresses:  []*net.IPNet{inIP},
			MTU:          network.MTU,
		},
		Peer2: configitems.VethPeer{
			IfName:       rtOutIfName,
			NetNamespace: configitems.MainNsName,
			IPAddresses:  []*net.IPNet{outIP},
			MTU:          network.MTU,
		},
	}, nil)

	// DHCP server.
	dhcp := network.DHCP
	if dhcp.Enable {
		ipRange := a.subnetToHostIPRange(subnet)
		if dhcp.IPRange.FromIP != "" {
			ipRange.FromIP = net.ParseIP(dhcp.IPRange.FromIP)
			ipRange.ToIP = net.ParseIP(dhcp.IPRange.ToIP)
		}
		ntpServer := dhcp.PublicNTP
		if dhcp.PrivateNTP != "" {
			ep := a.getEndpoint(dhcp.PrivateNTP)
			ntpServer = ep.IP // XXX Or FQDN?
		}
		var dnsServers []net.IP
		for _, dnsServer := range dhcp.PublicDNS {
			dnsServers = append(dnsServers, net.ParseIP(dnsServer))
		}
		for _, dnsServer := range dhcp.PrivateDNS {
			ep := a.getEndpoint(dnsServer)
			dnsServers = append(dnsServers, net.ParseIP(ep.IP))
		}
		gatewayIP := gwIP.IP
		if dhcp.WithoutDefaultRoute {
			gatewayIP = nil
		}
		var staticEntries []configitems.MACToIP
		for _, entry := range dhcp.StaticEntries {
			mac, _ := net.ParseMAC(entry.MAC)
			staticEntries = append(staticEntries, configitems.MACToIP{
				MAC: mac,
				IP:  net.ParseIP(entry.IP),
			})
		}
		intendedCfg.PutItem(configitems.DhcpServer{
			ServerName:     network.LogicalLabel,
			NetNamespace:   nsName,
			VethName:       brVethName,
			VethPeerIfName: brInIfName,
			Subnet:         subnet,
			IPRange:        ipRange,
			StaticEntries:  staticEntries,
			GatewayIP:      gatewayIP,
			DomainName:     dhcp.DomainName,
			DNSServers:     dnsServers,
			NTPServer:      ntpServer,
			WPAD:           network.DHCP.WPAD,
		}, nil)
	}

	// Routing.
	rt := networkRTBaseIndex + index
	intendedCfg.PutItem(configitems.IPRule{
		SrcNet:   subnet,
		Table:    rt,
		Priority: networkIPRulePriority,
	}, nil)
	intendedCfg.PutItem(configitems.IPRule{
		DstNet:   subnet,
		Table:    rt,
		Priority: networkIPRulePriority,
	}, nil)
	defaultDst := allIPv4
	if isIPv6 {
		defaultDst = allIPv6
	}
	// - default route from inside of the network namespace
	intendedCfg.PutItem(configitems.Route{
		NetNamespace: nsName,
		Table:        syscall.RT_TABLE_MAIN,
		DstNet:       defaultDst,
		OutputIf: configitems.RouteOutIf{
			VethName:       rtVethName,
			VethPeerIfName: rtInIfName,
		},
		GwIP: outIP.IP,
	}, nil)
	// - route for every endpoint
	epTypename := api.Endpoint{}.ItemType()
	for itemID, item := range a.netModel.items {
		if itemID.typename != epTypename {
			continue
		}
		ep := a.labeledItemToEndpoint(item)
		_, epSubnet, _ := net.ParseCIDR(ep.Subnet)
		reachable := network.Router == nil ||
			strListContains(network.Router.ReachableEndpoints, ep.LogicalLabel)
		if reachable {
			epVethName, _, epOutIfName := a.endpointVethName(ep.LogicalLabel)
			intendedCfg.PutItem(configitems.Route{
				NetNamespace: configitems.MainNsName,
				Table:        rt,
				DstNet:       epSubnet,
				OutputIf: configitems.RouteOutIf{
					VethName:       epVethName,
					VethPeerIfName: epOutIfName,
				},
				GwIP: net.ParseIP(ep.IP),
			}, nil)
		} else {
			intendedCfg.PutItem(configitems.Route{
				NetNamespace: configitems.MainNsName,
				Table:        rt,
				DstNet:       epSubnet,
			}, nil)
		}
	}
	// - route for every other network (including itself)
	for _, network2 := range a.netModel.Networks {
		_, net2Subnet, _ := net.ParseCIDR(network2.Subnet)
		reachable := network.Router == nil ||
			network2.LogicalLabel == network.LogicalLabel ||
			strListContains(network.Router.ReachableNetworks, network2.LogicalLabel)
		if reachable {
			net2VethName, _, net2OutIfName := a.networkRtVethName(network2.LogicalLabel)
			net2InIP, _ := a.genVethIPsForNetwork(network2.LogicalLabel, isIPv6)
			intendedCfg.PutItem(configitems.Route{
				NetNamespace: configitems.MainNsName,
				Table:        rt,
				DstNet:       net2Subnet,
				OutputIf: configitems.RouteOutIf{
					VethName:       net2VethName,
					VethPeerIfName: net2OutIfName,
				},
				GwIP: net2InIP.IP,
			}, nil)
		} else {
			intendedCfg.PutItem(configitems.Route{
				NetNamespace: configitems.MainNsName,
				Table:        rt,
				DstNet:       net2Subnet,
			}, nil)
		}
	}
	// - route for the outside world if enabled
	outsideRechability := network.Router == nil || network.Router.OutsideReachability
	hostPort, hostPortfound := a.macLookup.GetInterfaceByMAC(hostPortMACPrefix, true)
	hostGwIP := a.getHostGwIP(isIPv6)
	if outsideRechability && hostPortfound && hostGwIP != nil {
		intendedCfg.PutItem(configitems.Route{
			NetNamespace: configitems.MainNsName,
			Table:        rt,
			DstNet:       defaultDst,
			OutputIf: configitems.RouteOutIf{
				PhysIf: configitems.PhysIf{
					MAC:          hostPort.MAC,
					LogicalLabel: hostPortLogicalLabel,
				},
			},
			GwIP: hostGwIP,
		}, nil)
	}
	// - routes towards EVE
	var routesTowardsEVE []api.IPRoute
	if network.Router != nil {
		routesTowardsEVE = network.Router.RoutesTowardsEVE
	}
	for _, route := range routesTowardsEVE {
		_, dstNetwork, _ := net.ParseCIDR(route.DstNetwork)
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
		intendedCfg.PutItem(configitems.Route{
			NetNamespace: configitems.MainNsName,
			Table:        rt,
			DstNet:       dstNetwork,
			OutputIf: configitems.RouteOutIf{
				VethName:       rtVethName,
				VethPeerIfName: rtOutIfName,
			},
			GwIP: inIP.IP,
		}, nil)
	}
	// - everything else is unreachable
	intendedCfg.PutItem(configitems.Route{
		NetNamespace: configitems.MainNsName,
		Table:        rt,
		DstNet:       defaultDst,
		Metric:       ^uint32(0), // Lowest prio.
	}, nil)

	// Transparent proxy.
	if network.TransparentProxy != "" {
		ep := a.getEndpoint(network.TransparentProxy)
		httpsPorts := []api.ProxyPort{{Port: 443}}
		controllerPort := a.netModel.Host.ControllerPort
		if controllerPort != 443 {
			httpsPorts = append(httpsPorts, api.ProxyPort{Port: controllerPort})
		}
		// iptables to transparently redirect traffic into the proxy
		dnatRules := []configitems.IptablesRule{
			{
				Args: []string{"-p", "tcp", "--dport", "80", "-j", "DNAT",
					"--to-destination", ep.IP},
				Description: "Send HTTP traffic into the proxy",
			},
		}
		for _, httpsPort := range httpsPorts {
			dnatRules = append(dnatRules, configitems.IptablesRule{
				Args: []string{"-p", "tcp", "--dport", strconv.Itoa(int(httpsPort.Port)),
					"-j", "DNAT", "--to-destination", ep.IP},
				Description: fmt.Sprintf("Send HTTPS traffic (port %d) into the proxy",
					httpsPort.Port),
			})
		}
		intendedCfg.PutItem(configitems.IptablesChain{
			NetNamespace: nsName,
			ChainName:    "PREROUTING",
			Table:        "nat",
			ForIPv6:      false,
			Rules:        dnatRules,
		}, nil)
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
	return intendedCfg
}

func (a *agent) getIntendedFirewall() dg.Graph {
	graphArgs := dg.InitArgs{Name: firewallSG}
	intendedCfg := dg.New(graphArgs)
	iptablesRules := make([]configitems.IptablesRule, 0, 2+len(a.netModel.Firewall.Rules))
	// Allow any subsequent traffic that results from an already allowed connection.
	iptablesRules = append(iptablesRules, configitems.IptablesRule{
		Args: []string{"-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
	})
	// Add explicitly configured firewall rules.
	for _, rule := range a.netModel.Firewall.Rules {
		iptablesRules = append(iptablesRules, a.getIntendedFwRule(rule))
	}
	// Implicitly allow everything not matched by the rules above.
	allowTheRest := api.FwRule{Action: api.FwAllow}
	iptablesRules = append(iptablesRules, a.getIntendedFwRule(allowTheRest))
	intendedCfg.PutItem(configitems.IptablesChain{
		NetNamespace: configitems.MainNsName,
		ChainName:    fwIptablesChain,
		Table:        "filter",
		ForIPv6:      false,
		Rules:        iptablesRules,
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
	return intendedCfg
}

func (a *agent) getIntendedFwRule(rule api.FwRule) configitems.IptablesRule {
	var ruleArgs []string
	if rule.SrcSubnet != "" {
		ruleArgs = append(ruleArgs, "-s", rule.SrcSubnet)
	}
	if rule.DstSubnet != "" {
		ruleArgs = append(ruleArgs, "-d", rule.DstSubnet)
	}
	switch rule.Protocol {
	case api.AnyProto:
		ruleArgs = append(ruleArgs, "-p", "all")
	case api.ICMP:
		ruleArgs = append(ruleArgs, "-p", "icmp")
	case api.TCP:
		ruleArgs = append(ruleArgs, "-p", "tcp")
	case api.UDP:
		ruleArgs = append(ruleArgs, "-p", "udp")
	}
	if len(rule.Ports) > 0 {
		var ports []string
		for _, port := range rule.Ports {
			ports = append(ports, strconv.Itoa(int(port)))
		}
		ruleArgs = append(ruleArgs, "--match", "multiport",
			"--dport", strings.Join(ports, ","))
	}
	switch rule.Action {
	case api.FwAllow:
		ruleArgs = append(ruleArgs, "-j", "ACCEPT")
	case api.FwReject:
		ruleArgs = append(ruleArgs, "-j", "REJECT")
	case api.FwDrop:
		ruleArgs = append(ruleArgs, "-j", "DROP")
	}
	return configitems.IptablesRule{
		Args: ruleArgs,
	}
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
		var ip net.IP
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
			ip = a.netModel.hostIP
		case strings.HasPrefix(staticEntry.IP, api.EndpointIPRefPrefix):
			epLL := strings.TrimPrefix(staticEntry.IP, api.EndpointIPRefPrefix)
			ep := a.getEndpoint(epLL)
			ip = net.ParseIP(ep.IP)
		default:
			ip = net.ParseIP(staticEntry.IP)
		}
		staticEntries = append(staticEntries, configitems.DnsEntry{
			FQDN: fqdn,
			IP:   ip,
		})
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
	epIP := net.ParseIP(proxy.IP)
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
		ListenIP:     epIP,
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
	epIP := net.ParseIP(proxy.IP)
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
		ListenIP:     epIP,
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
	epIP := net.ParseIP(httpSrv.IP)
	intendedCfg.PutItem(configitems.HttpServer{
		ServerName:   httpSrv.LogicalLabel,
		NetNamespace: nsName,
		VethName:     vethName,
		ListenIP:     epIP,
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
	_, subnet, _ := net.ParseCIDR(ep.Subnet) // already validated
	epIP := &net.IPNet{IP: net.ParseIP(ep.IP), Mask: subnet.Mask}
	gwIP := a.genEndpointGwIP(subnet, epIP.IP)
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
			dnsServers = append(dnsServers, net.ParseIP(ep.IP))
		}
		netNs.ResolvConf = configitems.ResolvConf{
			Create:     true,
			DNSServers: dnsServers,
		}
	}
	graph.PutItem(netNs, nil)
	graph.PutItem(configitems.Veth{
		VethName: vethName,
		Peer1: configitems.VethPeer{
			IfName:       inIfName,
			NetNamespace: nsName,
			IPAddresses:  []*net.IPNet{epIP},
			MTU:          ep.MTU,
		},
		Peer2: configitems.VethPeer{
			IfName:       outIfName,
			NetNamespace: configitems.MainNsName,
			IPAddresses:  []*net.IPNet{gwIP},
			MTU:          ep.MTU,
		},
	}, nil)
	defaultDst := allIPv4
	isIPv6 := len(subnet.IP) == net.IPv6len
	if isIPv6 {
		defaultDst = allIPv6
	}
	graph.PutItem(configitems.Route{
		NetNamespace: nsName,
		DstNet:       defaultDst,
		OutputIf: configitems.RouteOutIf{
			VethName:       vethName,
			VethPeerIfName: inIfName,
		},
		GwIP: gwIP.IP,
	}, nil)
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
