package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Endpoints simulate "remote" clients and servers.
type Endpoints struct {
	// Clients : list of clients. Can be used to run requests towards EVE.
	Clients []Client `json:"clients,omitempty"`
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
	// TransparentProxies are proxying both HTTP and HTTPS traffic transparently.
	TransparentProxies []TransparentProxy `json:"transparentProxies,omitempty"`
	// NetbootServers : HTTP/TFTP servers providing artifacts needed to boot EVE OS
	// over a network (using netboot/PXE + iPXE).
	NetbootServers []NetbootServer `json:"netbootServers,omitempty"`
}

// GetAll : returns all endpoints as one list.
// Returned list items are of type Endpoint, which is a common struct embedded
// inside each endpoint.
func (eps Endpoints) GetAll() (all []Endpoint) {
	for _, client := range eps.Clients {
		all = append(all, client.Endpoint)
	}
	for _, dnsSrv := range eps.DNSServers {
		all = append(all, dnsSrv.Endpoint)
	}
	for _, ntpSrv := range eps.NTPServers {
		all = append(all, ntpSrv.Endpoint)
	}
	for _, httpSrv := range eps.HTTPServers {
		all = append(all, httpSrv.Endpoint)
	}
	for _, exProxy := range eps.ExplicitProxies {
		all = append(all, exProxy.Endpoint)
	}
	for _, tProxy := range eps.TransparentProxies {
		all = append(all, tProxy.Endpoint)
	}
	for _, netBootSrv := range eps.NetbootServers {
		all = append(all, netBootSrv.Endpoint)
	}
	return all
}

// Endpoint simulates "remote" client or a server.
type Endpoint struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string `json:"logicalLabel"`
	// FQDN : Fully qualified domain name of the endpoint.
	FQDN string `json:"fqdn"`
	// Single-stack endpoint IP (v4 or v6) configuration.
	// Define either this or DualStack.
	EndpointIPConfig
	// Dual-stack endpoint IP configuration.
	// Define either this or the (single-stack) embedded EndpointIPConfig.
	DualStack DualStackEndpoint `json:"dualStack"`
	// DirectL2Connect : configure direct L2 connectivity between the endpoint and EVE.
	// Use alternatively or additionally to Subnet+IP options.
	DirectL2Connect DirectL2EpConnect `json:"directL2Connect"`
	// MTU of the endpoint's interface.
	// If not defined (zero value), the default MTU for Ethernet, which is 1500 bytes,
	// will be set.
	MTU uint16 `json:"mtu"`
}

// IsDualStack returns true if Endpoint is configured to operate in dual-stack IP mode.
func (e Endpoint) IsDualStack() bool {
	return e.DualStack.IPv4.Subnet != "" || e.DualStack.IPv6.Subnet != ""
}

// EndpointIPConfig : IP configuration for Endpoint.
type EndpointIPConfig struct {
	// Subnet : network address + netmask (IPv4 or IPv6).
	// Subnet needs to fit at least two host IP addresses,
	// one for the endpoint, another for a gateway.
	Subnet string `json:"subnet"`
	// IP should be inside the Subnet.
	IP string `json:"ip"`
}

// DualStackEndpoint : dual-stack IP configuration for Endpoint.
type DualStackEndpoint struct {
	// IPv4 config for Endpoint.
	IPv4 EndpointIPConfig `json:"ipv4"`
	// IPv6 config for Endpoint.
	IPv6 EndpointIPConfig `json:"ipv6"`
}

// ItemType
func (e Endpoint) ItemType() string {
	return "endpoint"
}

// ItemLogicalLabel
func (e Endpoint) ItemLogicalLabel() string {
	return e.LogicalLabel
}

// EndpointBridgeRefPrefix : prefix used for references to bridges from endpoints.
const EndpointBridgeRefPrefix = "bridge-endpoint-"

