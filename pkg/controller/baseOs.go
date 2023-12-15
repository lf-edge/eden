package controller

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/config"
)

// GetBaseOSConfig return baseOSConfigs config from cloud by ID
func (cloud *CloudCtx) GetBaseOSConfig(id string) (baseOSConfig *config.BaseOSConfig, err error) {
	for _, baseOS := range cloud.baseOSConfigs {
		if baseOS.Uuidandversion.Uuid == id {
			return baseOS, nil
		}
	}
	return nil, fmt.Errorf("not found BaseOSConfig with ID: %s", id)
}

// ListBaseOSConfig return baseOSConfigs configs from cloud
func (cloud *CloudCtx) ListBaseOSConfig() []*config.BaseOSConfig {
	return cloud.baseOSConfigs
}

// AddBaseOsConfig add baseOSConfigs config to cloud
func (cloud *CloudCtx) AddBaseOsConfig(baseOSConfig *config.BaseOSConfig) error {
	for _, baseConfig := range cloud.baseOSConfigs {
		if baseConfig.Uuidandversion.Uuid == baseOSConfig.Uuidandversion.GetUuid() {
			return fmt.Errorf("baseOSConfigs already exists with ID: %s", baseOSConfig.Uuidandversion.GetUuid())
		}
	}
	cloud.baseOSConfigs = append(cloud.baseOSConfigs, baseOSConfig)
	return nil
}

// RemoveBaseOsConfig remove BaseOsConfig from cloud
func (cloud *CloudCtx) RemoveBaseOsConfig(id string) error {
	for ind, baseOS := range cloud.baseOSConfigs {
		if baseOS.Uuidandversion.Uuid == id {
			utils.DelEleInSlice(&cloud.baseOSConfigs, ind)
			return nil
		}
	}
	return fmt.Errorf("not found baseOSConfigs with ID: %s", id)
}
