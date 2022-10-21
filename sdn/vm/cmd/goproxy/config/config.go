package config

import (
	sdnapi "github.com/lf-edge/eden/sdn/vm/api"
)

// ProxyConfig : proxy configuration formatted with JSON and passed to goproxy
// using the "-c" command line argument.
type ProxyConfig struct {
	// ListenIP : IP address to listen on.
	// Leave empty to listen on all available interfaces instead of just
	// the interface with the given host address.
	ListenIP string `json:"listenIP"`
	// Hostname : domain name of the proxy.
	Hostname string `json:"hostname"`
	// HTTPPort : specify on which port+protocol to listen for requests
	// to proxy HTTP traffic.
	// Zero port number can be used to disable HTTP proxying.
	HTTPPort sdnapi.ProxyPort `json:"httpPort"`
	// HTTPSPorts : specify on which port(s)+protocol(s) to listen
	// for requests to proxy HTTPS traffic.
	// Empty list can be used to disable HTTPS proxying.
	// The reason why we allow multiple HTTPS proxies is that with Adam controller
	// we use port different than 443 for HTTPS. With transparent proxy it is therefore
	// necessary to listen on multiple ports for HTTPS traffic (we cannot redirect all
	// HTTPS traffic to a single port because we would loose information about the original
	// destination port). For explicit proxy it does not make much sense to specify
	// multiple endpoints.
	HTTPSPorts []sdnapi.ProxyPort `json:"httpsPorts"`
	// Transparent : enable for transparent proxy (not known to the client).
	Transparent bool `json:"transparent"`
	// LogFile : file to write all log messages into.
	LogFile string `json:"logFile"`
	// PidFile : file to write goproxy process PID.
	PidFile string `json:"pidFile"`
	// Verbose : enable to have all proxied requests logged.
	Verbose bool `json:"verbose"`
	// CertPEM : Proxy certificate of the certificate authority in the PEM format.
	// Proxy will use CA cert to sign certificate that it generates for itself.
	// EVE should be configured to trust CA certificate.
	// Not needed if proxy is listening only on HTTP port and just forwarding
	// all flows (i.e. not terminating TLS).
	CACertPEM string `json:"caCertPEM"`
	// CAKeyPEM : Proxy key of the certificate authority in the PEM format.
	// Proxy will use CA cert to sign certificate that it generates for itself.
	// EVE should be configured to trust CA certificate.
	// Not needed if proxy is listening only on HTTP port and just forwarding
	// all flows (i.e. not terminating TLS).
	CAKeyPEM string `json:"caKeyPEM"`
	// ProxyRules : a set of rules that decides what to do with proxied traffic.
	// By default (no rules defined), proxy will just forward all the flows.
	ProxyRules []sdnapi.ProxyRule `json:"proxyRules"`
	// Users : define for username/password authentication, leave empty otherwise.
	Users []sdnapi.UserCredentials `json:"users"`
}
