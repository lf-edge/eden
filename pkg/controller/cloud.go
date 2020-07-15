package controller

import (
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

//CloudCtx is struct for use with controller
type CloudCtx struct {
	Controller
	devices              []*device.Ctx
	datastores           []*config.DatastoreConfig
	images               []*config.Image
	contentTrees         []*config.ContentTree
	volumes              []*config.Volume
	baseOS               []*config.BaseOSConfig
	networkInstances     []*config.NetworkInstanceConfig
	networks             []*config.NetworkConfig
	physicalIOs          map[string]*config.PhysicalIO
	systemAdapters       map[string]*config.SystemAdapter
	applicationInstances []*config.AppInstanceConfig
	devModels            map[DevModelType]*DevModel
	vars                 *utils.ConfigVars
}

//Cloud is an interface of cloud
type Cloud interface {
	Controller
	AddDevice(devUUID uuid.UUID) (dev *device.Ctx, err error)
	GetEdgeNode(name string) *device.Ctx
	GetDeviceUUID(devUUID uuid.UUID) (dev *device.Ctx, err error)
	GetBaseOSConfig(id string) (baseOSConfig *config.BaseOSConfig, err error)
	ListBaseOSConfig() []*config.BaseOSConfig
	AddBaseOsConfig(baseOSConfig *config.BaseOSConfig) error
	RemoveBaseOsConfig(id string) error
	AddDataStore(dataStoreConfig *config.DatastoreConfig) error
	GetDataStore(id string) (ds *config.DatastoreConfig, err error)
	RemoveDataStore(id string) error
	ListDataStore() []*config.DatastoreConfig
	GetNetworkInstanceConfig(id string) (networkInstanceConfig *config.NetworkInstanceConfig, err error)
	AddNetworkInstanceConfig(networkInstanceConfig *config.NetworkInstanceConfig) error
	ListNetworkInstanceConfig() []*config.NetworkInstanceConfig
	RemoveNetworkInstanceConfig(id string) error
	GetImage(id string) (image *config.Image, err error)
	AddImage(imageConfig *config.Image) error
	RemoveImage(id string) error
	ListImage() []*config.Image
	GetContentTree(id string) (image *config.ContentTree, err error)
	AddContentTree(contentTreeConfig *config.ContentTree) error
	RemoveContentTree(id string) error
	ListContentTree() []*config.ContentTree
	GetVolume(id string) (image *config.Volume, err error)
	AddVolume(volumeConfig *config.Volume) error
	RemoveVolume(id string) error
	ListVolume() []*config.Volume
	GetConfigBytes(dev *device.Ctx, pretty bool) ([]byte, error)
	GetDeviceFirst() (dev *device.Ctx, err error)
	ConfigSync(dev *device.Ctx) (err error)
	ConfigParse(config *config.EdgeDevConfig) (dev *device.Ctx, err error)
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
	GetApplicationInstanceConfig(id string) (applicationInstanceConfig *config.AppInstanceConfig, err error)
	AddApplicationInstanceConfig(applicationInstanceConfig *config.AppInstanceConfig) error
	RemoveApplicationInstanceConfig(id string) error
	ListApplicationInstanceConfig() []*config.AppInstanceConfig
	StateUpdate(dev *device.Ctx) (err error)
	OnBoardDev(node *device.Ctx) error
	GetVars() *utils.ConfigVars
	SetVars(*utils.ConfigVars)
	GetAllNodes()
}
