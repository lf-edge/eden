package openevec

// OpenEVEC base type for all actions
type OpenEVEC struct {
	cfg *EdenSetupArgs
}

// CreateOpenEVEC returns OpenEVEC instance
func CreateOpenEVEC(cfg *EdenSetupArgs) *OpenEVEC {
	return &OpenEVEC{cfg: cfg}
}
