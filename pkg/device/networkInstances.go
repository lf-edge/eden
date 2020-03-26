package device

func (cfg *DevCtx) SetNetworkInstanceConfig(configIDs []string) *DevCtx {
	cfg.networkInstances = configIDs
	return cfg
}
