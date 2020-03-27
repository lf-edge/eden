package device

//SetBaseOSConfig set BaseOSConfig by configIDs from cloud
func (cfg *Ctx) SetBaseOSConfig(configIDs []string) *Ctx {
	cfg.baseOSConfigs = configIDs
	return cfg
}
