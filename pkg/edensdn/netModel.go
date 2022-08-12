package edensdn

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net"
	"os"

	"github.com/lf-edge/eden/pkg/utils"
	sdnapi "github.com/lf-edge/eden/sdn/api"
)

// defaultNetModel : default network model.
// More or less corresponds to the static network config used before Eden-SDN was implemented.
var defaultNetModel = sdnapi.NetworkModel{
	Ports: []sdnapi.Port{
		{
			LogicalLabel: "eth0",
			AdminUP:      true,
		},
		{
			LogicalLabel: "eth1",
			AdminUP:      true,
		},
	},
	Bridges: []sdnapi.Bridge{
		{
			LogicalLabel: "bridge0",
			Ports:        []string{"eth0", "eth1"},
		},
	},
	Networks: []sdnapi.Network{
		{
			LogicalLabel: "network0",
			Bridge:       "bridge0",
			Subnet:       "172.22.1.0/24",
			GwIP:         "172.22.1.1",
			DHCP: sdnapi.DHCP{
				Enable: true,
				IPRange: sdnapi.IPRange{
					FromIP: "172.22.1.10",
					ToIP:   "172.22.1.20",
				},
				DomainName: "sdn",
				DNSClientConfig: sdnapi.DNSClientConfig{
					PrivateDNS: []string{"dns-server0"},
				},
			},
		},
	},
	Endpoints: sdnapi.Endpoints{
		Clients: []sdnapi.Client{
			{
				Endpoint: sdnapi.Endpoint{
					LogicalLabel: "client0",
					FQDN:         "client0.sdn",
					Subnet:       "10.17.17.0/24",
					IP:           "10.17.17.2",
				},
			},
		},
		DNSServers: []sdnapi.DNSServer{
			{
				Endpoint: sdnapi.Endpoint{
					LogicalLabel: "dns-server0",
					FQDN:         "dns-server0.sdn",
					Subnet:       "10.18.18.0/24",
					IP:           "10.18.18.2",
				},
				StaticEntries: []sdnapi.DNSEntry{
					{
						// See config item "adam.domain".
						FQDN: "mydomain.adam",
						IP:   "adam-ip",
					},
				},
				UpstreamServers: []string{
					"8.8.8.8",
					"1.1.1.1",
				},
			},
		},
	},
}

// GetDefaultNetModel : get default network model.
// Used unless the user selects custom network model.
func GetDefaultNetModel() (model sdnapi.NetworkModel, err error) {
	model = defaultNetModel
	addMissingMACs(&model)
	err = addMissingHostConfig(&model)
	return
}

// LoadNetModeFromFile loads network model stored inside a JSON file.
func LoadNetModeFromFile(filepath string) (sdnapi.NetworkModel, error) {
	var model sdnapi.NetworkModel
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		err = fmt.Errorf("failed to read net model from file '%s': %w",
			filepath, err)
		return model, err
	}
	err = json.Unmarshal(content, &model)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal net model from file '%s': %w",
			filepath, err)
		return model, err
	}
	addMissingMACs(&model)
	err = addMissingHostConfig(&model)
	return model, err
}

// GenerateSdnMgmtMAC (deterministically) generates MAC address for interface
// connecting Eden-SDN with the Host.
func GenerateSdnMgmtMAC() string {
	hostname, _ := os.Hostname()
	h := fnv.New32a()
	h.Write([]byte(hostname))
	hash := h.Sum32()
	hwAddr := make(net.HardwareAddr, 6)
	// 08:33:33 is prefix expected by SDN to be used for the management interface
	// See hostPortMACPrefix in ./sdn/cmd/sdnagent/agent.go
	hwAddr[0] = 0x08
	hwAddr[1] = 0x33
	hwAddr[2] = 0x33
	for i := 0; i < 3; i++ {
		hwAddr[i+3] = byte(hash & 0xff)
		hash >>= 8
	}
	return hwAddr.String()
}

// generatePortMAC (deterministically) generates MAC address for a given port.
// Used when MAC address is not specified inside the network model.
func generatePortMAC(logicalLabel string, sdnSide bool) string {
	h := fnv.New32a()
	h.Write([]byte(logicalLabel))
	hash := h.Sum32()
	hwAddr := make(net.HardwareAddr, 6)
	hwAddr[0] = 0x02
	if sdnSide {
		hwAddr[1] = 0xfd
	} else {
		hwAddr[1] = 0xfe
	}
	for i := 0; i < 4; i++ {
		hwAddr[i+2] = byte(hash & 0xff)
		hash >>= 8
	}
	return hwAddr.String()
}

// addMissingMACs generates and inserts MAC addresses into the model for ports
// which were defined without MAC address included.
func addMissingMACs(model *sdnapi.NetworkModel) {
	for i, port := range model.Ports {
		if port.MAC == "" {
			model.Ports[i].MAC = generatePortMAC(port.LogicalLabel, true)
		}
		if port.EVEConnect.MAC == "" {
			model.Ports[i].EVEConnect.MAC = generatePortMAC(port.LogicalLabel, false)
		}
	}
}

func addMissingHostConfig(netModel *sdnapi.NetworkModel) error {
	if netModel.Host == nil {
		hostIP, err := utils.GetIPForDockerAccess()
		if err != nil {
			return fmt.Errorf("failed to find suitable host IP: %v", err)
		}
		netModel.Host = &sdnapi.HostConfig{
			HostIPs:     []string{hostIP},
			NetworkType: sdnapi.Ipv4Only, // XXX For now everything is IPv4 only
		}
	}
	return nil
}
