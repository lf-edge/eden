package controller

import (
	"encoding/json"
	"errors"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

//ConfigSync set config for devID
func (cloud *CloudCtx) ConfigSync(dev *device.Ctx) (err error) {
	devConfig, err := cloud.GetConfigBytes(dev)
	if err != nil {
		return err
	}
	return cloud.ConfigSet(dev.GetID(), devConfig)
}

//GetDeviceUUID return device object by devUUID
func (cloud *CloudCtx) GetDeviceUUID(devUUID uuid.UUID) (dev *device.Ctx, err error) {
	for _, el := range cloud.devices {
		if devUUID.String() == el.GetID().String() {
			return el, nil
		}
	}
	return nil, errors.New("no device found")
}

//GetDeviceFirst return first device object
func (cloud *CloudCtx) GetDeviceFirst() (dev *device.Ctx, err error) {
	if len(cloud.devices) == 0 {
		return nil, errors.New("no device found")
	}
	return cloud.devices[0], nil
}

//AddDevice add device with specified devUUID
func (cloud *CloudCtx) AddDevice(devUUID uuid.UUID) (dev *device.Ctx, err error) {
	for _, el := range cloud.devices {
		if el.GetID().String() == devUUID.String() {
			return nil, errors.New("already exists")
		}
	}
	dev = device.CreateWithBaseConfig(devUUID)
	cloud.devices = append(cloud.devices, dev)
	return
}

//ApplyDevModel apply networks, adapters and physicalIOs from DevModel to device
func (cloud *CloudCtx) ApplyDevModel(dev *device.Ctx, devModel *DevModel) error {
	var err error
	dev.SetAdaptersForSwitch(devModel.adapterForSwitches)
	var adapters []string
	for _, el := range devModel.adapters {
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		err = cloud.AddSystemAdapter(id.String(), el)
		if err != nil {
			return err
		}
		adapters = append(adapters, id.String())
	}
	dev.SetSystemAdaptersConfig(adapters)
	var networks []string
	for _, el := range devModel.networks {
		err = cloud.AddNetworkConfig(el)
		if err != nil {
			return err
		}
		networks = append(networks, el.Id)
	}
	dev.SetNetworkConfig(networks)
	var physicalIOs []string
	for _, el := range devModel.physicalIOs {
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		err = cloud.AddPhysicalIO(id.String(), el)
		if err != nil {
			return err
		}
		physicalIOs = append(physicalIOs, id.String())
	}
	dev.SetPhysicalIOConfig(physicalIOs)
	dev.SetDevModel(string(devModel.devModelType))
	return nil
}

func checkIfDatastoresContains(dsID string, ds []*config.DatastoreConfig) bool {
	for _, el := range ds {
		if el.Id == dsID {
			return true
		}
	}
	return false
}

func (cloud *CloudCtx) checkDrive(drive *config.Drive, dataStores []*config.DatastoreConfig) (result []*config.DatastoreConfig, err error) {
	if drive.Image == nil {
		return nil, errors.New("empty Image in Drive")
	}
	dataStore, err := cloud.GetDataStore(drive.Image.DsId)
	if err != nil {
		return nil, err
	}
	if !checkIfDatastoresContains(dataStore.Id, dataStores) {
		return append(dataStores, dataStore), nil
	}
	return dataStores, nil
}

//GetConfigBytes generate json representation of device config
func (cloud *CloudCtx) GetConfigBytes(dev *device.Ctx) ([]byte, error) {
	var baseOS []*config.BaseOSConfig
	var dataStores []*config.DatastoreConfig
	for _, baseOSConfigID := range dev.GetBaseOSConfigs() {
		baseOSConfig, err := cloud.GetBaseOSConfig(baseOSConfigID)
		if err != nil {
			return nil, err
		}
		//check drives from baseOSConfigs
		for _, drive := range baseOSConfig.Drives {
			dataStores, err = cloud.checkDrive(drive, dataStores)
			if err != nil {
				return nil, err
			}
		}
		baseOS = append(baseOS, baseOSConfig)
	}
	var applicationInstances []*config.AppInstanceConfig
	for _, applicationInstanceConfigID := range dev.GetApplicationInstances() {
		applicationInstance, err := cloud.GetApplicationInstanceConfig(applicationInstanceConfigID)
		if err != nil {
			return nil, err
		}
		//check drives from apps
		for _, drive := range applicationInstance.Drives {
			dataStores, err = cloud.checkDrive(drive, dataStores)
			if err != nil {
				return nil, err
			}
		}
		//check network instances from apps
		for _, networkInstanceConfig := range applicationInstance.Interfaces {
			if networkInstanceConfig != nil && networkInstanceConfig.NetworkId != "" {
				networkInstanceConfigArray := dev.GetNetworkInstances()
				if _, ok := utils.FindEleInSlice(networkInstanceConfigArray, networkInstanceConfig.NetworkId); !ok {
					networkInstanceConfigArray = append(networkInstanceConfigArray, networkInstanceConfig.NetworkId)
					dev.SetNetworkInstanceConfig(networkInstanceConfigArray)
				}
			}
		}
		applicationInstances = append(applicationInstances, applicationInstance)
	}
	var networkInstanceConfigs []*config.NetworkInstanceConfig
	for _, networkInstanceConfigID := range dev.GetNetworkInstances() {
		networkInstanceConfig, err := cloud.GetNetworkInstanceConfig(networkInstanceConfigID)
		if err != nil {
			return nil, err
		}
		networkInstanceConfigs = append(networkInstanceConfigs, networkInstanceConfig)
	}
	var physicalIOs []*config.PhysicalIO
	for _, physicalIOID := range dev.GetPhysicalIOs() {
		physicalIOConfig, err := cloud.GetPhysicalIO(physicalIOID)
		if err != nil {
			return nil, err
		}
		physicalIOs = append(physicalIOs, physicalIOConfig)
	}
	var networkConfigs []*config.NetworkConfig
	for _, networkConfigID := range dev.GetNetworks() {
		networkConfig, err := cloud.GetNetworkConfig(networkConfigID)
		if err != nil {
			return nil, err
		}
		networkConfigs = append(networkConfigs, networkConfig)
	}
	var systemAdapterConfigs []*config.SystemAdapter
	for _, systemAdapterConfigID := range dev.GetSystemAdapters() {
		systemAdapterConfig, err := cloud.GetSystemAdapter(systemAdapterConfigID)
		if err != nil {
			return nil, err
		}
		systemAdapterConfigs = append(systemAdapterConfigs, systemAdapterConfig)
	}
	var configItems []*config.ConfigItem
	for _, sshKey := range dev.GetSSHKeys() {
		configItems = append(configItems, &config.ConfigItem{
			Key:   "debug.enable.ssh",
			Value: sshKey,
		})
	}
	if dev.GetVncAccess() {
		configItems = append(configItems, &config.ConfigItem{
			Key:   "app.allow.vnc",
			Value: "true",
		})
	}
	if dev.GetControllerLogLevel() != "" {
		configItems = append(configItems, &config.ConfigItem{
			Key:   "debug.default.remote.loglevel",
			Value: dev.GetControllerLogLevel(),
		})
	}
	devConfig := &config.EdgeDevConfig{
		Id: &config.UUIDandVersion{
			Uuid:    dev.GetID().String(),
			Version: "4",
		},
		Apps:              applicationInstances,
		Networks:          networkConfigs,
		Datastores:        dataStores,
		LispInfo:          nil,
		Base:              baseOS,
		Reboot:            nil,
		Backup:            nil,
		ConfigItems:       configItems,
		SystemAdapterList: systemAdapterConfigs,
		DeviceIoList:      physicalIOs,
		Manufacturer:      "",
		ProductName:       "",
		NetworkInstances:  networkInstanceConfigs,
		Enterprise:        "",
		Name:              "",
	}
	return json.Marshal(devConfig)
}
