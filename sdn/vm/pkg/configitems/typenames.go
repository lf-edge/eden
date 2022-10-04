package configitems

const (
	// Name used for singleton items.
	singletonName = "singleton"
	// PhysIfTypename : typename for physical network interfaces.
	PhysIfTypename = "Physical-Interface"
	// IfHandleTypename : typename for network interface handle.
	IfHandleTypename = "Interface-Handle"
	// NetNamespaceTypename : typename for network namespaces.
	NetNamespaceTypename = "Network-Namespace"
	// BondTypename : typename for bond interface.
	BondTypename = "Bond"
	// BridgeTypename : typename for bridges.
	BridgeTypename = "Bridge"
	// BridgeTypename : typename for veths.
	VethTypename = "Veth"
	// SysctlTypename : typename for item representing kernel
	// parameters set using sysctl for a given net namespace.
	SysctlTypename = "Sysctl"
	// DhcpClientTypename : typename for DHCP/DHCPv6 client.
	DhcpClientTypename = "DHCP-Client"
	// DhcpServerTypename : typename for DHCP/DHCPv6 server.
	DhcpServerTypename = "DHCP-Server"
	// DnsServerTypename : typename for DNS server.
	DnsServerTypename = "DNS-Server"
	// RouteTypename : typename for IP route.
	RouteTypename = "Route"
	// IPRuleTypename : typename for IP rule.
	IPRuleTypename = "IP-Rule"
	// IPtablesChainTypename : typename for a single iptables chain (IPv4).
	IPtablesChainTypename = "Iptables-Chain"
	// IP6tablesChainTypename : typename for a single ip6tables chain (IPv6).
	IP6tablesChainTypename = "Ip6tables-Chain"
	// HTTPProxyTypename : typename for HTTP proxy.
	HTTPProxyTypename = "HTTP-Proxy"
)
