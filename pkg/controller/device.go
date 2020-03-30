package controller

import (
	"encoding/json"
	"errors"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

//ConfigSync set config for devID
func (cloud *CloudCtx) ConfigSync(devUUID *uuid.UUID) (err error) {
	devConfig, err := cloud.GetConfigBytes(devUUID)
	if err != nil {
		return err
	}
	return cloud.ConfigSet(devUUID, devConfig)
}

//GetDeviceUUID return device object by devUUID
func (cloud *CloudCtx) GetDeviceUUID(devUUID *uuid.UUID) (dID *device.Ctx, err error) {
	for _, el := range cloud.devices {
		if devUUID.String() == el.GetID().String() {
			return el, nil
		}
	}
	return nil, errors.New("no device found")
}

//GetDeviceFirst return first device object
func (cloud *CloudCtx) GetDeviceFirst() (devUUID *device.Ctx, err error) {
	if len(cloud.devices) == 0 {
		return nil, errors.New("no device found")
	}
	return cloud.devices[0], nil
}

//AddDevice add device with specified devUUID
func (cloud *CloudCtx) AddDevice(devUUID *uuid.UUID) error {
	for _, el := range cloud.devices {
		if el.GetID().String() == devUUID.String() {
			return errors.New("already exists")
		}
	}
	cloud.devices = append(cloud.devices, device.CreateWithBaseConfig(devUUID))
	return nil
}

func checkIfDatastoresContains(devUUID string, ds []*config.DatastoreConfig) bool {
	for _, el := range ds {
		if el.Id == devUUID {
			return true
		}
	}
	return false
}

//GetConfigBytes generate json representation of device config
func (cloud *CloudCtx) GetConfigBytes(devUUID *uuid.UUID) ([]byte, error) {
	dev, err := cloud.GetDeviceUUID(devUUID)
	if err != nil {
		return nil, err
	}
	var BaseOS []*config.BaseOSConfig
	var DataStores []*config.DatastoreConfig
	for _, baseOSConfigID := range dev.GetBaseOSConfigs() {
		baseOSConfig, err := cloud.GetBaseOSConfig(baseOSConfigID)
		if err != nil {
			return nil, err
		}
		for _, drive := range baseOSConfig.Drives {
			if drive.Image == nil {
				return nil, errors.New("empty Image in Drive")
			}
			dataStore, err := cloud.GetDataStore(drive.Image.DsId)
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
		networkInstanceConfig, err := cloud.GetNetworkInstanceConfig(networkInstanceConfigID)
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
