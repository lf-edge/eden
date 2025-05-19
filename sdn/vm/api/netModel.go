package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
)

// NetworkModel is used to declaratively describe the intended state of the networking
// around EVE VM(s) for the testing purposes. The model is submitted in the JSON format
// to Eden-SDN Agent, running inside a separate VM, connected to EVE VM(s) via inter-VM
// network interfaces, and emulating the desired networking using the Linux network stack,
// network namespaces, netfilter and with the help of several open-source projects,
// such as dnsmasq, radvd, goproxy, etc.
type NetworkModel struct {
	// Ports : network interfaces connecting EVE VM(s) with Eden-SDN VM.
	// Each port is essentially an interconnected pair of network interfaces, with one side
	// inserted into the EVE VM and the other to Eden-SDN. These interface pairs are created
	// in both VMs in the order as listed here. This means that Ports[0] will appear
	// as the first interface in EVE (likely named "eth0" by the kernel) and likewise
	// as the first interface in Eden-SDN. Note that Eden-SDN will have one more extra
	// interface, added as last and used for management (for eden to talk to SDN mgmt agent).
	Ports []Port `json:"ports"`
	// Bonds are aggregating multiple ports for load-sharing and redundancy purposes.
	Bonds []Bond `json:"bonds"`
	// Bridges provide L2 connectivity.
	Bridges []Bridge `json:"bridges"`
	// Networks provide L3 connectivity.
	Networks []Network `json:"networks"`
	// Endpoints simulate "remote" clients and servers.
	Endpoints Endpoints `json:"endpoints"`
	// Firewall is applied between Networks, Endpoints and the outside of Eden-SDN
	// (controller, Internet).
	Firewall Firewall `json:"firewall"`
	// Host configuration that Eden-SDN is informed about.
	// Eden SDN needs to learn about the host configuration to properly route traffic
	// to the controller and beyond to the Internet.
	// If not defined (nil pointer), Eden will try to detect host config automatically.
	Host *HostConfig `json:"host,omitempty"`
}

// GetPortByMAC : lookup port by MAC address.
func (m NetworkModel) GetPortByMAC(mac string) *Port {
	for _, port := range m.Ports {
		if port.MAC == mac {
			return &port
		}
	}
	return nil
}

// LabeledItem is implemented by anything that has logical label associated with it.
// These methods helps with the config parsing and validation.
type LabeledItem interface {
	// ItemType : name of the item type (e.g. "port", "bond", "bridge", etc.)
	ItemType() string
	// ItemLogicalLabel : logical label of the item.
	ItemLogicalLabel() string
	// ReferencesFromItem : all references to logical labels from inside of the item.
	ReferencesFromItem() []LogicalLabelRef
}

// LabeledItemWithCategory : items of the same type can be further separated with categories.
// Still the pair (type, logicalLabel) remains as the unique item ID.
type LabeledItemWithCategory interface {
	LabeledItem
	// ItemCategory : optional item category (e.g. different kinds of endpoints).
	ItemCategory() string
}

// LogicalLabelRef : reference to an item's logical label.
type LogicalLabelRef struct {
	// ItemType: Type of the referenced item.
	ItemType string
	// ItemCategory : Category of the referenced item. Can be empty.
	ItemCategory string
	// ItemLogicalLabel : LogicalLabel of the referenced item.
	ItemLogicalLabel string
	// RefKey is used to enforce reference exclusivity.
	// There should not be more than one reference towards
	// the same item with the same RefKey.
	RefKey string
}

// Port is a network interface connecting EVE VM with Eden-SDN VM.
type Port struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string `json:"logicalLabel"`
	// MAC address assigned to the interface on the SDN side.
	// If not specified by the user, Eden will generate a random MAC address.
	MAC string `json:"mac"`
	// AdminUP : whether the interface should be UP on the SDN side.
	// Put down to test link-down scenarios on EVE.
	AdminUP bool `json:"adminUP"`
	// EVEConnect : plug the other side of the port into a given EVE instance.
	EVEConnect EVEConnect `json:"eveConnect"`
	// TC : traffic control.
	TC TrafficControl `json:"trafficControl"`
}

