package controller

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
)

//GetPhysicalIO return PhysicalIO config from cloud by ID
func (cloud *CloudCtx) GetPhysicalIO(id string) (physicalIO *config.PhysicalIO, err error) {
	physicalIO, ok := cloud.physicalIOs[id]
	if !ok {
		return nil, fmt.Errorf("not found PhysicalIO with ID: %s", id)
	}
	return physicalIO, nil
}

//AddPhysicalIO add PhysicalIO config to cloud
func (cloud *CloudCtx) AddPhysicalIO(id string, physicalIO *config.PhysicalIO) error {
	if cloud.physicalIOs == nil {
		cloud.physicalIOs = make(map[string]*config.PhysicalIO)
	}
	cloud.physicalIOs[id] = physicalIO
	return nil
}

//RemovePhysicalIO remove PhysicalIO config to cloud
func (cloud *CloudCtx) RemovePhysicalIO(id string) error {
	_, err := cloud.GetPhysicalIO(id)
	if err != nil {
		return err
	}
	delete(cloud.physicalIOs, id)
	return nil
}
