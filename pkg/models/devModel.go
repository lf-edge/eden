package models

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lf-edge/eve-api/go/config"
	"github.com/lf-edge/eve-api/go/evecommon"
)

// devModelType is type of dev model
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

// ModelFile for loading model from file
type ModelFile struct {
	IOMemberList []*PhysicalIO         `json:"ioMemberList,omitempty"`
	VlanAdapters []*config.VlanAdapter `json:"vlanAdapters,omitempty"`
	BondAdapters []*config.BondAdapter `json:"bondAdapters,omitempty"`

	// The lists below are usually not part of the device model,
	// but instead are configured dynamically in run-time.
	// However, the separation between static and dynamic config
	// is fully up to the controller, EVE receives config as a whole
	// and is able to handle run-time change of most of the config items.
	// Here in eden we allow to override otherwise hard-coded networks and
	// systemAdapters and to create fully customized configurations.
	Networks       []*config.NetworkConfig `json:"networks,omitempty"`
	SystemAdapters []*config.SystemAdapter `json:"systemAdapterList,omitempty"`
}

// OverwriteDevModelFromFile replace default config with config from provided file
func OverwriteDevModelFromFile(fileName string, model DevModel) error {
	var mFile ModelFile
	b, err := os.ReadFile(fileName)
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
	model.SetVlanAdapters(mFile.VlanAdapters)
	model.SetBondAdapters(mFile.BondAdapters)
	if len(mFile.Networks) > 0 {
		model.SetNetworks(mFile.Networks)
	}
	if len(mFile.SystemAdapters) > 0 {
		model.SetAdapters(mFile.SystemAdapters)
	}
	return nil
}

// DevModel is an interface to use for describe device
type DevModel interface {
	Adapters() []*config.SystemAdapter
	SetAdapters([]*config.SystemAdapter)
	Networks() []*config.NetworkConfig
	SetNetworks([]*config.NetworkConfig)
	PhysicalIOs() []*config.PhysicalIO
	SetPhysicalIOs([]*config.PhysicalIO)
	VlanAdapters() []*config.VlanAdapter
	SetVlanAdapters([]*config.VlanAdapter)
	BondAdapters() []*config.BondAdapter
	SetBondAdapters([]*config.BondAdapter)
	AdapterForSwitches() []string
	DevModelType() string
	SetWiFiParams(ssid string, psk string)
	GetPortConfig(ssid string, psk string) string
	DiskFormat() string
	DiskReadyMessage() string
	Config() map[string]interface{}
}

// GetDevModelByName return DevModel object by DevModelType string
func GetDevModelByName(modelType string) (DevModel, error) {
	return GetDevModel(devModelType(modelType))
}

// GetDevModel return DevModel object by DevModelType
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