// TrafficControl allows to control traffic going through a port.
// It can be used to emulate slow and faulty networks.
type TrafficControl struct {
	// Delay refers to the duration, measured in milliseconds, by which each packet
	// will be delayed.
	Delay uint32 `json:"delay"`
	// DelayJitter : jitter in milliseconds added to the delay.
	DelayJitter uint32 `json:"delayJitter"`
	// LossProbability : probability of a packet loss (in percent).
	LossProbability uint8 `json:"lossProbability"`
	// CorruptProbability : probability of a packet corruption (in percent).
	CorruptProbability uint8 `json:"corruptProbability"`
	// DuplicateProbability : probability of a packet duplication (in percent).
	DuplicateProbability uint8 `json:"duplicateProbability"`
	// ReorderProbability represents the percentage probability of a packet's order
	// being modified within the queue.
	ReorderProbability uint8 `json:"reorderProbability"`
	// RateLimit represents the maximum speed, measured in kilobytes per second,
	// at which traffic can flow through the port.
	RateLimit uint32 `json:"rateLimit"`
	// QueueLimit : number of kilobytes that can be queued before being sent further.
	// Packets that would exceed the queue size are dropped.
	// Mandatory if RateLimit is set.
	QueueLimit uint32 `json:"queueLimit"`
	// BurstLimit represents the maximum amount of data, measured in kilobytes,
	// that can be sent or received in a short burst or interval, temporarily exceeding
	// the rate limit.
	// Mandatory if RateLimit is set.
	BurstLimit uint32 `json:"burstLimit"`
}

// ItemType
func (p Port) ItemType() string {
	return "port"
}

// ItemLogicalLabel
func (p Port) ItemLogicalLabel() string {
	return p.LogicalLabel
}

// ReferencesFromItem
func (p Port) ReferencesFromItem() []LogicalLabelRef {
	return nil
}

// EVEConnect : connects Port to a given EVE instance.
type EVEConnect struct {
	// EVEInstance : name of the EVE instance to which a given port is connected.
	// In the future, Eden may support running multiple EVE instances connected
	// to the same SDN and controller. It is likely that each such instance
	// will be assigned a unique logical label, which this field will reference.
	// However, currently Eden is only able to manage a single EVE instance.
	// For the time being it is therefore expected that this field is empty
	// and EVEConnect refers to the one and only EVE instance.
	EVEInstance string `json:"eveInstance"`
	// MAC address assigned to the interface on the EVE side.
	// If not specified by the user, Eden will generate a random MAC address.
	MAC string `json:"mac"`
}

// Bridge provides L2 connectivity.
type Bridge struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string `json:"logicalLabel"`
	// Logical labels of ports.
	Ports []string `json:"ports"`
	// Logical labels of bonds.
	Bonds []string `json:"bonds"`
	// WithSTP: enable to run the Spanning Tree Protocol (STP).
	WithSTP bool `json:"withSTP"`
}

// ItemType
func (b Bridge) ItemType() string {
	return "bridge"
}

// ItemLogicalLabel
func (b Bridge) ItemLogicalLabel() string {
	return b.LogicalLabel
}

// PortMasterRef : reference to a physical port by a master interface (bond or bridge).
// Same between (and within) bonds and bridges to enforce exclusive access to the port.
const PortMasterRef = "port-master"

// ReferencesFromItem
func (b Bridge) ReferencesFromItem() []LogicalLabelRef {
	var refs []LogicalLabelRef
	for _, port := range b.Ports {
		refs = append(refs, LogicalLabelRef{
			ItemType:         Port{}.ItemType(),
			ItemLogicalLabel: port,
			RefKey:           PortMasterRef,
		})
	}
	for _, bond := range b.Bonds {
		refs = append(refs, LogicalLabelRef{
			ItemType:         Bond{}.ItemType(),
			ItemLogicalLabel: bond,
			RefKey:           PortMasterRef,
		})
	}
	return refs
}

