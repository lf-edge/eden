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
	drives           map[uuid.UUID]*config.Drive
	baseOS           []*config.BaseOSConfig
	networkInstances []*config.NetworkInstanceConfig
}

//Cloud is an interface of cloud
type Cloud interface {
	Controller
	AddDevice(devUUID *uuid.UUID) error
	GetDeviceUUID(devUUID *uuid.UUID) (dID *device.Ctx, err error)
	GetBaseOSConfig(ID string) (baseOSConfig *config.BaseOSConfig, err error)
	AddBaseOsConfig(baseOSConfig *config.BaseOSConfig) error
	AddDatastore(datastoreConfig *config.DatastoreConfig) error
	GetDataStore(ID string) (ds *config.DatastoreConfig, err error)
	GetNetworkInstanceConfig(id string) (networkInstanceConfig *config.NetworkInstanceConfig, err error)
	AddNetworkInstanceConfig(networkInstanceConfig *config.NetworkInstanceConfig) error
	RemoveNetworkInstanceConfig(id string) error
	GetImage(ID string) (image *config.Image, err error)
	AddImage(imageConfig *config.Image) error
	GetConfigBytes(devUUID *uuid.UUID) ([]byte, error)
	GetDeviceFirst() (devUUID *device.Ctx, err error)
	ConfigSync(devUUID *uuid.UUID) (err error)
}
