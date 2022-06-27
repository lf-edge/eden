package api

import (
	"bytes"
	"encoding/json"
)

// NetworkModel is used to declaratively describe the intended state of the networking
// around EVE VM for the testing purposes. The model is submitted in the JSON format to Eden-SDN Agent,
// running inside a separate VM, connected to EVE VM via inter-VM network interface, and emulating
// the desired networking using the Linux network stack, network namespaces, netfilter and with
// the help of several open-source projects, such as dnsmasq, radvd, mitmproxy, etc.
type NetworkModel struct {
	// Ports : network interfaces connecting EVE VM with Eden-SDN VM.
	Ports []Port
	// Bonds are aggregating multiple ports for load-sharing and redundancy purposes.
	Bonds []Bond
	// Bridges provide L2 connectivity.
	Bridges []Bridge
	// Networks provide L3 connectivity.
	Networks []Network
	// Endpoints simulate "remote" clients and servers.
	Endpoints Endpoints
	// Firewall is applied between Networks, Endpoints and the outside of Eden-SDN
	// (controller, Internet).
	Firewall Firewall
	// Host configuration that Eden-SDN is informed about.
	// Eden SDN needs to learn about the host configuration to properly route traffic
	// to the controller and beyond to the Internet.
	// If not defined (nil pointer), Eden will try to detect host config automatically.
	Host *HostConfig
}

// Port is a network interface connecting EVE VM with Eden-SDN VM.
type Port struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string
	// IfName : interface name in the kernel. The same name will be used by both EVE VM
	// and Eden-SDN VM.
	IfName string
	// MTU : Maximum transmission unit.
	MTU uint16
	// AdminUP : whether the interface should be UP on the SDN side.
	// Put down to test link-down scenarios on EVE.
	AdminUP bool
}

// Bridge provides L2 connectivity.
type Bridge struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string
	// IfName : bridge interface name in the kernel.
	IfName string
	// Logical labels of ports.
	Ports []string
	// Logical labels of bonds.
	Bonds []string
}

// Network provides L3 connectivity.
type Network struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string
	// Logical label of a Bridge to which the network is attached.
	Bridge string
	// Leave zero value to express intent of not using VLAN for this network.
	VlanID uint16
	// Subnet : network address + netmask (IPv4 or IPv6).
	Subnet string
	// GwIP should be inside the Subnet.
	GwIP string
	// DHCP configuration.
	DHCP DHCP
	// MitMProxy is a proxy that both HTTP and HTTPS traffic is forwarded through
	// transparently, before it reaches router, endpoints, firewall, etc.
	MitMProxy MitMProxy
	// Router configuration. Every network has a separate routing context.
	// Undefined (nil) means that everything should be routed and accessible.
	// That includes all networks, endpoints and the outside of Eden SDN.
	Router *Router
}

// DHCP configuration.
// Note that for IPv6 we prefer to use Router Advertisement over DHCPv6 to publish
// all this information to hosts on the network.
// TODO: need to check if we can announce NTP servers using NDP.
type DHCP struct {
	// Enable DHCP. Set to false to use EVE with static IP addressing.
	Enable bool
	// IPRange : a range of IP addresses to allocate from.
	// Not applicable for IPv6.
	IPRange IPRange
	// DomainName : name of the domain assigned to the network.
	// It is propagated to clients using the DHCP option 15.
	DomainName string
	// DNSClientConfig : DNS configuration passed to clients via DHCP.
	DNSClientConfig
	// Public NTP server to announce via DHCP option 42.
	// Do not configure both PublicNTP and PrivateNTP.
	PublicNTP string
	// Logical label of an NTP endpoint running inside Eden SDN, announced to client
	// via DHCP option 42.
	// Do not configure both PublicNTP and PrivateNTP.
	PrivateNTP string
	// WPAD : URL with a location of a PAC file, announced using the Web Proxy Auto-Discovery
	// Protocol (WPAD) and DHCP.
	// The PAC file should contain a javascript that the client will use to determine
	// which proxy to use for a given request.
	// URL example: http://wpad.example.com/wpad.dat
	// The client will learn the PAC file location using the DHCP option 252.
	// An alternative approach is to use DNS (with a DNSServer endpoint).
	WPAD string
}

