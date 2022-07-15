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
	// ResolvConfTypename : typename for singleton item representing resolv.conf.
	ResolvConfTypename = "Resolv-Conf"
	// IPForwardingTypename : typename for singleton item representing enabled
	// or disabled IP forwarding.
	IPForwardingTypename = "IP-Forwarding"
	// DhcpcdTypename : typename for dhcpcd program (DHCP and DHCPv6 client).
	DhcpcdTypename = "DHCP-Client"
)
