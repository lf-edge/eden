package device

import "github.com/satori/go.uuid"

//Ctx is base struct for device
type Ctx struct {
	id               *uuid.UUID
	baseOSConfigs    []string
	networkInstances []string
}

//CreateWithBaseConfig generate base config for device with id and associate with cloudCtx
func CreateWithBaseConfig(id *uuid.UUID) *Ctx {
	return &Ctx{
		id: id,
	}
}

//GetID return id of device
func (cfg *Ctx) GetID() *uuid.UUID { return cfg.id }

//GetBaseOSConfigs return baseOSConfigs of device
func (cfg *Ctx) GetBaseOSConfigs() []string { return cfg.baseOSConfigs }

//GetNetworkInstances return networkInstances of device
func (cfg *Ctx) GetNetworkInstances() []string { return cfg.networkInstances }