// Network provides L3 connectivity.
type Network struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string `json:"logicalLabel"`
	// Logical label of a Bridge to which the network is attached.
	Bridge string `json:"bridge"`
	// Leave zero value to express intent of not using VLAN for this network.
	VlanID uint16 `json:"vlanID"`
	// MTU : Maximum transmission unit size set for this network.
	// If not defined (zero value), the default MTU for Ethernet, which is 1500 bytes,
	// will be set.
	MTU uint16 `json:"mtu"`
	// Single-stack network IP (v4 or v6) configuration.
	// Define either this or DualStack.
	NetworkIPConfig
	// Dual-stack network IP configuration.
	// Define either this or the (single-stack) embedded NetworkIPConfig.
	DualStack DualStackNetwork `json:"dualStack"`
	// TransparentProxy : Logical label of a TransparentProxy endpoint, performing
	// proxying of both HTTP and HTTPS traffic transparently.
	// Traffic will flow as follows:
	//   EVE -> Network -> Router -> Firewall -> TransparentProxy -> Router -> Firewall ...
	//    ... -> Endpoint
	//        OR
	//        -> another Network
	//        OR
	//        -> Outside-of-SDN-VM
	TransparentProxy string `json:"transparentProxy,omitempty"`
	// Router configuration. Every network has a separate routing context.
	// Undefined (nil) means that everything should be routed and accessible.
	// That includes all networks, endpoints and the outside of Eden SDN.
	Router *Router `json:"router,omitempty"`
}

// IsDualStack returns true if Network is configured to operate in dual-stack IP mode.
func (n Network) IsDualStack() bool {
	return n.DualStack.IPv4.Subnet != "" || n.DualStack.IPv6.Subnet != ""
}

// HasIPv4Subnet returns true if the network has IPv4 subnet.
func (n Network) HasIPv4Subnet() bool {
	if n.IsDualStack() {
		return true
	}
	_, subnet, _ := net.ParseCIDR(n.Subnet)
	return subnet != nil && subnet.IP.To4() != nil
}

// HasIPv6Subnet returns true if the network has IPv6 subnet.
func (n Network) HasIPv6Subnet() bool {
	if n.IsDualStack() {
		return true
	}
	_, subnet, _ := net.ParseCIDR(n.Subnet)
	return subnet != nil && subnet.IP.To4() == nil
}

// NetworkIPConfig : IP configuration for Network.
type NetworkIPConfig struct {
	// Subnet : network address + netmask (IPv4 or IPv6).
	Subnet string `json:"subnet"`
	// GwIP should be inside the Subnet.
	GwIP string `json:"gwIP"`
	// DHCP configuration.
	DHCP DHCP `json:"dhcp"`
}

// DualStackNetwork : dual-stack IP configuration for Network.
type DualStackNetwork struct {
	// IPv4 config for Network.
	IPv4 NetworkIPConfig `json:"ipv4"`
	// IPv6 config for Network.
	IPv6 NetworkIPConfig `json:"ipv6"`
}

// ItemType
func (n Network) ItemType() string {
	return "network"
}

// ItemLogicalLabel
func (n Network) ItemLogicalLabel() string {
	return n.LogicalLabel
}

// NetworkBridgeRefPrefix : prefix used for references to bridges from networks.
const NetworkBridgeRefPrefix = "bridge-network"

