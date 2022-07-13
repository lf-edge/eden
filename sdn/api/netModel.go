package api

import (
	"bytes"
	"encoding/json"
)

// NetworkModel is used to declaratively describe the intended state of the networking
// around EVE VM(s) for the testing purposes. The model is submitted in the JSON format to Eden-SDN Agent,
// running inside a separate VM, connected to EVE VM(s) via inter-VM network interfaces, and emulating
// the desired networking using the Linux network stack, network namespaces, netfilter and with
// the help of several open-source projects, such as dnsmasq, radvd, goproxy, etc.
type NetworkModel struct {
	// Ports : network interfaces connecting EVE VM(s) with Eden-SDN VM.
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

// Port is a network interface connecting EVE VM with Eden-SDN VM.
type Port struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string `json:"logicalLabel"`
	// MAC address assigned to the interface on the SDN side.
	// If not specified by the user, Eden will generate a random MAC address.
	MAC string `json:"mac"`
	// MTU : Maximum transmission unit (for the SDN side).
	MTU uint16 `json:"mtu"`
	// AdminUP : whether the interface should be UP on the SDN side.
	// Put down to test link-down scenarios on EVE.
	AdminUP bool `json:"adminUP"`
	// EVEConnect : plug the other side of the port into a given EVE instance.
	EVEConnect *EVEConnect `json:"eveConnect"`
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
	// IfOrder determines order of ports connected to the same EVE instance.
	// Port with the lowest IfOrder will be first (eth0), port with the second
	// lowest IfOrder will appear as second (eth1), etc.
	IfOrder uint8 `json:"ifOrder"`
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
}

// Network provides L3 connectivity.
type Network struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string `json:"logicalLabel"`
	// Logical label of a Bridge to which the network is attached.
	Bridge string `json:"bridge"`
	// Leave zero value to express intent of not using VLAN for this network.
	VlanID uint16 `json:"vlanID"`
	// Subnet : network address + netmask (IPv4 or IPv6).
	Subnet string `json:"subnet"`
	// GwIP should be inside the Subnet.
	GwIP string `json:"gwIP"`
	// DHCP configuration.
	DHCP DHCP `json:"dhcp"`
	// TransparentProxy is a proxy that both HTTP and HTTPS traffic is forwarded through
	// transparently, before it reaches router, endpoints, firewall, etc.
	TransparentProxy *Proxy `json:"transparentProxy,omitempty"`
	// Router configuration. Every network has a separate routing context.
	// Undefined (nil) means that everything should be routed and accessible.
	// That includes all networks, endpoints and the outside of Eden SDN.
	Router *Router `json:"router,omitempty"`
}

// DHCP configuration.
// Note that for IPv6 we prefer to use Router Advertisement over DHCPv6 to publish
// all this information to hosts on the network.
// But DHCPv6 is still needed and used to convey NTP and netboot configuration (if provided).
type DHCP struct {
	// Enable DHCP. Set to false to use EVE with static IP addressing.
	Enable bool `json:"enable"`
	// IPRange : a range of IP addresses to allocate from.
	// Not applicable for IPv6.
	IPRange IPRange `json:"ipRange"`
	// DomainName : name of the domain assigned to the network.
	// It is propagated to clients using the DHCP option 15.
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
	// An alternative approach is to use DNS (with a DNSServer endpoint).
	WPAD string `json:"wpad"`
	// NetbootServer : Logical label of a NetbootServer endpoint which the client should use
	// to boot EVE OS from. The IP address or FQDN and the provisioning file (iPXE script)
	// location will be announced to the client using DHCP options 66 and 67 (59 in DHCPv6).
	// Eden-SDN will announce either IP address or FQDN depending on whether any of the assigned
	// private DNS servers is able to resolve the NetbootServer domain name.
	NetbootServer string `json:"netbootServer"`
}

// DNSClientConfig : DNS configuration for a client.
type DNSClientConfig struct {
	// PublicDNS : list of IP addresses of public DNS servers to announce via DHCP option 6.
	// For example: ["1.1.1.1", "8.8.8.8"]
	PublicDNS []string `json:"publicDNS"`
	// PrivateDNS : list of DNS servers running as endpoints inside Eden SDN,
	// announced to clients via DHCP option 6.
	// The list should contain logical labels of those endpoints, not IP addresses!
	PrivateDNS []string `json:"privateDNS"`
}

