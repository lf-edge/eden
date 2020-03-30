package controller

import (
	"encoding/json"
	"errors"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

//GetDeviceUUID return device object by devUUID
func (adam *Ctx) GetDeviceUUID(devUUID *uuid.UUID) (dID *device.Ctx, err error) {
	for _, el := range adam.Devices {
		if devUUID.String() == el.GetID().String() {
			return el, nil
		}
	}
	return nil, errors.New("no device found")
}

//GetDeviceFirst return first device object
func (adam *Ctx) GetDeviceFirst() (dID *device.Ctx, err error) {
	if len(adam.Devices) == 0 {
		return nil, errors.New("no device found")
	}
	return adam.Devices[0], nil
}

//AddDevice add device with specified devUUID
func (adam *Ctx) AddDevice(devUUID *uuid.UUID) error {
	for _, el := range adam.Devices {
		if el.GetID().String() == devUUID.String() {
			return errors.New("already exists")
		}
	}
	adam.Devices = append(adam.Devices, device.CreateWithBaseConfig(devUUID))
	return nil
}

func checkIfDatastoresContains(id string, ds []*config.DatastoreConfig) bool {
	for _, el := range ds {
		if el.Id == id {
			return true
		}
	}
	return false
}

//GetConfigBytes generate json representation of device config
func (adam *Ctx) GetConfigBytes(devUUID *uuid.UUID) ([]byte, error) {
	dev, err := adam.GetDeviceUUID(devUUID)
	if err != nil {
		return nil, err
	}
	var BaseOS []*config.BaseOSConfig
	var DataStores []*config.DatastoreConfig
	for _, baseOSConfigID := range dev.GetBaseOSConfigs() {
		baseOSConfig, err := adam.GetBaseOSConfig(baseOSConfigID)
		if err != nil {
			return nil, err
		}
		for _, drive := range baseOSConfig.Drives {
			if drive.Image == nil {
				return nil, errors.New("empty Image in Drive")
			}
			dataStore, err := adam.GetDataStore(drive.Image.DsId)
			if err != nil {
				return nil, err
			}
			if !checkIfDatastoresContains(dataStore.Id, DataStores) {
				DataStores = append(DataStores, dataStore)
			}
		}
		BaseOS = append(BaseOS, baseOSConfig)
	}
	var NetworkInstanceConfigs []*config.NetworkInstanceConfig
	for _, networkInstanceConfigID := range dev.GetNetworkInstances() {
		networkInstanceConfig, err := adam.GetNetworkInstanceConfig(networkInstanceConfigID)
		if err != nil {
			return nil, err
		}
		NetworkInstanceConfigs = append(NetworkInstanceConfigs, networkInstanceConfig)
	}
	devConfig := &config.EdgeDevConfig{
		Id: &config.UUIDandVersion{
			Uuid:    dev.GetID().String(),
			Version: "4",
		},
		Apps:              nil,
		Networks:          nil,
		Datastores:        DataStores,
		LispInfo:          nil,
		Base:              BaseOS,
		Reboot:            nil,
		Backup:            nil,
		ConfigItems:       nil,
		SystemAdapterList: nil,
		DeviceIoList:      nil,
		Manufacturer:      "",
		ProductName:       "",
		NetworkInstances:  NetworkInstanceConfigs,
		Enterprise:        "",
		Name:              "",
	}
	return json.Marshal(devConfig)
}
