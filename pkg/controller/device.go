package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/viper"
	"strconv"
)

//StateUpdate refresh state file
func (cloud *CloudCtx) StateUpdate(dev *device.Ctx) (err error) {
	devConfig, err := cloud.GetConfigBytes(dev, false)
	if err != nil {
		return err
	}
	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	edenConfig, err := utils.DefaultConfigPath()
	if err != nil {
		return err
	}
	loaded, err := utils.LoadConfigFile(edenConfig)
	if err != nil {
		return err
	}
	if loaded {
		if err = utils.GenerateStateFile(edenDir, utils.StateObject{
			EveConfig:  string(devConfig),
			EveDir:     viper.GetString("eve.dist"),
			AdamDir:    cloud.GetDir(),
			EveUUID:    viper.GetString("eve.uuid"),
			DeviceUUID: dev.GetID().String(),
			QEMUConfig: viper.GetString("eve.qemu-config"),
		}); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("cannot load config %s", edenConfig)
}

type idCheckable interface {
	GetId() *config.UUIDandVersion
}
type uuidCheckable interface {
	GetUuidandversion() *config.UUIDandVersion
}

func getID(uuidAndVersion idCheckable) (uuid.UUID, error) {
	if uuidAndVersion == nil {
		return uuid.Nil, fmt.Errorf("nil object")
	}
	if uuidAndVersion.GetId() == nil {
		return uuid.Nil, fmt.Errorf("nil UUIDandVersion")
	}
	if uuidAndVersion.GetId().GetUuid() == "" {
		return uuid.Nil, fmt.Errorf("nil UUIDandVersion")
	}
	return uuid.FromString(uuidAndVersion.GetId().GetUuid())
}

func getUUID(uuidAndVersion uuidCheckable) (uuid.UUID, error) {
	if uuidAndVersion == nil {
		return uuid.Nil, fmt.Errorf("nil object")
	}
	if uuidAndVersion.GetUuidandversion() == nil {
		return uuid.Nil, fmt.Errorf("nil UUIDandVersion")
	}
	if uuidAndVersion.GetUuidandversion().GetUuid() == "" {
		return uuid.Nil, fmt.Errorf("nil UUIDandVersion")
	}
	return uuid.FromString(uuidAndVersion.GetUuidandversion().GetUuid())
}

//ConfigParse load config into cloud
func (cloud *CloudCtx) ConfigParse(config *config.EdgeDevConfig) (device *device.Ctx, err error) {
	devId, err := getID(config)
	if err != nil {
		return nil, fmt.Errorf("problem with uuid field")
	}
	dev, err := cloud.GetDeviceUUID(devId)
	if err != nil { //not found
		dev, err = cloud.AddDevice(devId)
		if err != nil {
			return nil, fmt.Errorf("cloud.AddDevice: %s", err)
		}
	}
	version, _ := strconv.Atoi(config.Id.Version)
	dev.SetConfigVersion(version)
	for _, el := range config.ConfigItems {
		dev.SetConfigItem(el.GetKey(), el.GetValue())
	}
	for _, el := range config.Datastores {
		_ = cloud.AddDataStore(el)
	}

	var baseOSs []string
	for _, el := range config.Base {
		_ = cloud.AddBaseOsConfig(el)
		for _, img := range el.Drives {
			_ = cloud.AddImage(img.Image)
		}
		id, err := getUUID(el)
		if err != nil {
			return nil, err
		}
		baseOSs = append(baseOSs, id.String())
	}
	dev.SetBaseOSConfig(baseOSs)

	var physIOIDs []string
	for _, el := range config.DeviceIoList {
		id, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}
		_ = cloud.AddPhysicalIO(id.String(), el)
		physIOIDs = append(physIOIDs, id.String())
	}
	dev.SetPhysicalIOConfig(physIOIDs)

	var networkInstances []string
	for _, el := range config.NetworkInstances {
		_ = cloud.AddNetworkInstanceConfig(el)
		id, err := getUUID(el)
		if err != nil {
			return nil, err
		}
		networkInstances = append(networkInstances, id.String())
	}
	dev.SetNetworkInstanceConfig(networkInstances)

	var networks []string
	for _, el := range config.Networks {
		_ = cloud.AddNetworkConfig(el)
		networks = append(networks, el.Id)
	}
	dev.SetNetworkConfig(networks)

	var systemAdapters []string
	for _, el := range config.SystemAdapterList {
		id, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}
		_ = cloud.AddSystemAdapter(id.String(), el)
		systemAdapters = append(systemAdapters, id.String())
	}
	dev.SetSystemAdaptersConfig(systemAdapters)

	var appInstances []string
	for _, el := range config.Apps {
		_ = cloud.AddApplicationInstanceConfig(el)
		for _, img := range el.Drives {
			_ = cloud.AddImage(img.Image)
		}
		id, err := getUUID(el)
		if err != nil {
			return nil, err
		}
		appInstances = append(appInstances, id.String())
	}
	dev.SetApplicationInstanceConfig(appInstances)

	if config.Reboot != nil {
		dev.SetRebootCounter(config.Reboot.Counter, config.Reboot.DesiredState)
	}

	return dev, nil
}

//ConfigSync set config for devID
func (cloud *CloudCtx) ConfigSync(dev *device.Ctx) (err error) {
	devConfig, err := cloud.GetConfigBytes(dev, false)
	if err != nil {
		return err
	}
	if err = cloud.ConfigSet(dev.GetID(), devConfig); err != nil {
		return err
	}
	return cloud.StateUpdate(dev)
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
func (cloud *CloudCtx) GetConfigBytes(dev *device.Ctx, pretty bool) ([]byte, error) {
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
	for k, v := range dev.GetConfigItems() {
		configItems = append(configItems, &config.ConfigItem{
			Key:   k,
			Value: v,
		})
	}

	rebootCounter, rebootState := dev.GetRebootCounter()
	rebootCmd := &config.DeviceOpsCmd{Counter: rebootCounter, DesiredState: rebootState}
	devConfig := &config.EdgeDevConfig{
		Id: &config.UUIDandVersion{
			Uuid:    dev.GetID().String(),
			Version: strconv.Itoa(dev.GetConfigVersion()),
		},
		Apps:              applicationInstances,
		Networks:          networkConfigs,
		Datastores:        dataStores,
		LispInfo:          nil,
		Base:              baseOS,
		Reboot:            rebootCmd,
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
	if pretty {
		return json.MarshalIndent(devConfig, "", "    ")
	} else {
		return json.Marshal(devConfig)
	}
}