// ReferencesFromItem
func (n Network) ReferencesFromItem() []LogicalLabelRef {
	var refs []LogicalLabelRef
	// Bridge reference.
	var bridgeRefKey string
	if n.VlanID == 0 {
		// At most one non-VLANed network for this bridge.
		bridgeRefKey = NetworkBridgeRefPrefix
	} else {
		// Ensures unique VLAN IDs.
		bridgeRefKey = fmt.Sprintf("%s-vlan%d", NetworkBridgeRefPrefix, n.VlanID)
	}
	refs = append(refs, LogicalLabelRef{
		ItemType:         Bridge{}.ItemType(),
		ItemLogicalLabel: n.Bridge,
		RefKey:           bridgeRefKey,
	})
	// References from inside the DHCP config.
	if n.DHCP.Enable {
		for _, dns := range n.DHCP.PrivateDNS {
			refs = append(refs, LogicalLabelRef{
				ItemType:         Endpoint{}.ItemType(),
				ItemCategory:     DNSServer{}.ItemCategory(),
				ItemLogicalLabel: dns,
				// Avoids duplicate DNS servers for the same network.
				RefKey: "dns-for-network-" + n.LogicalLabel,
			})
		}
		if n.DHCP.PrivateNTP != "" {
			refs = append(refs, LogicalLabelRef{
				ItemType:         Endpoint{}.ItemType(),
				ItemCategory:     NTPServer{}.ItemCategory(),
				ItemLogicalLabel: n.DHCP.PrivateNTP,
				RefKey:           "ntp-for-network-" + n.LogicalLabel,
			})
		}
		if n.DHCP.NetbootServer != "" {
			refs = append(refs, LogicalLabelRef{
				ItemType:         Endpoint{}.ItemType(),
				ItemCategory:     NetbootServer{}.ItemCategory(),
				ItemLogicalLabel: n.DHCP.NetbootServer,
				RefKey:           "netboot-for-network-" + n.LogicalLabel,
			})
		}
	}
	// Routable networks.
	if n.Router != nil {
		for _, reachEp := range n.Router.ReachableEndpoints {
			refs = append(refs, LogicalLabelRef{
				ItemType:         Endpoint{}.ItemType(),
				ItemLogicalLabel: reachEp,
				RefKey:           "reachable-by-network-" + n.LogicalLabel,
			})
		}
		for _, reachNet := range n.Router.ReachableNetworks {
			refs = append(refs, LogicalLabelRef{
				ItemType:         Network{}.ItemType(),
				ItemLogicalLabel: reachNet,
				RefKey:           "reachable-by-network-" + n.LogicalLabel,
			})
		}
	}
	// Reference to a TransparentProxy.
	if n.TransparentProxy != "" {
		refs = append(refs, LogicalLabelRef{
			ItemType:         Endpoint{}.ItemType(),
			ItemCategory:     TransparentProxy{}.ItemCategory(),
			ItemLogicalLabel: n.TransparentProxy,
			RefKey:           "network-tproxy-" + n.LogicalLabel,
		})
	}
	return refs
}

// DHCP configuration.
// For IPv6, if only DNS-related options are set, SLAAC remains the sole method
// for address assignment. Specifying any additional options (e.g., IP range,
// NTP servers) enables DHCPv6.
type DHCP struct {
	// Enables DHCP. Set to false to use static IP addressing in EVE.
	// For IPv6, SLAAC remains available regardless of this setting.
	Enable bool `json:"enable"`
	// IPRange : a range of IP addresses to allocate from.
	IPRange IPRange `json:"ipRange"`
	// StaticEntries : list of MAC->IP entries statically configured for the DHCP server.
	StaticEntries []MACToIP `json:"staticEntries"`
	// WithoutDefaultRoute : do not advertise default route to DHCP clients.
	WithoutDefaultRoute bool `json:"withoutDefaultRoute"`
	// DomainName : name of the domain assigned to the network.
	// It is propagated to clients using the DHCP option 15 (24 in DHCPv6).
	DomainName string `json:"domainName"`
	// DNSClientConfig : DNS configuration passed to clients via DHCP.
	DNSClientConfig
	// Public NTP server to announce via DHCP option 42 (56 in DHCPv6).
	// Do not configure both PublicNTP and PrivateNTP.
	PublicNTP string `json:"publicNTP"`
	// Logical label of an NTP endpoint running inside Eden SDN, announced to client
	// via DHCP option 42 (56 in DHCPv6).
	// Do not configure both PublicNTP and PrivateNTP.
	PrivateNTP string `json:"privateNTP"`
	// WPAD : URL with a location of a PAC file, announced using the Web Proxy Auto-Discovery
	// Protocol (WPAD) and DHCP.
	// The PAC file should contain a javascript that the client will use to determine
	// which proxy to use for a given request.
	// URL example: http://wpad.example.com/wpad.dat
	// The client will learn the PAC file location using the DHCP option 252.
	// Not supported for IPv6.
	// An alternative (and the only available for IPv6) approach is to use DNS
	// (with a DNSServer endpoint).
	WPAD string `json:"wpad"`
	// NetbootServer : Logical label of a NetbootServer endpoint which the client should use
	// to boot EVE OS from. The IP address or FQDN and the provisioning file (iPXE script)
	// location will be announced to the client using DHCP options 66 and 67 (59 in DHCPv6).
	// Eden-SDN will announce either IP address or FQDN depending on whether any of the assigned
	// private DNS servers is able to resolve the NetbootServer domain name.
	NetbootServer string `json:"netbootServer"`
}

