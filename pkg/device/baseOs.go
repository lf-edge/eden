package device

func (cfg *DevCtx) SetBaseOSConfig(configIDs []string) *DevCtx {
	cfg.baseOSConfigs = configIDs
	return cfg
}