// ReferencesFromItem can be further extended by endpoint specializations.
func (e Endpoint) ReferencesFromItem() []LogicalLabelRef {
	var refs []LogicalLabelRef
	if e.DirectL2Connect.Bridge != "" {
		refs = append(refs, LogicalLabelRef{
			ItemType:         Bridge{}.ItemType(),
			ItemLogicalLabel: e.DirectL2Connect.Bridge,
			RefKey:           EndpointBridgeRefPrefix + e.LogicalLabel,
		})
	}
	return refs
}

// DirectL2EpConnect : direct L2 connection between an endpoint and EVE.
type DirectL2EpConnect struct {
	// Logical label of a Bridge to which the endpoint is connected.
	Bridge string `json:"bridge"`
	// Access VLAN ID.
	// Leave zero value to express intent of not using VLAN filtering for this endpoint.
	VlanID uint16 `json:"vlanID"`
}

// Client emulates a remote client.
// Can be used to run requests towards EVE.
type Client struct {
	// Endpoint configuration.
	Endpoint
}

// ItemCategory
func (e Client) ItemCategory() string {
	return "client"
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

// ItemCategory
func (e DNSServer) ItemCategory() string {
	return "dns-server"
}

// ReferencesFromItem
func (e DNSServer) ReferencesFromItem() []LogicalLabelRef {
	refs := e.Endpoint.ReferencesFromItem()
	for i, entry := range e.StaticEntries {
		if strings.HasPrefix(entry.FQDN, EndpointFQDNRefPrefix) {
			refKey := fmt.Sprintf("dns-server-%s-entry-%d-fqdn", e.LogicalLabel, i)
			logicalLabel := strings.TrimPrefix(entry.FQDN, EndpointFQDNRefPrefix)
			refs = append(refs, LogicalLabelRef{
				ItemType:         Endpoint{}.ItemType(),
				ItemLogicalLabel: logicalLabel,
				RefKey:           refKey,
			})
		}
		if strings.HasPrefix(entry.IP, EndpointIPRefPrefix) {
			refKey := fmt.Sprintf("dns-server-%s-entry-%d-ip", e.LogicalLabel, i)
			logicalLabel := strings.TrimPrefix(entry.IP, EndpointIPRefPrefix)
			refs = append(refs, LogicalLabelRef{
				ItemType:         Endpoint{}.ItemType(),
				ItemLogicalLabel: logicalLabel,
				RefKey:           refKey,
			})
		}
	}
	return refs
}

const (
	// EndpointFQDNRefPrefix : prefix used to symbolically reference endpoint FQDN
	// (instead of directly entering the FQDN).
	// Can be used in DNSEntry.FQDN.
	EndpointFQDNRefPrefix = "endpoint-fqdn." // Followed by the endpoint logical label.
	// EndpointIPRefPrefix : prefix used to symbolically reference endpoint IP address(es).
	// (instead of directly entering the IP address).
	// Translates to both IPv4 and IPv6 address if Endpoint runs in dual-stack mode.
	// Can be used in DNSEntry.IP.
	EndpointIPRefPrefix = "endpoint-ip." // Followed by the endpoint logical label.
	// EndpointIPv4RefPrefix : prefix used to symbolically reference endpoint IPv4 address.
	EndpointIPv4RefPrefix = "endpoint-ipv4." // Followed by the endpoint logical label.
	// EndpointIPv6RefPrefix : prefix used to symbolically reference endpoint IPv6 address.
	EndpointIPv6RefPrefix = "endpoint-ipv6." // Followed by the endpoint logical label.
	// AdamIPRef : string used to symbolically reference adam IP address(es).
	// Translates to both IPv4 and IPv6 address if Adam runs in dual-stack mode.
	// Can be used in DNSEntry.IP.
	AdamIPRef = "adam-ip"
	// AdamIPv4Ref : string used to symbolically reference Adam IPv4 address.
	AdamIPv4Ref = "adam-ipv4"
	// AdamIPv6Ref : string used to symbolically reference Adam IPv6 address.
	// Can be used in DNSEntry.IP.
	AdamIPv6Ref = "adam-ipv6"
)

// DNSEntry : Mapping between FQDN and an IP address.
type DNSEntry struct {
	// FQDN : Fully qualified domain name.
	// Can be a reference to endpoint FQDN:
	//  - "endpoint-fqdn.<endpoint-logical-label>" - translated to endpoint's FQDN by Eden-SDN
	FQDN string `json:"fqdn"`
	// IP address or a special value that Eden-SDN will automatically translate
	// to the corresponding IP address:
	//  - "endpoint-ip.<endpoint-logical-label>"
	//        - translated to IP address(es) of the endpoint
	//        - translates to both IPv4 and IPv6 addresses if endpoint runs in dual-stack mode
	//  - "endpoint-ipv4.<endpoint-logical-label>"
	//        - translated to IPv4 address of the endpoint
	//  - "endpoint-ipv6.<endpoint-logical-label>"
	//        - translated to IPv6 address of the endpoint
	//  - "adam-ip"
	//        - translated to IP address(es) on which Adam (open-source controller)
	//          is deployed and accessible
	//        - translates to both IPv4 and IPv6 addresses if Adam runs in dual-stack mode
	//  - "adam-ipv4"
	//        - translated to IPv4 address of Adam controller
	//  - "adam-ipv6"
	//        - translated to IPv6 address of Adam controller
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
	// Maps URL Path to a content to be returned inside the HTTP(s) response body.
	Paths map[string]HTTPContent `json:"paths"`
}

// ItemCategory
func (e HTTPServer) ItemCategory() string {
	return "http-server"
}

// ReferencesFromItem
func (e HTTPServer) ReferencesFromItem() []LogicalLabelRef {
	refs := e.Endpoint.ReferencesFromItem()
	for _, dns := range e.PrivateDNS {
		refs = append(refs, LogicalLabelRef{
			ItemType:         Endpoint{}.ItemType(),
			ItemCategory:     DNSServer{}.ItemCategory(),
			ItemLogicalLabel: dns,
			// Avoids duplicate DNS servers within the same HTTP server.
			RefKey: "http-server-" + e.LogicalLabel,
		})
	}
	return refs
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

// ItemCategory
func (e NTPServer) ItemCategory() string {
	return "ntp-server"
}

// ExplicitProxy : HTTP(S) proxy configured explicitly.
type ExplicitProxy struct {
	// Endpoint configuration.
	Endpoint
	// Proxy configuration (common to transparent and explicit proxies).
	Proxy
	// HTTPProxy : enable HTTP proxy (i.e. proxying of HTTP traffic) and specify
	// on which port+protocol to listen for proxy requests (can be HTTP or HTTPS).
	// Zero port number can be used to disable HTTP proxy.
	HTTPProxy ProxyPort `json:"httpProxy"`
	// HTTPSProxy : enable HTTPS proxy (i.e. proxying of HTTPS traffic) and specify
	// on which port+protocol to listen for proxy requests (can be HTTP or HTTPS).
	// Zero port number can be used to disable HTTPS proxy.
	HTTPSProxy ProxyPort `json:"httpsProxy"`
	// Users : define for username/password authentication, leave empty otherwise.
	Users []UserCredentials `json:"users"`
}

// ProxyPort : port+protocol used to *listen* for incoming request for proxying.
// Note that it can differ from protocol that is being proxied.
type ProxyPort struct {
	// Port : port number on which the HTTP/HTTPS proxy listens.
	Port uint16 `json:"port"`
	// ListenProto : protocol used to listen for incoming request for proxying
	// (not necessary the protocol which is then being proxied)
	ListenProto ProxyListenProto `json:"listenProto"`
}

// ItemCategory
func (e ExplicitProxy) ItemCategory() string {
	return "explicit-proxy"
}

// ReferencesFromItem
func (e ExplicitProxy) ReferencesFromItem() []LogicalLabelRef {
	refs := e.Endpoint.ReferencesFromItem()
	for _, dns := range e.PrivateDNS {
		refs = append(refs, LogicalLabelRef{
			ItemType:         Endpoint{}.ItemType(),
			ItemCategory:     DNSServer{}.ItemCategory(),
			ItemLogicalLabel: dns,
			// Avoids duplicate DNS servers within the same explicit proxy.
			RefKey: "explicit-proxy-" + e.LogicalLabel,
		})
	}
	return refs
}

// Proxy can be either transparent or configured explicitly.
type Proxy struct {
	// DNSClientConfig : DNS configuration to be applied for the proxy.
	DNSClientConfig
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

// UserCredentials : User credentials for an explicit proxy.
type UserCredentials struct {
	// Username
	Username string `json:"username"`
	// Password
	Password string `json:"password"`
}

// TransparentProxy is a proxy that both HTTP and HTTPS traffic is forwarded through
// transparently.
type TransparentProxy struct {
	// Endpoint configuration.
	Endpoint
	// Proxy configuration (common to transparent and explicit proxies).
	Proxy
}

// ItemCategory categorizes the item within Endpoints.
func (e TransparentProxy) ItemCategory() string {
	return "transparent-proxy"
}

// ReferencesFromItem lists references to private DNS servers (if there are any).
func (e TransparentProxy) ReferencesFromItem() []LogicalLabelRef {
	refs := e.Endpoint.ReferencesFromItem()
	for _, dns := range e.PrivateDNS {
		refs = append(refs, LogicalLabelRef{
			ItemType:         Endpoint{}.ItemType(),
			ItemCategory:     DNSServer{}.ItemCategory(),
			ItemLogicalLabel: dns,
			// Avoids duplicate DNS servers within the same transparent proxy.
			RefKey: "transparent-proxy-" + e.LogicalLabel,
		})
	}
	return refs
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
//
//	# Boot for iPXE. The idea is to send two different
//	# filenames, the first loads iPXE, and the second tells iPXE what to
//	# load. The dhcp-match sets the ipxe tag for requests from iPXE.
//	#dhcp-boot=undionly.kpxe
//	#dhcp-match=set:ipxe,175 # iPXE sends a 175 option.
//	#dhcp-boot=tag:ipxe,http://boot.ipxe.org/demo/boot.php
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

// ItemCategory
func (e NetbootServer) ItemCategory() string {
	return "netboot-server"
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

// ProxyListenProto : protocol used to listen for incoming requests for proxying.
type ProxyListenProto uint8

const (
	// ProxyListenProtoUnspecified : protocol is not specified.
	// Used for transparent proxy where the protocol is implicit.
	// With explicit proxy Eden-SDN assumes HTTP as the default.
	ProxyListenProtoUnspecified ProxyListenProto = iota
	// ProxyListenProtoHTTP : proxy listens on HTTP for new proxy requests.
	ProxyListenProtoHTTP
	// ProxyListenProtoHTTPS : proxy listens on HTTPS for new proxy requests.
	ProxyListenProtoHTTPS
)

// ProxyListenProtoToString : convert ProxyListenProto to string representation
// used in JSON.
var ProxyListenProtoToString = map[ProxyListenProto]string{
	ProxyListenProtoUnspecified: "",
	ProxyListenProtoHTTP:        "http",
	ProxyListenProtoHTTPS:       "https",
}

// ProxyListenProtoToID : get ProxyListenProto from a string representation.
var ProxyListenProtoToID = map[string]ProxyListenProto{
	"":      ProxyListenProtoUnspecified,
	"http":  ProxyListenProtoHTTP,
	"https": ProxyListenProtoHTTPS,
}

// MarshalJSON marshals the enum as a quoted json string.
func (s ProxyListenProto) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(ProxyListenProtoToString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON un-marshals a quoted json string to the enum value.
func (s *ProxyListenProto) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = ProxyListenProtoToID[j]
	return nil
}