// Proxy can be either transparent or configured explicitly.
type Proxy struct {
	// CertPEM : Proxy certificate of the certificate authority in the PEM format.
	// Proxy will use CA cert to sign certificate that it generates for itself.
	// EVE should be configured to trust CA certificate.
	// Not needed if proxy is just forwarding all flows (i.e. not terminating TLS).
	CACertPEM string `json:"caCertPEM"`
	// CAKeyPEM : Proxy key of the certificate authority in the PEM format.
	// Proxy will use CA cert to sign certificate that it generates for itself.
	// EVE should be configured to trust CA certificate.
	// Not needed if proxy is just forwarding all flows (i.e. not terminating TLS).
	CAKeyPEM string `json:"caKeyPEM"`
	// ProxyRules : a set of rules that decides what to do with proxied traffic.
	// By default (no rules defined), proxy will just forward all the flows.
	ProxyRules []ProxyRule `json:"proxyRules"`
}

// ProxyRule : rule used by a proxy, which, if matches a given flow, decides what
// to do with it.
type ProxyRule struct {
	// ReqHost : host from HTTP request header (or from the SNI value in the TLS ClientHello)
	// to match this rule with (e.g. "google.com").
	// Empty ReqHost should be used with the default rule (put one at most).
	ReqHost string `json:"reqHost"`
	// Action to take.
	Action ProxyAction `json:"action"`
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
}

// Endpoints simulate "remote" clients and servers.
type Endpoints struct {
	// Clients : list of clients. Can be used to run requests towards EVE.
	Clients []Endpoint `json:"clients,omitempty"`
	// DNSServers : list of DNS servers. Can be referenced in DHCP.PrivateDNS.
	DNSServers []DNSServer `json:"dnsServers,omitempty"`
	// NTPServers : list of NTP servers. Can be referenced in DHCP.PrivateNTP.
	NTPServers []NTPServer `json:"ntpServers,omitempty"`
	// HTTPServers : list of HTTP(s) servers. Can be used to test HTTP(s) connectivity
	// from EVE, to serve PAC files, etc.
	HTTPServers []HTTPServer `json:"httpServers,omitempty"`
	// ExplicitProxies : proxies that must be configured explicitly.
	// Consider using together with NetworkModel.Firewall, configured to block HTTP(s)
	// traffic that tries to bypass a proxy.
	ExplicitProxies []ExplicitProxy `json:"explicitProxies,omitempty"`
	// NetbootServers : HTTP/TFTP servers providing artifacts needed to boot EVE OS
	// over a network (using netboot/PXE + iPXE).
	NetbootServers []NetbootServer `json:"netbootServers,omitempty"`
}

// Endpoint simulates "remote" client or a server.
type Endpoint struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string `json:"logicalLabel"`
	// FQDN : Fully qualified domain name of the endpoint.
	FQDN string `json:"fqdn"`
	// Subnet : network address + netmask (IPv4 or IPv6).
	// Subnet needs to fit at least two host IP addresses,
	// one for the endpoint, another for a gateway.
	Subnet string `json:"subnet"`
	// IP should be inside of the Subnet.
	IP string `json:"ip"`
	// MTU of the endpoint's interface.
	MTU uint16 `json:"mtu"`
}

// DNSServer : endpoint providing DNS service.
type DNSServer struct {
	// Endpoint configuration.
	Endpoint
	// StaticEntries : list of FQDN->IP entries statically configured
	// for the server. These are typically used for endpoints running inside Eden-SDN,
	// which are obviously not known to the public DNS servers.
	StaticEntries []DNSEntry `json:"staticEntries"`
	// UpstreamServers : list of IP addresses of public DNS servers to forward
	// requests to (unless there is a static entry).
	UpstreamServers []string `json:"upstreamServers"`
}

// DNSEntry : Mapping between FQDN and an IP address.
type DNSEntry struct {
	// FQDN : Fully qualified domain name.
	FQDN string `json:"fqdn"`
	// IP address or a special value that Eden SDN will automatically translate
	// to the corresponding IP address:
	//  - "endpoint.<endpoint-logical-label>" - translated to IP address of the endpoint
	//  - "adam" - translated to IP address on which Adam (open-source controller) is deployed and accessible
	IP string `json:"ip"`
}

