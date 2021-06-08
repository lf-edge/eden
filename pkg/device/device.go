package device

import (
	"fmt"
	"log"

	"github.com/lf-edge/eve/api/go/config"
	"github.com/satori/go.uuid"
)

//EdgeNodeState determinate state of EdgeNode
type EdgeNodeState int

var (
	//NotOnboarded EdgeNode
	NotOnboarded EdgeNodeState
	//Onboarded EdgeNode
	Onboarded EdgeNodeState = 1
)

//Ctx is base struct for EdgeNode
type Ctx struct {
	onboardKey                 string
	serial                     string
	state                      EdgeNodeState
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
	contentTrees               []string
	volumes                    []string
	configItems                map[string]string
	rebootCounter              uint32
	rebootState                bool
	devModel                   string
	remote                     bool
	remoteAddr                 string
	epoch                      int64
	cipherContexts             []*config.CipherContext
	globalProfile              string
	localProfileServer         string
	profileServerToken         string
}

//CreateEdgeNode generates EdgeNode
func CreateEdgeNode() *Ctx {
	id, _ := uuid.NewV4()
	return &Ctx{
		id:            id,
		rebootCounter: 1000,
		rebootState:   false,
		configVersion: 4,
		configItems:   map[string]string{},
		state:         NotOnboarded,
		epoch:         0,
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

//GetRemote return true if EVE is remote
func (cfg *Ctx) GetRemote() bool { return cfg.remote }

//SetRemote set remote status of EVE
func (cfg *Ctx) SetRemote(remote bool) { cfg.remote = remote }

//GetRemoteAddr return remote address to access EVE
func (cfg *Ctx) GetRemoteAddr() string { return cfg.remoteAddr }

//SetRemoteAddr set remote address to access EVE
func (cfg *Ctx) SetRemoteAddr(remoteAddr string) { cfg.remoteAddr = remoteAddr }

//GetEpoch return remote address to access EVE
func (cfg *Ctx) GetEpoch() int64 { return cfg.epoch }

//SetEpoch set remote address to access EVE
func (cfg *Ctx) SetEpoch(epoch int64) { cfg.epoch = epoch }

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

//SetContentTreeConfig set contentTrees configs by configIDs from cloud
func (cfg *Ctx) SetContentTreeConfig(configIDs []string) *Ctx {
	cfg.contentTrees = configIDs
	return cfg
}

//GetContentTrees return ContentTrees of device
func (cfg *Ctx) GetContentTrees() []string { return cfg.contentTrees }

//SetVolumeConfigs set volumes configs by configIDs from cloud
func (cfg *Ctx) SetVolumeConfigs(configIDs []string) *Ctx {
	cfg.volumes = configIDs
	return cfg
}

//GetVolumes return volumes of device
func (cfg *Ctx) GetVolumes() []string { return cfg.volumes }

//SetRebootCounter setter
func (cfg *Ctx) SetRebootCounter(counter uint32, state bool) {
	cfg.rebootCounter = counter
	cfg.rebootState = state
}

//GetRebootCounter getter
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

//GetOnboardKey getter
func (cfg *Ctx) GetOnboardKey() string {
	return cfg.onboardKey
}

//SetOnboardKey setter
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

//SetCipherContexts set CipherContexts for device
func (cfg *Ctx) SetCipherContexts(contexts []*config.CipherContext) *Ctx {
	cfg.cipherContexts = contexts
	return cfg
}

//GetCipherContexts get CipherContexts of device
func (cfg *Ctx) GetCipherContexts() []*config.CipherContext {
	return cfg.cipherContexts
}

//SetDeviceItem for setting devConfig fields
func (cfg *Ctx) SetDeviceItem(key string, val string) error {
	switch key {
	case "global_profile":
		cfg.globalProfile = val
	case "local_profile_server":
		cfg.localProfileServer = val
	case "profile_server_token":
		cfg.profileServerToken = val
	default:
		return fmt.Errorf("unsopported key: %s", key)
	}
	return nil
}

//GetGlobalProfile get globalProfile
func (cfg *Ctx) GetGlobalProfile() string {
	return cfg.globalProfile
}

//SetGlobalProfile set globalProfile
func (cfg *Ctx) SetGlobalProfile(globalProfile string) {
	cfg.globalProfile = globalProfile
}

//GetLocalProfileServer get localProfileServer
func (cfg *Ctx) GetLocalProfileServer() string {
	return cfg.localProfileServer
}

//SetLocalProfileServer set localProfileServer
func (cfg *Ctx) SetLocalProfileServer(localProfileServer string) {
	cfg.localProfileServer = localProfileServer
}

//GetProfileServerToken get profileServerToken
func (cfg *Ctx) GetProfileServerToken() string {
	return cfg.profileServerToken
}

//SetProfileServerToken set profileServerToken
func (cfg *Ctx) SetProfileServerToken(profileServerToken string) {
	cfg.profileServerToken = profileServerToken
}
