package expect

import (
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/lf-edge/eden/pkg/models"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/evecommon"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//NetInstanceExpectation stores options for create NetworkInstanceConfigs for apps
type NetInstanceExpectation struct {
	name          string
	subnet        string
	portsReceived []string
	ports         map[int]int
	netInstType   string
	uplinkAdapter string
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
	adapter := exp.uplinkAdapter
	if instanceExpect.uplinkAdapter != "" {
		adapter = &config.Adapter{
			Name: instanceExpect.uplinkAdapter,
			Type: evecommon.PhyIoType_PhyIoNetEth,
		}
	}
	netInst = &config.NetworkInstanceConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		InstType: config.ZNetworkInstType_ZnetInstLocal, //we use local networks for now
		Activate: true,
		Port:     adapter,
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
		devModel, err := models.GetDevModelByName(exp.ctrl.GetVars().DevModel)
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
		for _, netInstID := range exp.device.GetNetworkInstances() {
			netInst, err := exp.ctrl.GetNetworkInstanceConfig(netInstID)
			if err != nil {
				log.Fatalf("no baseOS %s found in controller: %s", netInstID, err)
			}
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

// parseACE returns ACE from string notation
func parseACE(inp string) *config.ACE {
	//set default to host
	aclType := "host"
	if _, _, err := net.ParseCIDR(inp); err == nil {
		//check if it is subnet
		aclType = "ip"
	} else {
		if ip := net.ParseIP(inp); ip != nil {
			//check if it is ip
			aclType = "ip"
		}
	}
	return &config.ACE{
		Matches: []*config.ACEMatch{{
			Type:  aclType,
			Value: inp,
		}},
		Dir: config.ACEDirection_BOTH,
	}
}

// getAcls returns rules for access/deny/forwarding traffic
func (exp *AppExpectation) getAcls(ni *NetInstanceExpectation) []*config.ACE {
	var acls []*config.ACE
	var aclID int32 = 1
	if exp.acl != nil {
		// in case of defined acl allow access only to them
		for _, el := range exp.acl {
			acl := parseACE(el)
			if acl != nil {
				acl.Id = aclID
				acls = append(acls, acl)
				aclID++
			}
		}
	} else {
		// allow access to all addresses
		acls = append(acls, &config.ACE{
			Matches: []*config.ACEMatch{{
				Type:  "ip",
				Value: "0.0.0.0/0",
			}},
			Id: aclID,
		})
		aclID++
	}
	if ni.ports != nil {
		// forward defined ports
		for po, pi := range ni.ports {
			acls = append(acls, &config.ACE{
				Id: aclID,
				Matches: []*config.ACEMatch{{
					Type:  "protocol",
					Value: "tcp",
				}, {
					Type:  "lport",
					Value: strconv.Itoa(po),
				}},
				Actions: []*config.ACEAction{{
					Portmap: true,
					AppPort: uint32(pi),
				}},
				Dir: config.ACEDirection_BOTH})
			aclID++
		}
	}
	return acls
}