// HTTPServer : HTTP(s) server.
type HTTPServer struct {
	// Endpoint configuration.
	Endpoint
	// DNSClientConfig : DNS configuration to be applied for the HTTP server.
	DNSClientConfig
	// HTTPPort : port to listen for HTTP requests.
	// Zero value can be used to disable HTTP.
	HTTPPort uint16 `json:"httpPort"`
	// HTTPSPort : port to listen for HTTPS requests.
	// Zero value can be used to disable HTTPS.
	HTTPSPort uint16 `json:"httpsPort"`
	// CertPEM : Server certificate in the PEM format. Required for HTTPS.
	CertPEM string `json:"certPEM"`
	// KeyPEM : Server key in the PEM format. Required for HTTPS.
	KeyPEM string `json:"keyPEM"`
	// Maps URL Path to a content to be returned inside the HTTP(s) response body
	// (text/plain content type).
	Paths map[string]HTTPContent `json:"paths"`
}

// HTTPContent : content returned by an HTTP(s) handler.
type HTTPContent struct {
	// ContentType : HTTP(S) Content-Type.
	ContentType string `json:"contentType"`
	// Content : content returned inside a HTTP(s) response body.
	// It is a string, so binary content is not possible for now.
	Content string `json:"content"`
}

// NTPServer : NTP server.
type NTPServer struct {
	// Endpoint configuration.
	Endpoint
	// List of (public) NTP servers to synchronize with, each referenced
	// by an IP address or a FQDN.
	UpstreamServers []string `json:"upstreamServers"`
}

// ExplicitProxy : HTTP(S) proxy configured explicitly.
type ExplicitProxy struct {
	// Endpoint configuration.
	Endpoint
	// Proxy configuration (common to transparent and explicit proxies).
	Proxy
	// DNSClientConfig : DNS configuration to be applied for the proxy.
	DNSClientConfig
	// HTTPPort : HTTP proxy port.
	// Zero value can be used to disable HTTP proxy.
	HTTPPort uint16 `json:"httpPort"`
	// HTTPSPort : HTTPS proxy port.
	// Zero value can be used to disable HTTPS proxy.
	HTTPSPort uint16 `json:"httpsPort"`
	// Users : define for username/password authentication, leave empty otherwise.
	Users []UserCredentials `json:"users"`
}

// UserCredentials : User credentials for an explicit proxy.
type UserCredentials struct {
	// Username
	Username string `json:"username"`
	// Password
	Password string `json:"password"`
}

// Note that traffic not matched by any rule is allowed!
type Firewall struct {
	// Rules : firewall rules applied in the order as configured.
	Rules []FwRule `json:"rules"`
}

// FwRule : a firewall rule.
type FwRule struct {
	// SrcSubnet : subnet to match the source IP address with.
	SrcSubnet string `json:"srcSubnet"`
	// DstSubnet : subnet to match the destination IP address with.
	DstSubnet string `json:"dstSubnet"`
	// Protocol : filter by protocol.
	Protocol FwProto `json:"protocol"`
	// Ports : list of destination port to which the rule applies.
	// Empty = any.
	Ports []uint16 `json:"ports"`
	// Action to take.
	Action FwAction `json:"action"`
}

// NetbootServer provides HTTP and TFTP server endpoints, serving all artifacts
// needed to boot EVE OS over a network (using iPXE, potentially also supporting
// older PXE-only clients).
// Use in combination with DHCP (see DHCP.NetbootServer).
// XXX Note that in all likelihood, TFTP will serve an iPXE (UEFI) bootloader,
// that once booted will download and boot EVE artifacts over the HTTP endpoint.
// This can work with only a little magic in the DHCP server configuration,
// known as chainloading [1].
// If a client only understands netboot/PXE, DHCP will point the client first
// to the TFTP endpoint. Once the client has booted iPXE, it will be directed
// by the DHCP server to the iPXE script from the HTTP endpoint (just like any other
// iPXE-enabled client).
//
// Example config for dnsmasq:
//   # Boot for iPXE. The idea is to send two different
//   # filenames, the first loads iPXE, and the second tells iPXE what to
//   # load. The dhcp-match sets the ipxe tag for requests from iPXE.
//   #dhcp-boot=undionly.kpxe
//   #dhcp-match=set:ipxe,175 # iPXE sends a 175 option.
//   #dhcp-boot=tag:ipxe,http://boot.ipxe.org/demo/boot.php
//
// [1] https://ipxe.org/howto/chainloading
type NetbootServer struct {
	// Endpoint configuration.
	Endpoint
	// TFTPArtifacts : boot artifacts served by the TFTP server.
	// If not specified, Eden will automatically put iPXE bootloader
	// as the entrypoint.
	TFTPArtifacts []NetbootArtifact `json:"tftpArtifacts"`
	// HTTPArtifacts : boot artifacts served by the HTTP server.
	// If not specified, Eden will automatically put iPXE artifacts
	// needed to boot EVE OS (as links to eserver where these artifacts
	// are uploaded).
	HTTPArtifacts []NetbootArtifact `json:"httpArtifacts"`
}

