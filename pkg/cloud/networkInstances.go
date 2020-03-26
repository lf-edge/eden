package cloud

import (
	"errors"
	"github.com/itmo-eve/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
)

func (cfg *CloudCtx) AddNetworkInstanceConfig(networkInstanceConfig *config.NetworkInstanceConfig) error {
	cfg.networkInstances = append(cfg.networkInstances, networkInstanceConfig)
	return nil
}

func (cfg *CloudCtx) getNetworkInstanceInd(id string) (networkInstanceConfigInd int, err error) {
	for ind, el := range cfg.networkInstances {
		if el != nil && el.Uuidandversion != nil && el.Uuidandversion.Uuid == id {
			return ind, nil
		}
	}
	return -1, errors.New("not found")
}

func (cfg *CloudCtx) GetNetworkInstanceConfig(id string) (networkInstanceConfig *config.NetworkInstanceConfig, err error) {
	networkInstanceConfigInd, err := cfg.getNetworkInstanceInd(id)
	if err != nil {
		return nil, err
	}
	return cfg.networkInstances[networkInstanceConfigInd], nil
}

func (cfg *CloudCtx) RemoveNetworkInstanceConfig(id string) error {
	networkInstanceConfigInd, err := cfg.getNetworkInstanceInd(id)
	if err != nil {
		return err
	}
	utils.DelEleInSlice(cfg.networkInstances, networkInstanceConfigInd)
	return nil
}
