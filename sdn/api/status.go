package api

import (
	 "github.com/lf-edge/eve/libs/depgraph"
)

// Status : Current status of Eden SDN as reported by the SDN agent.
type Status struct {
	// MgmtIPAddress : IP address on which SDN agent is available from the host.
	MgmtIPAddress string `json:"mgmtIPAddress"`
	// ConfigErrors : a set of current configuration errors. Normally this should be empty.
	ConfigErrors []ConfigError `json:"configErrors,omitempty"`
	// TODO: more fields...
}

// ConfigError : error returned if the SDN agent failed to configure some configuration item.
type ConfigError struct {
	// ItemRef : reference to a configuration item.
	ItemRef depgraph.ItemRef
	// ErrMsg : error message
	ErrMsg  string
}