// NetbootArtifact - one of the artifacts used to boot EVE OS over a network.
type NetbootArtifact struct {
	// Filename : name of the file.
	// It will be served by the associated NetbootServer at the endpoint:
	// (http|tftp)://<netboot-server-fqdn>/<Filename>
	Filename string `json:"filename"`
	// DownloadFromURL : HTTP URL from where the artifact will be downloaded
	// by the netboot server. It can for example point to the eserver.
	// Note that Netboot server will forward the artifact content, not redirect
	// to this URL.
	DownloadFromURL string `json:"downloadFromURL"`
	// Entrypoint : Is this the entrypoint for netboot (i.e. the artifact to boot from)?
	// In case of iPXE, this would be enabled for the inital iPXE script.
	// Exactly one NetbootArtifact should be marked as entrypoint inside
	// both lists NetbootServer.TFTPArtifacts and NetbootServer.HTTPArtifacts.
	// If enabled, this file is then announced to netboot clients using DHCP
	// option 67 (59 in DHCPv6).
	Entrypoint bool `json:"entrypoint"`
}

// HostConfig : host configuration that Eden-SDN needs to be informed about.
type HostConfig struct {
	// HostIPs : list of IP addresses used by the host system (on top of which
	// Eden runs).
	// Eden SDN requires at least one routable host IP address.
	HostIPs []string `json:"hostIPs"`
	// NetworkType : which IP versions are used by the host.
	// Even if host uses IPv4 only, it is still possible to have IPv6 inside
	// Eden-SDN. Connectivity between EVE (IPv6) and the controller (IPv4) is established
	// automatically using DNS64 and NAT64. However, the opposite case (from IPv4 to IPv6)
	// is not supported. In other words, to test EVE with IPv4, it is required for the host
	// to use IPv4 (single or dual stack).
	NetworkType NetworkType `json:"networkType"`
}

// IPRange : a range of IP addresses.
type IPRange struct {
	// FromIP : start of the range (includes the address itself).
	FromIP string `json:"fromIP"`
	// FromIP : end of the range (includes the address itself).
	ToIP string `json:"toIP"`
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

// ProxyAction : proxy action.
type ProxyAction uint8

const (
	// PxForward : just forward proxied traffic.
	PxForward ProxyAction = iota
	// PxReject : reject (block) proxied traffic.
	PxReject
	// PxMITM : act as a man-in-the-middle (split TLS tunnel in two).
	PxMITM
)

// ProxyActionToString : convert ProxyAction to string representation used in JSON.
var ProxyActionToString = map[ProxyAction]string{
	PxForward: "forward",
	PxReject:  "reject",
	PxMITM:    "mitm",
}

// ProxyActionToID : get ProxyAction from a string representation.
var ProxyActionToID = map[string]ProxyAction{
	"":        PxForward, // default value
	"forward": PxForward,
	"reject":  PxReject,
	"mitm":    PxMITM,
}

// MarshalJSON marshals the enum as a quoted json string.
func (s ProxyAction) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(ProxyActionToString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON un-marshals a quoted json string to the enum value.
func (s *ProxyAction) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = ProxyActionToID[j]
	return nil
}

// FwAction : firewall action.
type FwAction uint8

const (
	// Allow traffic.
	FwAllow FwAction = iota
	// Deny traffic.
	FwDeny
)

// FwActionToString : convert FwAction to string representation used in JSON.
var FwActionToString = map[FwAction]string{
	FwAllow: "allow",
	FwDeny:  "deny",
}

// FwActionToID : get FwAction from a string representation.
var FwActionToID = map[string]FwAction{
	"":      FwAllow, // default value
	"allow": FwAllow,
	"deny":  FwDeny,
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
