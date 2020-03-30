package controller

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

//GetBaseOSConfig return baseOS config from cloud by ID
func (cfg *Ctx) GetBaseOSConfig(ID string) (baseOSConfig *config.BaseOSConfig, err error) {
	for _, baseOS := range cfg.baseOS {
		if baseOS.Uuidandversion.Uuid == ID {
			return baseOS, nil
		}
	}
	return nil, fmt.Errorf("not found with ID: %s", ID)
}

//AddBaseOsConfig add baseOS config to cloud
func (cfg *Ctx) AddBaseOsConfig(baseOSConfig *config.BaseOSConfig) error {
	for _, baseConfig := range cfg.baseOS {
		if baseConfig.Uuidandversion.Uuid == baseOSConfig.Uuidandversion.GetUuid() {
			return fmt.Errorf("already exists with ID: %s", baseOSConfig.Uuidandversion.GetUuid())
		}
	}
	cfg.baseOS = append(cfg.baseOS, baseOSConfig)
	return nil
}
