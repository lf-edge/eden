package controller

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

//GetBaseOSConfig return baseOS config from cloud by ID
func (cloud *CloudCtx) GetBaseOSConfig(ID string) (baseOSConfig *config.BaseOSConfig, err error) {
	for _, baseOS := range cloud.baseOS {
		if baseOS.Uuidandversion.Uuid == ID {
			return baseOS, nil
		}
	}
	return nil, fmt.Errorf("not found with ID: %s", ID)
}

//AddBaseOsConfig add baseOS config to cloud
func (cloud *CloudCtx) AddBaseOsConfig(baseOSConfig *config.BaseOSConfig) error {
	for _, baseConfig := range cloud.baseOS {
		if baseConfig.Uuidandversion.Uuid == baseOSConfig.Uuidandversion.GetUuid() {
			return fmt.Errorf("already exists with ID: %s", baseOSConfig.Uuidandversion.GetUuid())
		}
	}
	cloud.baseOS = append(cloud.baseOS, baseOSConfig)
	return nil
}
