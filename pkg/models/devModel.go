package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/evecommon"
)

//devModelType is type of dev model
type devModelType string

// PhysicalIO type for translation models into format of EVE`s config.PhysicalIO
type PhysicalIO struct {
	Ztype        evecommon.PhyIoType        `json:"ztype,omitempty"`
	Phylabel     string                     `json:"phylabel,omitempty"`
	Phyaddrs     map[string]string          `json:"phyaddrs,omitempty"`
	Logicallabel string                     `json:"logicallabel,omitempty"`
	Assigngrp    string                     `json:"assigngrp,omitempty"`
	Usage        evecommon.PhyIoMemberUsage `json:"usage,omitempty"`
	UsagePolicy  *config.PhyIOUsagePolicy   `json:"usagePolicy,omitempty"`
	Cbattr       map[string]string          `json:"cbattr,omitempty"`
}

func (physicalIO *PhysicalIO) translate() *config.PhysicalIO {
	return &config.PhysicalIO{
		Ptype:        physicalIO.Ztype,
		Phylabel:     physicalIO.Phylabel,
		Phyaddrs:     physicalIO.Phyaddrs,
		Logicallabel: physicalIO.Logicallabel,
		Assigngrp:    physicalIO.Assigngrp,
		Usage:        physicalIO.Usage,
		UsagePolicy:  physicalIO.UsagePolicy,
		Cbattr:       physicalIO.Cbattr,
	}
}

//ModelFile for loading model from file
type ModelFile struct {
	IOMemberList []*PhysicalIO `json:"ioMemberList,omitempty"`
}

//OverwriteDevModelFromFile replace default config with config from provided file
func OverwriteDevModelFromFile(fileName string, model DevModel) error {
	var mFile ModelFile
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &mFile); err != nil {
		return err
	}
	var ioConfigs []*config.PhysicalIO
	for _, el := range mFile.IOMemberList {
		ioConfigs = append(ioConfigs, el.translate())
	}
	model.SetPhysicalIOs(ioConfigs)
	return nil
}

//DevModel is an interface to use for describe device
type DevModel interface {
	Adapters() []*config.SystemAdapter
	Networks() []*config.NetworkConfig
	PhysicalIOs() []*config.PhysicalIO
	SetPhysicalIOs([]*config.PhysicalIO)
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
	case devModelTypeParallels:
		return createParallels()

	}
	return nil, fmt.Errorf("not implemented type: %s", devModelType)
}
