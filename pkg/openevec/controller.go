package openevec

import "reflect"

// OpenEVEC base type for all actions
type OpenEVEC struct {
	cfg *EdenSetupArgs
}

// CreateOpenEVEC returns OpenEVEC instance
func CreateOpenEVEC(cfg *EdenSetupArgs) *OpenEVEC {
	resolvePath(cfg.Eden.Root, reflect.ValueOf(cfg).Elem())
	return &OpenEVEC{cfg: cfg}
}