// MACToIP maps MAC address to IP address.
type MACToIP struct {
	// MAC address.
	MAC string `json:"mac"`
	// IP address.
	IP string `json:"ip"`
}

// DNSClientConfig : DNS configuration for a client.
type DNSClientConfig struct {
	// PublicDNS specifies a list of IP addresses of public DNS servers.
	// These will be announced to clients using:
	// - DHCPv4: via option 6 (Domain Name Server)
	// - DHCPv6: via option 23 (Recursive DNS Server - RDNSS)
	// - SLAAC (IPv6): via Router Advertisement (RFC 6106)
	// Example: ["1.1.1.1", "8.8.8.8"]
	PublicDNS []string `json:"publicDNS"`
	// PrivateDNS : list of DNS servers running as endpoints inside Eden SDN.
	// These will be announced to clients using the same mechanisms as public DNS servers.
	// The list should contain logical labels of those endpoints, not IP addresses!
	PrivateDNS []string `json:"privateDNS"`
}

// Router routing traffic for a network based on the reachability requirements.
type Router struct {
	// OutsideReachability : If enabled then it is possible to use the network to access
	// endpoints outside of Eden SDN (unless blocked by firewall).
	// This includes the controller (Adam, zedcloud), eserver (image cache) and the Internet.
	OutsideReachability bool `json:"outsideReachability"`
	// ReachableEndpoints : Logical labels of reachable endpoints.
	ReachableEndpoints []string `json:"reachableEndpoints"`
	// ReachableNetworks : Logical labels of reachable networks.
	ReachableNetworks []string `json:"reachableNetworks"`
	// RoutesTowardsEVE : a set of routes for traffic going towards EVE.
	// This is typically used if one of the apps running on EVE is used as a gateway
	// for other apps. Static routes are then needed on the Eden-SDN side to route
	// returning traffic back via that gw app.
	RoutesTowardsEVE []IPRoute `json:"routesTowardsEVE"`
}

// IPRoute : a single IP route entry.
type IPRoute struct {
	// DstNetwork : destination network address in the CIDR format:
	// <IP-address>/<prefix-length>
	DstNetwork string `json:"dstNetwork"`
	// Gateway IP address. It should be from within the EVE port network subnet.
	Gateway string `json:"gateway"`
}

// Firewall : network firewall.
// Note that traffic not matched by any rule is allowed!
type Firewall struct {
	// Rules : firewall rules applied in the order as configured.
	// Applied to traffic going <from> -> <to>:
	//   - network (EVE ports) -> another network
	//   - network (EVE ports) -> endpoint
	//   - network (EVE ports) -> outside of SDN VM (controller, Internet)
	//   - endpoint -> network (EVE ports)
	//   - endpoint -> another endpoint
	//   - endpoint -> outside of SDN VM (controller, Internet)
	// Note that once a connection is allowed, established and related traffic
	// (going in the opposite direction) is automatically allowed as well.
	Rules []FwRule `json:"rules"`
}

// FwRule : a firewall rule.
type FwRule struct {
	// SrcSubnet : subnet to match the source IP address with.
	// Can be empty to disable filtering based on source IP address.
	SrcSubnet string `json:"srcSubnet"`
	// DstSubnet : subnet to match the destination IP address with.
	// Can be empty to disable filtering based on destination IP address.
	DstSubnet string `json:"dstSubnet"`
	// Protocol : filter by protocol.
	Protocol FwProto `json:"protocol"`
	// Ports : list of destination port to which the rule applies.
	// For a non empty list, Protocol must be either TCP or UDP.
	// Empty = any.
	Ports []uint16 `json:"ports"`
	// Action to take.
	Action FwAction `json:"action"`
}

