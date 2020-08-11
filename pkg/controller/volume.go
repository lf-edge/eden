package controller

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
)

//GetVolume return Volume config from cloud by ID
func (cloud *CloudCtx) GetVolume(id string) (volume *config.Volume, err error) {
	for _, volume := range cloud.volumes {
		if volume.Uuid == id {
			return volume, nil
		}
	}
	return nil, fmt.Errorf("not found Volume with ID: %s", id)
}

//AddVolume add Volume config to cloud
func (cloud *CloudCtx) AddVolume(volume *config.Volume) error {
	for _, vol := range cloud.volumes {
		if vol.Uuid == volume.GetUuid() {
			return fmt.Errorf("volume already exists with ID: %s", volume.GetUuid())
		}
	}
	cloud.volumes = append(cloud.volumes, volume)
	return nil
}

//RemoveVolume remove Volume config to cloud
func (cloud *CloudCtx) RemoveVolume(id string) error {
	for ind, vol := range cloud.volumes {
		if vol.Uuid == id {
			utils.DelEleInSlice(&cloud.volumes, ind)
			return nil
		}
	}
	return fmt.Errorf("not found Volume with ID: %s", id)
}

//ListVolume return Volume configs from cloud
func (cloud *CloudCtx) ListVolume() []*config.Volume {
	return cloud.volumes
}
