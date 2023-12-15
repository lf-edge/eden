package controller

import (
	"fmt"

	"github.com/lf-edge/eve-api/go/config"
)

// GetVlanAdapter return VlanAdapter config from cloud by ID
func (cloud *CloudCtx) GetVlanAdapter(id string) (vlanAdapter *config.VlanAdapter, err error) {
	vlanAdapter, ok := cloud.vlanAdapters[id]
	if !ok {
		return nil, fmt.Errorf("not found VlanAdapter with ID: %s", id)
	}
	return vlanAdapter, nil
}

// AddVlanAdapter add VlanAdapter config to cloud
func (cloud *CloudCtx) AddVlanAdapter(id string, vlanAdapter *config.VlanAdapter) error {
	if cloud.vlanAdapters == nil {
		cloud.vlanAdapters = make(map[string]*config.VlanAdapter)
	}
	cloud.vlanAdapters[id] = vlanAdapter
	return nil
}

// RemoveVlanAdapter remove VlanAdapter config from cloud
func (cloud *CloudCtx) RemoveVlanAdapter(id string) error {
	_, err := cloud.GetVlanAdapter(id)
	if err != nil {
		return err
	}
	delete(cloud.vlanAdapters, id)
	return nil
}
