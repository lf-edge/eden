package controller

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
)

func (cloud *CloudCtx) getApplicationInstanceInd(id string) (applicationInstanceConfigInd int, err error) {
	for ind, el := range cloud.applicationInstances {
		if el != nil && el.Uuidandversion != nil && el.Uuidandversion.Uuid == id {
			return ind, nil
		}
	}
	return -1, fmt.Errorf("not found applicationInstance with ID: %s", id)
}

//GetApplicationInstanceConfig return AppInstanceConfig config from cloud by ID
func (cloud *CloudCtx) GetApplicationInstanceConfig(id string) (applicationInstanceConfig *config.AppInstanceConfig, err error) {
	applicationInstanceConfigInd, err := cloud.getApplicationInstanceInd(id)
	if err != nil {
		return nil, err
	}
	return cloud.applicationInstances[applicationInstanceConfigInd], nil
}

//AddApplicationInstanceConfig add AppInstanceConfig config to cloud
func (cloud *CloudCtx) AddApplicationInstanceConfig(applicationInstanceConfig *config.AppInstanceConfig) error {
	cloud.applicationInstances = append(cloud.applicationInstances, applicationInstanceConfig)
	return nil
}

//RemoveApplicationInstanceConfig remove AppInstanceConfig config to cloud
func (cloud *CloudCtx) RemoveApplicationInstanceConfig(id string) error {
	applicationInstanceConfigInd, err := cloud.getApplicationInstanceInd(id)
	if err != nil {
		return err
	}
	utils.DelEleInSlice(&cloud.applicationInstances, applicationInstanceConfigInd)
	return nil
}

//ListApplicationInstanceConfig return ApplicationInstance configs from cloud
func (cloud *CloudCtx) ListApplicationInstanceConfig() []*config.AppInstanceConfig {
	return cloud.applicationInstances
}
