package cloud

import (
	"errors"
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

func (cfg *CloudCtx) GetDataStore(Id string) (ds *config.DatastoreConfig, err error) {
	for _, dataStore := range cfg.datastores {
		if dataStore.Id == Id {
			return dataStore, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found with ID: %s", Id))
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
