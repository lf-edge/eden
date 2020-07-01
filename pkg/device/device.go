package device

import (
	"github.com/satori/go.uuid"
	"log"
)

//EdgeNodeState determinate state of EdgeNode
type EdgeNodeState int

var (
	NotOnboarded EdgeNodeState = 0 //NotOnboarded EdgeNode
	Onboarded    EdgeNodeState = 1 //Onboarded EdgeNode
)

//Ctx is base struct for EdgeNode
type Ctx struct {
	onboardKey                 string
	serial                     string
	state                      EdgeNodeState
	name                       string
	project                    string
	hash                       [32]byte
	id                         uuid.UUID
	configVersion              int
	baseOSConfigs              []string
	networkInstances           []string
	adaptersForSwitch          []string
	networks                   []string
	physicalIO                 []string
	systemAdapters             []string
	applicationInstanceConfigs []string
	configItems                map[string]string
	rebootCounter              uint32
	rebootState                bool
	devModel                   string
}

//CreateEdgeNode generates EdgeNode
func CreateEdgeNode() *Ctx {
	configItems := map[string]string{"timer.config.interval": "5"}
	id, _ := uuid.NewV4()
	return &Ctx{
		id:            id,
		rebootCounter: 0,
		rebootState:   false,
		configItems:   configItems,
		configVersion: 4,
		state:         NotOnboarded,
	}
}

//GetID return id of device
func (cfg *Ctx) GetID() uuid.UUID { return cfg.id }

//GetConfigVersion return configVersion of device
func (cfg *Ctx) GetConfigVersion() int { return cfg.configVersion }

//SetConfigVersion set configVersion of device
func (cfg *Ctx) SetConfigVersion(version int) { cfg.configVersion = version }

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

//GetConfigItems return GetConfigItems of device
func (cfg *Ctx) GetConfigItems() map[string]string { return cfg.configItems }

//SetConfigItem set ConfigItem of device
func (cfg *Ctx) SetConfigItem(key, val string) { cfg.configItems[key] = val }

//GetDevModel return devModel of device
func (cfg *Ctx) GetDevModel() string { return cfg.devModel }

//GetApplicationInstances return applicationInstanceConfigs of device
func (cfg *Ctx) GetApplicationInstances() []string { return cfg.applicationInstanceConfigs }

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

//SetDevModel set devModel
func (cfg *Ctx) SetDevModel(devModel string) {
	cfg.devModel = devModel
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

//SetProject setter
func (cfg *Ctx) SetProject(name string) {
	cfg.project = name
}

//Reboot node by incrementing RebootCounter
func (cfg *Ctx) Reboot() {
	if cfg == nil {
		log.Fatal("EdgeNode not initialized")
	}
	c, _ := cfg.GetRebootCounter()
	cfg.SetRebootCounter(c+1, true)
}

//GetName getter
func (cfg *Ctx) GetName() string {
	return cfg.name
}

//SetName setter
func (cfg *Ctx) SetName(name string) {
	cfg.name = name
}

//GetState setter
func (cfg *Ctx) GetState() EdgeNodeState {
	return cfg.state
}

//SetState setter
func (cfg *Ctx) SetState(state EdgeNodeState) {
	cfg.state = state
}

//GetSerial getter
func (cfg *Ctx) GetSerial() string {
	return cfg.serial
}

//SetSerial setter
func (cfg *Ctx) SetSerial(serial string) {
	cfg.serial = serial
}

//GetOnboardKey setter
func (cfg *Ctx) GetOnboardKey() string {
	return cfg.onboardKey
}

//GetOnboardKey getter
func (cfg *Ctx) SetOnboardKey(key string) {
	cfg.onboardKey = key
}

//SetID setter
func (cfg *Ctx) SetID(id uuid.UUID) {
	cfg.id = id
}

//CheckHash check hash and update
//returns true if hash is new
func (cfg *Ctx) CheckHash(newHash [32]byte) bool {
	if cfg.hash != newHash {
		cfg.hash = newHash
		return true
	}
	return false
}
