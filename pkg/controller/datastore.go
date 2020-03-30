package controller

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

//GetDataStore return DataStore config from cloud by ID
func (cloud *CloudCtx) GetDataStore(ID string) (ds *config.DatastoreConfig, err error) {
	for _, dataStore := range cloud.datastores {
		if dataStore.Id == ID {
			return dataStore, nil
		}
	}
	return nil, fmt.Errorf("not found with ID: %s", ID)
}

//AddDatastore add DataStore config to cloud
func (cloud *CloudCtx) AddDatastore(datastoreConfig *config.DatastoreConfig) error {
	for _, dataStore := range cloud.datastores {
		if dataStore.Id == datastoreConfig.Id {
			return fmt.Errorf("already exists with ID: %s", datastoreConfig.Id)
		}
	}
	cloud.datastores = append(cloud.datastores, datastoreConfig)
	return nil
}
