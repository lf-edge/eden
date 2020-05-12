package device

import (
	"github.com/satori/go.uuid"
)

//Ctx is base struct for device
type Ctx struct {
	id                         uuid.UUID
	baseOSConfigs              []string
	networkInstances           []string
	adaptersForSwitch          []string
	networks                   []string
	physicalIO                 []string
	systemAdapters             []string
	applicationInstanceConfigs []string
	sshKeys                    []string
	rebootCounter              uint32
	rebootState                bool
	devModel                   string
	vncAccess                  bool
	controllerLogLevel         string
	timerConfigInterval        int
}

//CreateWithBaseConfig generate base config for device with id and associate with cloudCtx
func CreateWithBaseConfig(id uuid.UUID) *Ctx {
	return &Ctx{
		id:                  id,
		rebootCounter:       1000,
		rebootState:         false,
		timerConfigInterval: 5,
	}
}

//GetID return id of device
func (cfg *Ctx) GetID() uuid.UUID { return cfg.id }

//GetBaseOSConfigs return baseOSConfigs of device
func (cfg *Ctx) GetBaseOSConfigs() []string { return cfg.baseOSConfigs }

//GetNetworkInstances return networkInstances of device
func (cfg *Ctx) GetNetworkInstances() []string { return cfg.networkInstances }

//GetNetworks return networks of device
func (cfg *Ctx) GetNetworks() []string { return cfg.networks }

//GetPhysicalIOs return physicalIO of device
func (cfg *Ctx) GetPhysicalIOs() []string { return cfg.physicalIO }

//GetSystemAdapters return systemAdapters of device
func (cfg *Ctx) GetSystemAdapters() []string { return cfg.systemAdapters }

//GetSSHKeys return sshKeys of device
func (cfg *Ctx) GetSSHKeys() []string { return cfg.sshKeys }

//GetDevModel return devModel of device
func (cfg *Ctx) GetDevModel() string { return cfg.devModel }

//GetApplicationInstances return applicationInstanceConfigs of device
func (cfg *Ctx) GetApplicationInstances() []string { return cfg.applicationInstanceConfigs }

//GetVncAccess return vncAccess of device
func (cfg *Ctx) GetVncAccess() bool { return cfg.vncAccess }

//GetControllerLogLevel return controllerLogLevel of device
func (cfg *Ctx) GetControllerLogLevel() string { return cfg.controllerLogLevel }

//GetTimerConfigInterval return timer.config.interval
func (cfg *Ctx) GetTimerConfigInterval() int { return cfg.timerConfigInterval }

//GetAdaptersForSwitch return adaptersForSwitch of device
func (cfg *Ctx) GetAdaptersForSwitch() []string {
	return cfg.adaptersForSwitch
}

//SetAdaptersForSwitch set adaptersForSwitch of device
func (cfg *Ctx) SetAdaptersForSwitch(adaptersForSwitch []string) {
	cfg.adaptersForSwitch = adaptersForSwitch
}

//SetBaseOSConfig set BaseOSConfig by configIDs from cloud
func (cfg *Ctx) SetBaseOSConfig(configIDs []string) *Ctx {
	cfg.baseOSConfigs = configIDs
	return cfg
}

//SetNetworkInstanceConfig set NetworkInstanceConfig by configIDs from cloud
func (cfg *Ctx) SetNetworkInstanceConfig(configIDs []string) *Ctx {
	cfg.networkInstances = configIDs
	return cfg
}

//SetNetworkConfig set networks by configIDs from cloud
func (cfg *Ctx) SetNetworkConfig(configIDs []string) *Ctx {
	cfg.networks = configIDs
	return cfg
}

//SetPhysicalIOConfig set physicalIO by configIDs from cloud
func (cfg *Ctx) SetPhysicalIOConfig(configIDs []string) *Ctx {
	cfg.physicalIO = configIDs
	return cfg
}

//SetSystemAdaptersConfig set systemAdapters by configIDs from cloud
func (cfg *Ctx) SetSystemAdaptersConfig(configIDs []string) *Ctx {
	cfg.systemAdapters = configIDs
	return cfg
}

//SetSSHKeys set sshKeys
func (cfg *Ctx) SetSSHKeys(keys []string) *Ctx {
	cfg.sshKeys = keys
	return cfg
}

//SetDevModel set devModel
func (cfg *Ctx) SetDevModel(devModel string) {
	cfg.devModel = devModel
}

//SetControllerLogLevel set controllerLogLevel
func (cfg *Ctx) SetControllerLogLevel(controllerLogLevel string) {
	cfg.controllerLogLevel = controllerLogLevel
}

//SetVncAccess set vncAccess
func (cfg *Ctx) SetVncAccess(enabled bool) {
	cfg.vncAccess = enabled
}

//SetTimerConfigInterval set timer.config.interval
func (cfg *Ctx) SetTimerConfigInterval(interval int) {
	cfg.timerConfigInterval = interval
}

//SetApplicationInstanceConfig set applicationInstanceConfigs by configIDs from cloud
func (cfg *Ctx) SetApplicationInstanceConfig(configIDs []string) *Ctx {
	cfg.applicationInstanceConfigs = configIDs
	return cfg
}

//SetRebootCounter setter
func (cfg *Ctx) SetRebootCounter(counter uint32, state bool) {
	cfg.rebootCounter = counter
	cfg.rebootState = state
}

//SetRebootCounter getter
func (cfg *Ctx) GetRebootCounter() (counter uint32, state bool) {
	return cfg.rebootCounter, cfg.rebootState
}
