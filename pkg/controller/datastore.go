package controller

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/config"
)

// GetDataStore return DataStore config from cloud by ID
func (cloud *CloudCtx) GetDataStore(id string) (ds *config.DatastoreConfig, err error) {
	for _, dataStore := range cloud.datastores {
		if dataStore.Id == id {
			return dataStore, nil
		}
	}
	return nil, fmt.Errorf("not found DatastoreConfig with ID: %s", id)
}

// AddDataStore add DataStore config to cloud
func (cloud *CloudCtx) AddDataStore(dataStoreConfig *config.DatastoreConfig) error {
	for _, dataStore := range cloud.datastores {
		if dataStore.Id == dataStoreConfig.Id {
			return fmt.Errorf("datastore already exists with ID: %s", dataStoreConfig.Id)
		}
	}
	cloud.datastores = append(cloud.datastores, dataStoreConfig)
	return nil
}

// RemoveDataStore remove DataStore config from cloud
func (cloud *CloudCtx) RemoveDataStore(id string) error {
	for ind, datastore := range cloud.datastores {
		if datastore.Id == id {
			utils.DelEleInSlice(&cloud.datastores, ind)
			return nil
		}
	}
	return fmt.Errorf("not found DataStore with ID: %s", id)
}

// ListDataStore return DataStore configs from cloud
func (cloud *CloudCtx) ListDataStore() []*config.DatastoreConfig {
	return cloud.datastores
}
