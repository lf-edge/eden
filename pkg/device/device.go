package device

import (
	"encoding/json"
	"errors"
	"github.com/lf-edge/eden/pkg/cloud"
	"github.com/lf-edge/eve/api/go/config"
)
import "github.com/satori/go.uuid"

//Ctx is base struct for device
type Ctx struct {
	id               uuid.UUID
	baseOSConfigs    []string
	networkInstances []string
	cloud            *cloud.Ctx
}

//CreateWithBaseConfig generate base config for device with id and associate with cloudCtx
func CreateWithBaseConfig(id uuid.UUID, cloudCtx *cloud.Ctx) *Ctx {
	return &Ctx{
		id:    id,
		cloud: cloudCtx,
	}
}

func checkIfDatastoresContains(id string, ds []*config.DatastoreConfig) bool {
	for _, el := range ds {
		if el.Id == id {
			return true
		}
	}
	return false
}

//GenerateJSONBytes generate json representation of device config
func (cfg *Ctx) GenerateJSONBytes() ([]byte, error) {
	var BaseOS []*config.BaseOSConfig
	var DataStores []*config.DatastoreConfig
	for _, baseOSConfigID := range cfg.baseOSConfigs {
		baseOSConfig, err := cfg.cloud.GetBaseOSConfig(baseOSConfigID)
		if err != nil {
			return nil, err
		}
		for _, drive := range baseOSConfig.Drives {
			if drive.Image == nil {
				return nil, errors.New("empty Image in Drive")
			}
			dataStore, err := cfg.cloud.GetDataStore(drive.Image.DsId)
			if err != nil {
				return nil, err
			}
			if !checkIfDatastoresContains(dataStore.Id, DataStores) {
				DataStores = append(DataStores, dataStore)
			}
		}
		BaseOS = append(BaseOS, baseOSConfig)
	}
	var NetworkInstanceConfigs []*config.NetworkInstanceConfig
	for _, networkInstanceConfigID := range cfg.networkInstances {
		networkInstanceConfig, err := cfg.cloud.GetNetworkInstanceConfig(networkInstanceConfigID)
		if err != nil {
			return nil, err
		}
		NetworkInstanceConfigs = append(NetworkInstanceConfigs, networkInstanceConfig)
	}
	devConfig := &config.EdgeDevConfig{
		Id: &config.UUIDandVersion{
			Uuid:    cfg.id.String(),
			Version: "4",
		},
		Apps:              nil,
		Networks:          nil,
		Datastores:        DataStores,
		LispInfo:          nil,
		Base:              BaseOS,
		Reboot:            nil,
		Backup:            nil,
		ConfigItems:       nil,
		SystemAdapterList: nil,
		DeviceIoList:      nil,
		Manufacturer:      "",
		ProductName:       "",
		NetworkInstances:  NetworkInstanceConfigs,
		Enterprise:        "",
		Name:              "",
	}
	return json.Marshal(devConfig)
}
