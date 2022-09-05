package config

import (
	sdnapi "github.com/lf-edge/eden/sdn/api"
)

// ProxyConfig : proxy configuration formatted with JSON and passed to goproxy
// using the "-c" command line argument.
type ProxyConfig struct {
	// ListenIP : IP address to listen on.
	// Leave empty to listen on all available interfaces instead of just
	// the interface with the given host address.
	ListenIP string `json:"listenIP"`
	// ListenPort : Port on which the proxy listens for both HTTP and HTTPS.
	ListenPort uint16 `json:"listenPort"`
	// LogFile : file to write all log messages into.
	LogFile string `json:"logFile"`
	// PidFile : file to write goproxy process PID.
	PidFile string `json:"pidFile"`
	// Verbose : enable to have all proxied requests logged.
	Verbose bool `json:"verbose"`
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
	ProxyRules []sdnapi.ProxyRule `json:"proxyRules"`
	// Users : define for username/password authentication, leave empty otherwise.
	Users []sdnapi.UserCredentials `json:"users"`
}