// DNSClientConfig : DNS configuration for a client.
type DNSClientConfig struct {
	// PublicDNS : list of IP addresses of public DNS servers to announce via DHCP option 6.
	// For example: ["1.1.1.1", "8.8.8.8"]
	PublicDNS []string
	// PrivateDNS : list of DNS servers running as endpoints inside Eden SDN,
	// announced to clients via DHCP option 6.
	// The list should contain logical labels of those endpoints, not IP addresses!
	PrivateDNS []string
}

// MitMProxy is a proxy that both HTTP and HTTPS traffic is forwarded through
// transparently.
type MitMProxy struct {
	// CertPEM : Proxy certificate in the PEM format.
	CertPEM string
	// KeyPEM : Proxy key in the PEM format.
	KeyPEM string
}

// Router routing traffic for a network based on the reachability requirements.
type Router struct {
	// OutsideReachability : If enabled then it is possible to use the network to access
	// endpoints outside of Eden SDN (unless blocked by firewall).
	// This includes the controller (Adam, zedcloud), eserver (image cache) and the Internet.
	OutsideReachability bool
	// ReachableEndpoints : Logical labels of reachable endpoints.
	ReachableEndpoints []string
	// ReachableNetworks : Logical labels of reachable networks.
	ReachableNetworks []string
}

// Endpoints simulate "remote" clients and servers.
type Endpoints struct {
	// Clients : list of clients. Can be used to run requests towards EVE.
	Clients []Endpoint
	// DNSServers : list of DNS servers. Can be referenced in DHCP.PrivateDNS.
	DNSServers []DNSServer
	// NTPServers : list of NTP servers. Can be referenced in DHCP.PrivateNTP.
	NTPServers []NTPServer
	// HTTPServers : list of HTTP(s) servers. Can be used to test HTTP(s) connectivity
	// from EVE, to serve PAC files, etc.
	HTTPServers []HTTPServer
	// Proxies : list of HTTP(s) proxies that can be configured explicitly.
	// Consider using together with NetworkModel.Firewall, configured to block HTTP(s)
	// traffic that tries to bypass proxies.
	Proxies []ExplicitProxy
}

// Endpoint simulates "remote" client or a server.
type Endpoint struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string
	// FQDN : Fully qualified domain name of the endpoint.
	FQDN string
	// Subnet : network address + netmask (IPv4 or IPv6).
	// Subnet needs to fit at least two host IP addresses,
	// one for the endpoint, another for a gateway.
	Subnet string
	// IP should be inside of the Subnet.
	IP string
	// MTU of the endpoint's interface.
	MTU uint16
}

// DNSServer : endpoint providing DNS service.
type DNSServer struct {
	// Endpoint configuration.
	Endpoint
	// StaticEntries : list of FQDN->IP entries statically configured
	// for the server. These are typically used for endpoints running inside Eden-SDN,
	// which are obviously not known to the public DNS servers.
	StaticEntries []DNSEntry
	// UpstreamServers : list of IP addresses of public DNS servers to forward
	// requests to (unless there is a static entry).
	UpstreamServers []string
}

// DNSEntry : Mapping between FQDN and an IP address.
type DNSEntry struct {
	// FQDN : Fully qualified domain name.
	FQDN string
	// IP address or a special value that Eden SDN will automatically translate
	// to the corresponding IP address:
	//  - "endpoint.<endpoint-logical-label>" - translated to IP address of the endpoint
	//  - "adam" - translated to IP address on which Adam (open-source controller) is deployed and accessible
	IP string
}

