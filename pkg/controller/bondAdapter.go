package controller

import (
	"fmt"

	"github.com/lf-edge/eve-api/go/config"
)

// GetBondAdapter return BondAdapter config from cloud by ID
func (cloud *CloudCtx) GetBondAdapter(id string) (bondAdapter *config.BondAdapter, err error) {
	bondAdapter, ok := cloud.bondAdapters[id]
	if !ok {
		return nil, fmt.Errorf("not found BondAdapter with ID: %s", id)
	}
	return bondAdapter, nil
}

// AddBondAdapter add BondAdapter config to cloud
func (cloud *CloudCtx) AddBondAdapter(id string, bondAdapter *config.BondAdapter) error {
	if cloud.bondAdapters == nil {
		cloud.bondAdapters = make(map[string]*config.BondAdapter)
	}
	cloud.bondAdapters[id] = bondAdapter
	return nil
}

// RemoveBondAdapter remove BondAdapter config from cloud
func (cloud *CloudCtx) RemoveBondAdapter(id string) error {
	_, err := cloud.GetBondAdapter(id)
	if err != nil {
		return err
	}
	delete(cloud.bondAdapters, id)
	return nil
}
