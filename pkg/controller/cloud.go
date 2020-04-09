package controller

import (
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

//CloudCtx is struct for use with controller
type CloudCtx struct {
	Controller
	devices          []*device.Ctx
	datastores       []*config.DatastoreConfig
	images           []*config.Image
	baseOS           []*config.BaseOSConfig
	networkInstances []*config.NetworkInstanceConfig
	networks         []*config.NetworkConfig
	physicalIOs      map[string]*config.PhysicalIO
	systemAdapters   map[string]*config.SystemAdapter
	devModels        map[DevModelType]*DevModel
}

//Cloud is an interface of cloud
type Cloud interface {
	Controller
	AddDevice(devUUID uuid.UUID) (dev *device.Ctx, err error)
	GetDeviceUUID(devUUID uuid.UUID) (dev *device.Ctx, err error)
	GetBaseOSConfig(id string) (baseOSConfig *config.BaseOSConfig, err error)
	AddBaseOsConfig(baseOSConfig *config.BaseOSConfig) error
	AddDataStore(dataStoreConfig *config.DatastoreConfig) error
	GetDataStore(id string) (ds *config.DatastoreConfig, err error)
	GetNetworkInstanceConfig(id string) (networkInstanceConfig *config.NetworkInstanceConfig, err error)
	AddNetworkInstanceConfig(networkInstanceConfig *config.NetworkInstanceConfig) error
	RemoveNetworkInstanceConfig(id string) error
	GetImage(id string) (image *config.Image, err error)
	AddImage(imageConfig *config.Image) error
	GetConfigBytes(dev *device.Ctx) ([]byte, error)
	GetDeviceFirst() (dev *device.Ctx, err error)
	ConfigSync(dev *device.Ctx) (err error)
	GetNetworkConfig(id string) (networkConfig *config.NetworkConfig, err error)
	AddNetworkConfig(networkInstanceConfig *config.NetworkConfig) error
	RemoveNetworkConfig(id string) error
	GetPhysicalIO(id string) (physicalIO *config.PhysicalIO, err error)
	AddPhysicalIO(id string, physicalIO *config.PhysicalIO) error
	RemovePhysicalIO(id string) error
	GetSystemAdapter(id string) (systemAdapter *config.SystemAdapter, err error)
	AddSystemAdapter(id string, systemAdapter *config.SystemAdapter) error
	RemoveSystemAdapter(id string) error
	GetDevModel(devModelType DevModelType) (*DevModel, error)
	GetDevModelByName(devModelType string) (*DevModel, error)
	CreateDevModel(PhysicalIOs []*config.PhysicalIO, Networks []*config.NetworkConfig, Adapters []*config.SystemAdapter, AdapterForSwitches []string, modelType DevModelType) *DevModel
	ApplyDevModel(dev *device.Ctx, devModel *DevModel) error
}