// HTTPServer : HTTP(s) server.
type HTTPServer struct {
	// Endpoint configuration.
	Endpoint
	// DNSClientConfig : DNS configuration to be applied for the HTTP server.
	DNSClientConfig
	// HTTPPort : port to listen for HTTP requests.
	// Zero value can be used to disable HTTP.
	HTTPPort uint16
	// HTTPSPort : port to listen for HTTPS requests.
	// Zero value can be used to disable HTTPS.
	HTTPSPort uint16
	// CertPEM : Server certificate in the PEM format. Required for HTTPS.
	CertPEM string
	// KeyPEM : Server key in the PEM format. Required for HTTPS.
	KeyPEM string
	// Maps URL Path to a content to be returned inside the HTTP(s) response body
	// (text/plain content type).
	Paths map[string]HTTPContent
}

// HTTPContent : content returned by an HTTP(s) handler.
type HTTPContent struct {
	// ContentType : HTTP(S) Content-Type.
	ContentType string
	// Content : content returned inside a HTTP(s) response body.
	// It is a string, so binary content is not possible for now.
	Content string
}

// NTPServer : NTP server.
type NTPServer struct {
	// Endpoint configuration.
	Endpoint
	// List of (public) NTP servers to synchronize with, each referenced
	// by an IP address or a FQDN.
	UpstreamServers []string
}

// ExplicitProxy : HTTP(S) proxy configured explicitly.
type ExplicitProxy struct {
	// HTTPServer : configuration for the underlying HTTP(S) server.
	// Note that parameter HTTPServer.Paths is ignored here.
	HTTPServer
	// Users : define for username/password authentication, leave empty otherwise.
	Users []UserCredentials
}

// UserCredentials : User credentials for an explicit proxy.
type UserCredentials struct {
	// Username
	Username string
	// Password
	Password string
}

// Note that traffic not matched by any rule is allowed!
type Firewall struct {
	// Rules : firewall rules applied in the order as configured.
	Rules FwRule
}

// FwRule : a firewall rule.
type FwRule struct {
	// SrcSubnet : subnet to match the source IP address with.
	SrcSubnet string
	// DstSubnet : subnet to match the destination IP address with.
	DstSubnet string
	// Protocol : filter by protocol.
	Protocol FwProto
	// Ports : list of destination port to which the rule applies.
	// Empty = any.
	Ports []uint16
	// Action to take.
	Action FwAction
}

// HostConfig : host configuration that Eden-SDN needs to be informed about.
type HostConfig struct {
	// HostIPs : list of IP addresses used by the host system (on top of which
	// Eden runs).
	// Eden SDN requires at least one routable host IP address.
	HostIPs []string
	// NetworkType : which IP versions are used by the host.
	// Even if host uses IPv4 only, it is still possible to have IPv6 inside
	// Eden-SDN. Connectivity between EVE (IPv6) and the controller (IPv4) is established
	// automatically using DNS64 and NAT64. However, the opposite case (from IPv4 to IPv6)
	// is not supported. In other words, to test EVE with IPv4, it is required for the host
	// to use IPv4 (single or dual stack).
	NetworkType NetworkType
}

// IPRange : a range of IP addresses.
type IPRange struct {
	// FromIP : start of the range (includes the address itself).
	FromIP string
	// FromIP : end of the range (includes the address itself).
	ToIP string
}

// NetworkType : type of the network wrt. IP version used.
type NetworkType uint8

const (
	// Ipv4Only : host uses IPv4 only.
	Ipv4Only NetworkType = iota
	// Ipv6Only : host uses IPv6 only.
	Ipv6Only
	// DualStack : host tuns with dual stack.
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
	// Allow traffic.
	Allow FwAction = iota
	// Deny traffic.
	Deny
)

// FwActionToString : convert FwAction to string representation used in JSON.
var FwActionToString = map[FwAction]string{
	Allow: "allow",
	Deny:  "deny",
}

// FwActionToID : get FwAction from a string representation.
var FwActionToID = map[string]FwAction{
	"":      Allow, // default value
	"allow": Allow,
	"deny":  Deny,
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
