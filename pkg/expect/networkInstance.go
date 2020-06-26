package expect

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//checkNetworkInstance checks if provided netInst match expectation
func (exp *appExpectation) checkNetworkInstance(netInst *config.NetworkInstanceConfig) bool {
	if netInst == nil {
		return false
	}
	if netInst.Ip.Subnet == defaults.DefaultAppSubnet {
		return true
	}
	return false
}

//createNetworkInstance creates NetworkInstanceConfig for appExpectation
func (exp *appExpectation) createNetworkInstance() (*config.NetworkInstanceConfig, error) {
	var netInst *config.NetworkInstanceConfig
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	subentIPs := utils.GetSubnetIPs(defaults.DefaultAppSubnet) //we use predefined subnet for now
	netInst = &config.NetworkInstanceConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		Displayname: "local",
		InstType:    config.ZNetworkInstType_ZnetInstLocal, //we use local networks for now
		Activate:    true,
		Port:        exp.uplinkAdapter,
		Cfg:         &config.NetworkInstanceOpaqueConfig{},
		IpType:      config.AddressType_IPV4,
		Ip: &config.Ipspec{
			Subnet:  defaults.DefaultAppSubnet,
			Gateway: subentIPs[1].String(),
			Dns:     []string{subentIPs[1].String()},
			DhcpRange: &config.IpRange{
				Start: subentIPs[2].String(),
				End:   subentIPs[len(subentIPs)-2].String(),
			},
		},
		Dns: nil,
	}
	return netInst, nil
}

//NetworkInstance expects network instance in cloud
//it gets NetworkInstanceConfig with defined in appExpectation params, or creates new one, if not exists
func (exp *appExpectation) NetworkInstance() (networkInstance *config.NetworkInstanceConfig) {
	var err error
	for _, netInst := range exp.ctrl.ListNetworkInstanceConfig() {
		if exp.checkNetworkInstance(netInst) {
			networkInstance = netInst
			break
		}
	}
	if networkInstance == nil { //if networkInstance not exists, create it
		if networkInstance, err = exp.createNetworkInstance(); err != nil {
			log.Fatalf("cannot create NetworkInstance: %s", err)
		}
		if err = exp.ctrl.AddNetworkInstanceConfig(networkInstance); err != nil {
			log.Fatalf("AddNetworkInstanceConfig: %s", err)
		}
	}
	return
}
