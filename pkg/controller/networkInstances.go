package controller

import (
	"errors"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
)

func (cfg *Ctx) getNetworkInstanceInd(id string) (networkInstanceConfigInd int, err error) {
	for ind, el := range cfg.networkInstances {
		if el != nil && el.Uuidandversion != nil && el.Uuidandversion.Uuid == id {
			return ind, nil
		}
	}
	return -1, errors.New("not found")
}

//GetNetworkInstanceConfig return NetworkInstance config from cloud by ID
func (cfg *Ctx) GetNetworkInstanceConfig(id string) (networkInstanceConfig *config.NetworkInstanceConfig, err error) {
	networkInstanceConfigInd, err := cfg.getNetworkInstanceInd(id)
	if err != nil {
		return nil, err
	}
	return cfg.networkInstances[networkInstanceConfigInd], nil
}

//AddNetworkInstanceConfig add NetworkInstance config to cloud
func (cfg *Ctx) AddNetworkInstanceConfig(networkInstanceConfig *config.NetworkInstanceConfig) error {
	cfg.networkInstances = append(cfg.networkInstances, networkInstanceConfig)
	return nil
}

//RemoveNetworkInstanceConfig remove NetworkInstance config to cloud
func (cfg *Ctx) RemoveNetworkInstanceConfig(id string) error {
	networkInstanceConfigInd, err := cfg.getNetworkInstanceInd(id)
	if err != nil {
		return err
	}
	utils.DelEleInSlice(cfg.networkInstances, networkInstanceConfigInd)
	return nil
}
