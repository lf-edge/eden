package controller

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
)

func (cloud *CloudCtx) getNetworkInstanceInd(id string) (networkInstanceConfigInd int, err error) {
	for ind, el := range cloud.networkInstances {
		if el != nil && el.Uuidandversion != nil && el.Uuidandversion.Uuid == id {
			return ind, nil
		}
	}
	return -1, fmt.Errorf("not found networkInstanceConfig with ID: %s", id)
}

//GetNetworkInstanceConfig return NetworkInstance config from cloud by ID
func (cloud *CloudCtx) GetNetworkInstanceConfig(id string) (networkInstanceConfig *config.NetworkInstanceConfig, err error) {
	networkInstanceConfigInd, err := cloud.getNetworkInstanceInd(id)
	if err != nil {
		return nil, err
	}
	return cloud.networkInstances[networkInstanceConfigInd], nil
}

//AddNetworkInstanceConfig add NetworkInstance config to cloud
func (cloud *CloudCtx) AddNetworkInstanceConfig(networkInstanceConfig *config.NetworkInstanceConfig) error {
	cloud.networkInstances = append(cloud.networkInstances, networkInstanceConfig)
	return nil
}

//RemoveNetworkInstanceConfig remove NetworkInstance config to cloud
func (cloud *CloudCtx) RemoveNetworkInstanceConfig(id string) error {
	networkInstanceConfigInd, err := cloud.getNetworkInstanceInd(id)
	if err != nil {
		return err
	}
	utils.DelEleInSlice(cloud.networkInstances, networkInstanceConfigInd)
	return nil
}
