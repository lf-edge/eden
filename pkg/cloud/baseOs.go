package cloud

import (
	"errors"
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

func (cfg *CloudCtx) GetBaseOSConfig(Id string) (baseOSConfig *config.BaseOSConfig, err error) {
	for _, baseOS := range cfg.baseOS {
		if baseOS.Uuidandversion.Uuid == Id {
			return baseOS, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found with ID: %s", Id))
}

func (cfg *CloudCtx) AddBaseOsConfig(baseOSConfig *config.BaseOSConfig) error {
	for _, baseConfig := range cfg.baseOS {
		if baseConfig.Uuidandversion.Uuid == baseOSConfig.Uuidandversion.GetUuid() {
			return errors.New(fmt.Sprintf("already exists with ID: %s", baseOSConfig.Uuidandversion.GetUuid()))
		}
	}
	cfg.baseOS = append(cfg.baseOS, baseOSConfig)
	return nil
}
