package api

import (
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

// ItemType
func (e Endpoint) ItemType() string {
	return "endpoint"
}

// ItemLogicalLabel
func (e Endpoint) ItemLogicalLabel() string {
	return e.LogicalLabel
}

// ReferencesFromItem (overshadowed by endpoint specializations).
func (e Endpoint) ReferencesFromItem() []LogicalLabelRef {
	return nil
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
	var refs []LogicalLabelRef
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
	// EndpointIPRefPrefix : prefix used to symbolically reference endpoint IP
	// (instead of directly entering the IP address).
	// Can be used in DNSEntry.IP.
	EndpointIPRefPrefix = "endpoint-ip." // Followed by the endpoint logical label.
	// AdamIPRef : string used to symbolically reference adam IP address.
	// Can be used in DNSEntry.IP.
	AdamIPRef = "adam-ip"
)

// DNSEntry : Mapping between FQDN and an IP address.
type DNSEntry struct {
	// FQDN : Fully qualified domain name.
	// Can be a reference to endpoint FQDN:
	//  - "endpoint-fqdn.<endpoint-logical-label>" - translated to endpoint's FQDN by Eden-SDN
	FQDN string `json:"fqdn"`
	// IP address or a special value that Eden-SDN will automatically translate
	// to the corresponding IP address:
	//  - "endpoint-ip.<endpoint-logical-label>" - translated to IP address of the endpoint
	//  - "adam-ip" - translated to IP address on which Adam (open-source controller) is deployed and accessible
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

// ItemCategory
func (e HTTPServer) ItemCategory() string {
	return "http-server"
}

// ReferencesFromItem
func (e HTTPServer) ReferencesFromItem() []LogicalLabelRef {
	var refs []LogicalLabelRef
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

// ItemCategory
func (e ExplicitProxy) ItemCategory() string {
	return "explicit-proxy"
}

// ReferencesFromItem
func (e ExplicitProxy) ReferencesFromItem() []LogicalLabelRef {
	var refs []LogicalLabelRef
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

// UserCredentials : User credentials for an explicit proxy.
type UserCredentials struct {
	// Username
	Username string `json:"username"`
	// Password
	Password string `json:"password"`
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
