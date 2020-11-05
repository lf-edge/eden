package expect

import (
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

//NetInstanceExpectation stores options for create NetworkInstanceConfigs for apps
type NetInstanceExpectation struct {
	name          string
	subnet        string
	portsReceived []string
	ports         map[int]int
	netInstType   string
}

//checkNetworkInstance checks if provided netInst match expectation
func (exp *AppExpectation) checkNetworkInstance(netInst *config.NetworkInstanceConfig, instanceExpect *NetInstanceExpectation) bool {
	if netInst == nil {
		return false
	}
	if netInst.Ip.Subnet == instanceExpect.subnet || //if subnet defined and the same
		(instanceExpect.name != "" && netInst.Displayname == instanceExpect.name) || //if name defined and the same
		(instanceExpect.netInstType == "switch" && netInst.InstType == config.ZNetworkInstType_ZnetInstSwitch) { //only one switch for now
		return true
	}
	return false
}

//createNetworkInstance creates NetworkInstanceConfig for AppExpectation
func (exp *AppExpectation) createNetworkInstance(instanceExpect *NetInstanceExpectation) (*config.NetworkInstanceConfig, error) {
	var netInst *config.NetworkInstanceConfig
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	subentIPs := utils.GetSubnetIPs(instanceExpect.subnet)
	netInst = &config.NetworkInstanceConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		InstType: config.ZNetworkInstType_ZnetInstLocal, //we use local networks for now
		Activate: true,
		Port:     exp.uplinkAdapter,
		Cfg:      &config.NetworkInstanceOpaqueConfig{},
		IpType:   config.AddressType_IPV4,
		Ip: &config.Ipspec{
			Subnet:  instanceExpect.subnet,
			Gateway: subentIPs[1].String(),
			Dns:     []string{subentIPs[1].String()},
			DhcpRange: &config.IpRange{
				Start: subentIPs[2].String(),
				End:   subentIPs[len(subentIPs)-2].String(),
			},
		},
		Dns: nil,
	}
	if instanceExpect.netInstType == "switch" {
		netInst.InstType = config.ZNetworkInstType_ZnetInstSwitch
		devModel, err := exp.ctrl.GetDevModelByName(exp.ctrl.GetVars().DevModel)
		if err != nil {
			log.Fatal(err)
		}
		netInst.Port = &config.Adapter{Name: devModel.GetFirstAdapterForSwitches()}
		netInst.Ip = &config.Ipspec{}
	}
	if instanceExpect.name == "" {
		rand.Seed(time.Now().UnixNano())
		instanceExpect.name = namesgenerator.GetRandomName(0)
	}
	netInst.Displayname = instanceExpect.name
	return netInst, nil
}

//NetworkInstances expects network instances in cloud
//it iterates over NetworkInstanceConfigs from exp.netInstances, gets or creates new one, if not exists
func (exp *AppExpectation) NetworkInstances() (networkInstances map[*NetInstanceExpectation]*config.NetworkInstanceConfig) {
	networkInstances = make(map[*NetInstanceExpectation]*config.NetworkInstanceConfig)
	for _, ni := range exp.netInstances {
		var err error
		var networkInstance *config.NetworkInstanceConfig
		for _, netInst := range exp.ctrl.ListNetworkInstanceConfig() {
			if exp.checkNetworkInstance(netInst, ni) {
				networkInstance = netInst
				break
			}
		}
		if networkInstance == nil { //if networkInstance not exists, create it
			if ni.name != "" && ni.subnet == "" {
				log.Fatalf("not found subnet with name %s", ni.name)
			}
			if networkInstance, err = exp.createNetworkInstance(ni); err != nil {
				log.Fatalf("cannot create NetworkInstance: %s", err)
			}
			if err = exp.ctrl.AddNetworkInstanceConfig(networkInstance); err != nil {
				log.Fatalf("AddNetworkInstanceConfig: %s", err)
			}
		}
		networkInstances[ni] = networkInstance
	}
	return
}
