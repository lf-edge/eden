package device

//SetNetworkInstanceConfig set NetworkInstanceConfig by configIDs from cloud
func (cfg *Ctx) SetNetworkInstanceConfig(configIDs []string) *Ctx {
	cfg.networkInstances = configIDs
	return cfg
}
