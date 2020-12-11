package controller

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

//StateUpdate refresh state file
func (cloud *CloudCtx) StateUpdate(dev *device.Ctx) (err error) {
	devConfig, err := cloud.GetConfigBytes(dev, true)
	if err != nil {
		return err
	}
	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	return utils.GenerateStateFile(edenDir, utils.StateObject{
		EveConfig:  string(devConfig),
		EveDir:     cloud.GetVars().EveDist,
		AdamDir:    cloud.GetDir(),
		EveUUID:    cloud.GetVars().EveUUID,
		DeviceUUID: dev.GetID().String(),
		QEMUConfig: cloud.GetVars().EveQemuConfig,
	})
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
	devID, err := getID(config)
	if err != nil {
		return nil, fmt.Errorf("problem with uuid field")
	}
	dev, err := cloud.GetDeviceUUID(devID)
	if err != nil { //not found
		dev, err = cloud.AddDevice(devID)
		if err != nil {
			return nil, fmt.Errorf("cloud.AddDevice: %s", err)
		}
	}
	version, _ := strconv.Atoi(config.Id.Version)
	dev.SetName(config.Name)
	dev.SetConfigVersion(version)
	dev.SetDevModel(config.ProductName)
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

	var volumes []string
	for _, el := range config.Volumes {
		_ = cloud.AddVolume(el)
		volumes = append(volumes, el.Uuid)
	}
	dev.SetVolumeConfigs(volumes)

	var contentTrees []string
	for _, el := range config.ContentInfo {
		_ = cloud.AddContentTree(el)
		contentTrees = append(contentTrees, el.Uuid)
	}
	dev.SetContentTreeConfig(contentTrees)

	if config.Reboot != nil {
		dev.SetRebootCounter(config.Reboot.Counter, config.Reboot.DesiredState)
	}
	res, err := cloud.GetConfigBytes(dev, false)
	if err != nil {
		return nil, fmt.Errorf("GetConfigBytes error: %s", err)
	}
	dev.CheckHash(sha256.Sum256(res))
	dev.SetRemote(cloud.vars.EveRemote)
	dev.SetRemoteAddr(cloud.vars.EveRemoteAddr)
	return dev, nil
}

