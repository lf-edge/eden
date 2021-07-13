package expect

import (
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/evecommon"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//NetInstanceExpectation stores options for create NetworkInstanceConfigs for apps
type NetInstanceExpectation struct {
	mac              string
	name             string
	subnet           string
	portsReceived    []string
	ports            map[int]int
	netInstType      string
	uplinkAdapter    string
	staticDNSEntries map[string][]string
}

//checkNetworkInstance checks if provided netInst match expectation
func (exp *AppExpectation) checkNetworkInstance(netInst *config.NetworkInstanceConfig, instanceExpect *NetInstanceExpectation) bool {
	if netInst == nil {
		return false
	}
	if (netInst.Ip.Subnet != "" && netInst.Ip.Subnet == instanceExpect.subnet) || //if subnet defined and the same
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
	adapter := exp.uplinkAdapter
	if instanceExpect.uplinkAdapter == "none" {
		adapter = nil
	} else if instanceExpect.uplinkAdapter != "" {
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
		Ip:       &config.Ipspec{},
	}
	if instanceExpect.netInstType == "switch" {
		netInst.InstType = config.ZNetworkInstType_ZnetInstSwitch
	} else {
		subentIPs := utils.GetSubnetIPs(instanceExpect.subnet)
		netInst.Ip = &config.Ipspec{
			Subnet:  instanceExpect.subnet,
			Gateway: subentIPs[1].String(),
			Dns:     []string{subentIPs[1].String()},
			DhcpRange: &config.IpRange{
				Start: subentIPs[2].String(),
				End:   subentIPs[len(subentIPs)-2].String(),
			},
		}
	}
	if instanceExpect.name == "" {
		rand.Seed(time.Now().UnixNano())
		instanceExpect.name = namesgenerator.GetRandomName(0)
	}
	netInst.Displayname = instanceExpect.name
	for hostname, ipAddrs := range instanceExpect.staticDNSEntries {
		netInst.Dns = append(netInst.Dns, &config.ZnetStaticDNSEntry{
			HostName: hostname,
			Address:  ipAddrs,
		})
	}
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
			if ni.name != "" && ni.netInstType == "local" && ni.subnet == "" {
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
func parseACE(ace ACE) *config.ACE {
	//set default to host
	aclType := "host"
	ep := ace.Endpoint
	if ep == defaults.DefaultHostOnlyNotation {
		//special case for host only acl
		ep = ""
	} else {
		if _, _, err := net.ParseCIDR(ep); err == nil {
			//check if it is subnet
			aclType = "ip"
		} else {
			if ip := net.ParseIP(ep); ip != nil {
				//check if it is ip
				aclType = "ip"
			}
		}
	}
	return &config.ACE{
		Matches: []*config.ACEMatch{{
			Type:  aclType,
			Value: ep,
		}},
		Dir: config.ACEDirection_BOTH,
		Actions: []*config.ACEAction{{
			Drop: ace.Drop,
		}},
	}
}

// getAcls returns rules for access/deny/forwarding traffic
func (exp *AppExpectation) getAcls(ni *NetInstanceExpectation) []*config.ACE {
	var acls []*config.ACE
	var aclID int32 = 1
	if _, hasAcls := exp.acl[ni.name]; hasAcls {
		// explicitly configured ACLs for the network instance
		for _, acl := range exp.acl[ni.name] {
			ace := parseACE(acl)
			if ace != nil {
				ace.Id = aclID
				acls = append(acls, ace)
				aclID++
			}
		}
	} else {
		// allow access to all addresses
		aclType := "ip"
		aclValue := "0.0.0.0/0"
		if val := exp.acl[""]; len(val) > 0 && val[0].Endpoint == defaults.DefaultHostOnlyNotation {
			// special case for host only acl applied for all NIs without explicit ACLs
			aclType = "host"
			aclValue = ""
		}
		acls = append(acls, &config.ACE{
			Matches: []*config.ACEMatch{{
				Type:  aclType,
				Value: aclValue,
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

// getAccessVID returns Access VLAN ID to assign to the interface between the app
// and the given network instance.
func (exp *AppExpectation) getAccessVID(ni *NetInstanceExpectation) uint32 {
	if exp.vlans == nil {
		return 0
	}
	return uint32(exp.vlans[ni.name])
}
