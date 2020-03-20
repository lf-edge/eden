package cloud

import (
	"errors"
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

type CloudCtx struct {
	datastores []*config.DatastoreConfig
	images     []*config.Image
	drives     map[uuid.UUID]*config.Drive
	baseOS     []*config.BaseOSConfig
}

func (cfg *CloudCtx) GetDataStore(Id string) (ds *config.DatastoreConfig, err error) {
	for _, dataStore := range cfg.datastores {
		if dataStore.Id == Id {
			return dataStore, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found with ID: %s", Id))
}

func (cfg *CloudCtx) GetImage(Id string) (image *config.Image, err error) {
	for _, image := range cfg.images {
		if image.Uuidandversion.Uuid == Id {
			return image, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found with ID: %s", Id))
}
func (cfg *CloudCtx) GetBaseOSConfig(Id string) (baseOSConfig *config.BaseOSConfig, err error) {
	for _, baseOS := range cfg.baseOS {
		if baseOS.Uuidandversion.Uuid == Id {
			return baseOS, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found with ID: %s", Id))
}

func (cfg *CloudCtx) AddImage(imageConfig *config.Image) error {
	for _, image := range cfg.images {
		if image.Uuidandversion.Uuid == imageConfig.Uuidandversion.GetUuid() {
			return errors.New(fmt.Sprintf("already exists with ID: %s", imageConfig.Uuidandversion.GetUuid()))
		}
	}
	_, err := cfg.GetDataStore(imageConfig.DsId)
	if err != nil {
		return err
	}
	cfg.images = append(cfg.images, imageConfig)
	return nil
}

func (cfg *CloudCtx) AddDatastore(datastoreConfig *config.DatastoreConfig) error {
	for _, dataStore := range cfg.datastores {
		if dataStore.Id == datastoreConfig.Id {
			return errors.New(fmt.Sprintf("already exists with ID: %s", datastoreConfig.Id))
		}
	}
	cfg.datastores = append(cfg.datastores, datastoreConfig)
	return nil
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
