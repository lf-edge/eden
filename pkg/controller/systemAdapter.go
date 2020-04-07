package controller

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

//GetSystemAdapter return SystemAdapter config from cloud by ID
func (cloud *CloudCtx) GetSystemAdapter(id string) (systemAdapter *config.SystemAdapter, err error) {
	systemAdapter, ok := cloud.systemAdapters[id]
	if !ok {
		return nil, fmt.Errorf("not found SystemAdapter with ID: %s", id)
	}
	return systemAdapter, nil
}

//AddSystemAdapter add SystemAdapter config to cloud
func (cloud *CloudCtx) AddSystemAdapter(id string, systemAdapter *config.SystemAdapter) error {
	if cloud.systemAdapters == nil {
		cloud.systemAdapters = make(map[string]*config.SystemAdapter)
	}
	cloud.systemAdapters[id] = systemAdapter
	return nil
}

//RemoveSystemAdapter remove SystemAdapter config to cloud
func (cloud *CloudCtx) RemoveSystemAdapter(id string) error {
	_, err := cloud.GetSystemAdapter(id)
	if err != nil {
		return err
	}
	delete(cloud.systemAdapters, id)
	return nil
}