// HostConfig : host configuration that Eden-SDN needs to be informed about.
type HostConfig struct {
	// HostIPs : list of IP addresses used by the host system (on top of which
	// Eden runs).
	// Eden SDN requires at least one routable host IP address.
	HostIPs []string `json:"hostIPs"`
	// NetworkType specifies which IP versions (IPv4, IPv6, or both) are used by the host.
	//
	// Note: IPv4-to-IPv6 translation (e.g., DNS64/NAT64) is not currently implemented.
	// As a result, IP connectivity between EVE and the controller requires matching
	// IP versions.
	// To ensure connectivity:
	// - The host must use the same IP version as EVE, or
	// - Run in dual-stack mode (supporting both IPv4 and IPv6)
	//
	// If Adam is used as the controller, this can be configured by setting the correct
	// IP version for the Docker bridge network. However, for access to any external
	// endpoints (e.g. upstream DNS servers), it is required that the host itself
	// has working connectivity of the required IP version.
	NetworkType NetworkType `json:"networkType"`
	// ControllerPort : port on which controller listens for device requests.
	ControllerPort uint16 `json:"controllerPort"`
}

// IPRange : a range of IP addresses.
type IPRange struct {
	// FromIP : start of the range (includes the address itself).
	FromIP string `json:"fromIP"`
	// ToIP : end of the range (includes the address itself).
	ToIP string `json:"toIP"`
}

// NetworkType : type of the network wrt. IP version used.
type NetworkType uint8

const (
	// Ipv4Only : host uses IPv4 only.
	Ipv4Only NetworkType = iota
	// Ipv6Only : host uses IPv6 only.
	Ipv6Only
	// DualStack : host runs with dual stack.
	DualStack = 8
)

// NetworkTypeToString : convert NetworkType to string representation used in JSON.
var NetworkTypeToString = map[NetworkType]string{
	Ipv4Only:  "ipv4-only",
	Ipv6Only:  "ipv6-only",
	DualStack: "dual-stack",
}

// NetworkTypeToID : get NetworkType from a string representation.
var NetworkTypeToID = map[string]NetworkType{
	"":           Ipv4Only, // default value
	"ipv4-only":  Ipv4Only,
	"ipv6-only":  Ipv6Only,
	"dual-stack": DualStack,
}

// MarshalJSON marshals the enum as a quoted json string.
func (s NetworkType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(NetworkTypeToString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON un-marshals a quoted json string to the enum value.
func (s *NetworkType) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = NetworkTypeToID[j]
	return nil
}

// FwAction : firewall action.
type FwAction uint8

const (
	// FwAllow : allow traffic.
	FwAllow FwAction = iota
	// FwReject : reject traffic.
	// The sender will be informed using an ICMP packet that the destination
	// is blocked/unavailable.
	FwReject
	// FwDrop : drop traffic.
	// Traffic is silently dropped.
	FwDrop
)

// FwActionToString : convert FwAction to string representation used in JSON.
var FwActionToString = map[FwAction]string{
	FwAllow:  "allow",
	FwReject: "reject",
	FwDrop:   "drop",
}

// FwActionToID : get FwAction from a string representation.
var FwActionToID = map[string]FwAction{
	"":       FwAllow, // default value
	"allow":  FwAllow,
	"reject": FwReject,
	"drop":   FwDrop,
}

// MarshalJSON marshals the enum as a quoted json string.
func (s FwAction) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(FwActionToString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON un-marshals a quoted json string to the enum value.
func (s *FwAction) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = FwActionToID[j]
	return nil
}

// FwProto : protocol to apply a firewall rule on.
type FwProto uint8

const (
	AnyProto FwProto = iota
	ICMP
	TCP
	UDP
)

// FwProtoToString : convert FwProto to string representation used in JSON.
var FwProtoToString = map[FwProto]string{
	AnyProto: "any",
	ICMP:     "icmp",
	TCP:      "tcp",
	UDP:      "udp",
}

// FwProtoToID : get FwProto from a string representation.
var FwProtoToID = map[string]FwProto{
	"":     AnyProto,
	"any":  AnyProto,
	"icmp": ICMP,
	"tcp":  TCP,
	"udp":  UDP,
}

// MarshalJSON marshals the enum as a quoted json string.
func (s FwProto) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(FwProtoToString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON un-marshals a quoted json string to the enum value.
func (s *FwProto) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = FwProtoToID[j]
	return nil
}
