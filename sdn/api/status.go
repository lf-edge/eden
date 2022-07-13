package api

import (
	"github.com/lf-edge/eve/libs/depgraph"
)

// SDNStatus : Current status of Eden SDN as reported by the SDN agent.
type SDNStatus struct {
	// MgmtIPs : IP addresses on which SDN agent is available from the host.
	MgmtIPs []string `json:"mgmtIPs"`
	// ConfigErrors : a set of current configuration errors. Normally this should be empty.
	ConfigErrors []ConfigError `json:"configErrors,omitempty"`
	// TODO: more fields...
}

// ConfigError : error returned if the SDN agent failed to configure some configuration item.
type ConfigError struct {
	// ItemRef : reference to a configuration item.
	ItemRef depgraph.ItemRef
	// ErrMsg : error message
	ErrMsg string
}
