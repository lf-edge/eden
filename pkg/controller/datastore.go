package controller

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

//GetDataStore return DataStore config from cloud by ID
func (cfg *Ctx) GetDataStore(ID string) (ds *config.DatastoreConfig, err error) {
	for _, dataStore := range cfg.datastores {
		if dataStore.Id == ID {
			return dataStore, nil
		}
	}
	return nil, fmt.Errorf("not found with ID: %s", ID)
}

//AddDatastore add DataStore config to cloud
func (cfg *Ctx) AddDatastore(datastoreConfig *config.DatastoreConfig) error {
	for _, dataStore := range cfg.datastores {
		if dataStore.Id == datastoreConfig.Id {
			return fmt.Errorf("already exists with ID: %s", datastoreConfig.Id)
		}
	}
	cfg.datastores = append(cfg.datastores, datastoreConfig)
	return nil
}
