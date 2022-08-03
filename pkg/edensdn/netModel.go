package edensdn

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net"
	"os"

	model "github.com/lf-edge/eden/sdn/api"
)

// defaultNetModel : default network model.
// More or less corresponds to the static network config used before Eden-SDN was implemented.
var defaultNetModel = model.NetworkModel{
	Ports: []model.Port{
		{
			LogicalLabel: "eth0",
			AdminUP:      true,
		},
		{
			LogicalLabel: "eth1",
			AdminUP:      true,
		},
	},
	Bridges: []model.Bridge{
		{
			LogicalLabel: "bridge0",
			Ports:        []string{"eth0", "eth1"},
		},
	},
	Networks: []model.Network{
		{
			LogicalLabel: "network0",
			Bridge:       "bridge0",
			Subnet:       "172.22.1.0/24",
			GwIP:         "172.22.1.1",
			DHCP: model.DHCP{
				Enable: true,
				IPRange: model.IPRange{
					FromIP: "172.22.1.10",
					ToIP:   "172.22.1.20",
				},
				DomainName: "sdn",
				DNSClientConfig: model.DNSClientConfig{
					PrivateDNS: []string{"dns-server0"},
				},
			},
		},
	},
	Endpoints: model.Endpoints{
		Clients: []model.Client{
			{
				Endpoint: model.Endpoint{
					LogicalLabel: "client0",
					FQDN:         "client0.sdn",
					Subnet:       "10.17.17.0/24",
					IP:           "10.17.17.2",
				},
			},
		},
		DNSServers: []model.DNSServer{
			{
				Endpoint: model.Endpoint{
					LogicalLabel: "dns-server0",
					FQDN:         "dns-server0.sdn",
					Subnet:       "10.18.18.0/24",
					IP:           "10.18.18.2",
				},
				StaticEntries: []model.DNSEntry{
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
func GetDefaultNetModel() model.NetworkModel {
	model := defaultNetModel
	addMissingMACs(&model)
	return model
}

// LoadNetModeFromFile loads network model stored inside a JSON file.
func LoadNetModeFromFile(filepath string) (model.NetworkModel, error) {
	var model model.NetworkModel
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
	return model, nil
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
func addMissingMACs(model *model.NetworkModel) {
	for i, port := range model.Ports {
		if port.MAC == "" {
			model.Ports[i].MAC = generatePortMAC(port.LogicalLabel, true)
		}
		if port.EVEConnect.MAC == "" {
			model.Ports[i].EVEConnect.MAC = generatePortMAC(port.LogicalLabel, false)
		}
	}
}
