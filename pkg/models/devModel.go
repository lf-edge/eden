package models

import (
	"fmt"

	"github.com/lf-edge/eve/api/go/config"
)

//devModelType is type of dev model
type devModelType string

//DevModel is an interface to use for describe device
type DevModel interface {
	Adapters() []*config.SystemAdapter
	Networks() []*config.NetworkConfig
	PhysicalIOs() []*config.PhysicalIO
	AdapterForSwitches() []string
	DevModelType() string
	GetFirstAdapterForSwitches() string
	SetWiFiParams(ssid string, psk string)
	GetPortConfig(ssid string, psk string) string
	DiskFormat() string
	DiskReadyMessage() string
	Config() map[string]interface{}
}

//GetDevModelByName return DevModel object by DevModelType string
func GetDevModelByName(modelType string) (DevModel, error) {
	return GetDevModel(devModelType(modelType))
}

//GetDevModel return DevModel object by DevModelType
func GetDevModel(devModelType devModelType) (DevModel, error) {
	switch devModelType {
	case devModelTypeQemu:
		return createQemu()
	case devModelTypeGeneral:
		return createGeneral()
	case devModelTypeGCP:
		return createGCP()
	case devModelTypeRaspberry:
		return createRpi()
	case devModelTypeVBox:
		return createVBox()
	}
	return nil, fmt.Errorf("not implemented type: %s", devModelType)
}