//ConfigSync set config for devID
func (cloud *CloudCtx) ConfigSync(dev *device.Ctx) (err error) {
	devConfig, err := cloud.GetConfigBytes(dev, false)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(devConfig)
	if dev.CheckHash(hash) {
		fmt.Println(string(devConfig)) //print config only if changed
		if devConfig, err = VersionIncrement(devConfig); err != nil {
			return fmt.Errorf("VersionIncrement error: %s", err)
		}
		if err = cloud.ConfigSet(dev.GetID(), devConfig); err != nil {
			return err
		}
		time.Sleep(time.Second)
		return cloud.StateUpdate(dev)
	}
	return nil
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

//GetDeviceCurrent return current device object
func (cloud *CloudCtx) GetDeviceCurrent() (dev *device.Ctx, err error) {
	id, err := cloud.DeviceGetByOnboardUUID(cloud.vars.EveUUID)
	if err != nil {
		return nil, err
	}
	if len(cloud.devices) == 0 {
		return nil, errors.New("no device found")
	}
	return cloud.GetDeviceUUID(id)
}

func (cloud *CloudCtx) processDev(id uuid.UUID, state device.EdgeNodeState) {
	configString, err := cloud.ConfigGet(id)
	if err != nil {
		log.Fatalf("ConfigGet error: %s", err)
	}
	var deviceConfig config.EdgeDevConfig
	err = protojson.Unmarshal([]byte(configString), &deviceConfig)
	if err != nil {
		log.Fatalf("unmarshal error: %s", err)
	}
	dev, err := cloud.ConfigParse(&deviceConfig)
	if err != nil {
		log.Fatalf("configParse error: %s", err)
	}
	dev.SetState(state)
	cloud.devices = append(cloud.devices, dev)

}

//GetAllNodes obtains all devices from controller
func (cloud *CloudCtx) GetAllNodes() {
	nodes, err := cloud.DeviceList(types.RegisteredDeviceFilter)
	if err != nil {
		log.Fatalf("DeviceList: %s", err)
	}
	for _, el := range nodes {
		id, err := uuid.FromString(el)
		if err != nil {
			log.Fatalf("Cannot parse Device UUID %s", err)
		}
		cloud.processDev(id, device.Onboarded)
	}
	nodes, err = cloud.DeviceList(types.NotRegisteredDeviceFilter)
	if err != nil {
		log.Fatalf("DeviceList: %s", err)
	}
	for _, el := range nodes {
		id, err := uuid.FromString(el)
		if err != nil {
			log.Fatalf("Cannot parse Device UUID %s", err)
		}
		cloud.processDev(id, device.NotOnboarded)
	}
}

//GetEdgeNode by name
func (cloud *CloudCtx) GetEdgeNode(name string) *device.Ctx {
	if name == "" {
		node, _ := cloud.GetDeviceCurrent()
		if node != nil {
			return node
		}
		return nil
	}
	if len(cloud.devices) == 0 {
		return nil
	}
	for _, el := range cloud.devices {
		if el.GetName() == name {
			return el
		}
	}
	return nil
}

//AddDevice add device with specified devUUID
func (cloud *CloudCtx) AddDevice(devUUID uuid.UUID) (dev *device.Ctx, err error) {
	for _, el := range cloud.devices {
		if el.GetID().String() == devUUID.String() {
			return nil, errors.New("already exists")
		}
	}
	dev = device.CreateEdgeNode()
	dev.SetID(devUUID)
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

//checkContentTreeDs checks dataStores and adds one from contentTree if needed
func (cloud *CloudCtx) checkContentTreeDs(contentTree *config.ContentTree, dataStores []*config.DatastoreConfig) (result []*config.DatastoreConfig, err error) {
	dataStore, err := cloud.GetDataStore(contentTree.GetDsId())
	if err != nil {
		return nil, err
	}
	if !checkIfDatastoresContains(dataStore.Id, dataStores) {
		return append(dataStores, dataStore), nil
	}
	return dataStores, nil
}

//checkDriveDs checks dataStores and adds one from drive if needed
func (cloud *CloudCtx) checkDriveDs(drive *config.Drive, dataStores []*config.DatastoreConfig) (result []*config.DatastoreConfig, err error) {
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
	var contentTrees []*config.ContentTree
	var volumes []*config.Volume
	var baseOS []*config.BaseOSConfig
	var dataStores []*config.DatastoreConfig
	for _, baseOSConfigID := range dev.GetBaseOSConfigs() {
		baseOSConfig, err := cloud.GetBaseOSConfig(baseOSConfigID)
		if err != nil {
			return nil, err
		}
		//check drives from baseOSConfigs
		for _, drive := range baseOSConfig.Drives {
			dataStores, err = cloud.checkDriveDs(drive, dataStores)
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
		for _, drive := range applicationInstance.Drives {
			dataStores, err = cloud.checkDriveDs(drive, dataStores)
			if err != nil {
				return nil, err
			}
		}
	volumeLoop:
		//we must check VolumeRefList of applicationInstance and add volumes and contentTrees into EdgeDevConfig
		for _, volumeRef := range applicationInstance.VolumeRefList {
			volumeConfig, err := cloud.GetVolume(volumeRef.Uuid)
			if err != nil {
				return nil, err
			}
			volumes = append(volumes, volumeConfig)
			if contentTreeConfig, err := cloud.GetContentTree(volumeConfig.Origin.DownloadContentTreeID); err == nil {
				for _, contentTree := range contentTrees {
					if contentTree.Uuid == contentTreeConfig.Uuid {
						//we already define this contentTree in EdgeDevConfig
						continue volumeLoop
					}
				}
				//add required datastores
				dataStores, err = cloud.checkContentTreeDs(contentTreeConfig, dataStores)
				if err != nil {
					return nil, err
				}
				contentTrees = append(contentTrees, contentTreeConfig)
			}
		}
		networkInstanceConfigArray := dev.GetNetworkInstances()
		//check network instances from apps
		for _, networkInstanceConfig := range applicationInstance.Interfaces {
			if networkInstanceConfig != nil && networkInstanceConfig.NetworkId != "" {
				if _, ok := utils.FindEleInSlice(networkInstanceConfigArray, networkInstanceConfig.NetworkId); !ok {
					networkInstanceConfigArray = append(networkInstanceConfigArray, networkInstanceConfig.NetworkId)
				}
			}
		}
		dev.SetNetworkInstanceConfig(networkInstanceConfigArray)
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
	//we need to sort to keep sha of config persist
	var configItems []*config.ConfigItem
	keys := make([]string, len(dev.GetConfigItems()))
	i := 0
	for k := range dev.GetConfigItems() {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		configItems = append(configItems, &config.ConfigItem{
			Key:   k,
			Value: dev.GetConfigItems()[k],
		})
	}

	rebootCounter, rebootState := dev.GetRebootCounter()
	rebootCmd := &config.DeviceOpsCmd{Counter: rebootCounter, DesiredState: rebootState}
	devConfig := &config.EdgeDevConfig{
		Id: &config.UUIDandVersion{
			Uuid:    dev.GetID().String(),
			Version: strconv.Itoa(dev.GetConfigVersion()),
		},
		Volumes:           volumes,
		ContentInfo:       contentTrees,
		Apps:              applicationInstances,
		Networks:          networkConfigs,
		Datastores:        dataStores,
		Base:              baseOS,
		Reboot:            rebootCmd,
		Backup:            nil,
		ConfigItems:       configItems,
		SystemAdapterList: systemAdapterConfigs,
		DeviceIoList:      physicalIOs,
		Manufacturer:      "",
		ProductName:       dev.GetDevModel(),
		NetworkInstances:  networkInstanceConfigs,
		Enterprise:        "",
		Name:              dev.GetName(),
	}
	if pretty {
		return json.MarshalIndent(devConfig, "", "    ")
	}
	return json.Marshal(devConfig)
}
