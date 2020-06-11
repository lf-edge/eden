package controller

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/evecommon"
)

//DevModelType is type of dev model
type DevModelType string

//DevModel is dev model fields
type DevModel struct {
	//physicalIOs is PhysicalIO slice for DevModel
	physicalIOs []*config.PhysicalIO
	//networks is NetworkConfig slice for DevModel
	networks []*config.NetworkConfig
	//adapters is SystemAdapter slice for DevModel
	adapters []*config.SystemAdapter
	//adapterForSwitches is name of adapter for use in switch
	adapterForSwitches []string
	devModelType       DevModelType
}

//GetFirstAdapterForSwitches return first adapter available for switch networkInstance
func (ctx *DevModel) GetFirstAdapterForSwitches() string {
	if len(ctx.adapterForSwitches) > 0 {
		return ctx.adapterForSwitches[0]
	}
	return "uplink"
}

//GetNetDHCPID return netDHCPID id
func (ctx *DevModel) GetNetDHCPID() string {
	return defaults.NetDHCPID
}

//GetNetNoDHCPID return netNoDHCPID id
func (ctx *DevModel) GetNetNoDHCPID() string {
	return defaults.NetNoDHCPID
}

//DevModelTypeEmpty is empty model type
const DevModelTypeEmpty DevModelType = "Empty"

//DevModelTypeQemu is model type for qemu
const DevModelTypeQemu DevModelType = "ZedVirtual-4G"

//CreateDevModel create manual DevModel with provided params
func (cloud *CloudCtx) CreateDevModel(PhysicalIOs []*config.PhysicalIO, Networks []*config.NetworkConfig, Adapters []*config.SystemAdapter, AdapterForSwitches []string, modelType DevModelType) *DevModel {
	devModel := &DevModel{adapterForSwitches: AdapterForSwitches, physicalIOs: PhysicalIOs, networks: Networks, adapters: Adapters, devModelType: modelType}
	if cloud.devModels == nil {
		cloud.devModels = make(map[DevModelType]*DevModel)
	}
	cloud.devModels[modelType] = devModel
	return devModel
}

//GetDevModelByName return DevModel object by DevModelType string
func (cloud *CloudCtx) GetDevModelByName(devModelType string) (*DevModel, error) {
	return cloud.GetDevModel(DevModelType(devModelType))
}

//GetDevModel return DevModel object by DevModelType
func (cloud *CloudCtx) GetDevModel(devModelType DevModelType) (*DevModel, error) {
	switch devModelType {
	case DevModelTypeEmpty:
		return cloud.CreateDevModel(nil, nil, nil, nil, DevModelTypeEmpty), nil
	case DevModelTypeQemu:
		return cloud.CreateDevModel(
				[]*config.PhysicalIO{{
					Ptype:        evecommon.PhyIoType_PhyIoNetEth,
					Phylabel:     "eth0",
					Logicallabel: "eth0",
					Assigngrp:    "eth0",
					Phyaddrs:     map[string]string{"Ifname": "eth0"},
					Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageMgmtAndApps,
					UsagePolicy: &config.PhyIOUsagePolicy{
						FreeUplink: true,
					},
				}, {
					Ptype:        evecommon.PhyIoType_PhyIoNetEth,
					Phylabel:     "eth1",
					Logicallabel: "eth1",
					Assigngrp:    "eth1",
					Phyaddrs:     map[string]string{"Ifname": "eth1"},
					Usage:        evecommon.PhyIoMemberUsage_PhyIoUsageShared,
					UsagePolicy: &config.PhyIOUsagePolicy{
						FreeUplink: true,
					},
				},
				},
				[]*config.NetworkConfig{
					{
						Id:   defaults.NetDHCPID,
						Type: config.NetworkType_V4,
						Ip: &config.Ipspec{
							Dhcp:      config.DHCPType_Client,
							DhcpRange: &config.IpRange{},
						},
						Wireless: nil,
					},
					{
						Id:   defaults.NetNoDHCPID,
						Type: config.NetworkType_V4,
						Ip: &config.Ipspec{
							Dhcp:      config.DHCPType_DHCPNone,
							DhcpRange: &config.IpRange{},
						},
						Wireless: nil,
					},
				},
				[]*config.SystemAdapter{
					{
						Name:        "eth0",
						Uplink:      true,
						NetworkUUID: defaults.NetDHCPID,
					},
					{
						Name:        "eth1",
						NetworkUUID: defaults.NetNoDHCPID,
					},
				},
				[]string{"eth1"},
				DevModelTypeQemu),
			nil
	}
	return nil, fmt.Errorf("not implemented type: %s", devModelType)
}
