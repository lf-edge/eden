package edensdn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/utils"
	model "github.com/lf-edge/eden/sdn/api"
	log "github.com/sirupsen/logrus"
)

const sshKeyPath = "./sdn/cert/ssh/id_rsa"

//LinkState of an EVE uplink interface.
type LinkState struct {
	EveIfName string
	IsUP      bool
}

// SdnClient is a client for talking to Eden-SDN management agent.
// It also allows to SSH into SDN VM, establish SSH port forwarding with SDN VM
// and to run command from inside of an endpoint deployed in Eden-SDN.
type SdnClient struct {
	SSHPort  uint16
	MgmtPort uint16
}

// GetNetworkModel : get network model currently applied to Eden-SDN.
func (client *SdnClient) GetNetworkModel() (netModel model.NetworkModel, err error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("http://localhost:%d/net-model.json", client.MgmtPort), nil)
	if err != nil {
		err = fmt.Errorf("failed to build HTTP request: %w", err)
		return
	}
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("request to GET network model failed: %w", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request to GET network model failed with resp: %s",
			resp.Status)
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read retrieved network model data: %w", err)
		return
	}
	err = json.Unmarshal(data, &netModel)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal retrieved network model data: %w", err)
		return
	}
	return
}

// ApplyNetworkModel : submit network model to Eden-SDN.
func (client *SdnClient) ApplyNetworkModel(netModel model.NetworkModel) (err error) {
	json, err := json.Marshal(netModel)
	if err != nil {
		err = fmt.Errorf("failed to marshal network model: %w", err)
		return
	}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("http://localhost:%d/net-model.json", client.MgmtPort),
		bytes.NewBuffer(json))
	if err != nil {
		err = fmt.Errorf("failed to build HTTP request: %w", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("request to PUT network model failed: %w", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var respBytes []byte
		var response string
		respBytes, err = io.ReadAll(resp.Body)
		if err == nil {
			response = string(respBytes)
		} else {
			response = fmt.Sprintf("failed to read response: %v", err)
		}
		err = fmt.Errorf("request to PUT network model failed with code=%d, " +
			"response: %s",	resp.StatusCode, response)
		return
	}
	return
}

// GetNetworkConfigGraph : get network config applied by Eden-SDN.
// Network config items and their dependencies are depicted using a DOT graph.
func (client *SdnClient) GetNetworkConfigGraph() (config string, err error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("http://localhost:%d/net-config.gv", client.MgmtPort), nil)
	if err != nil {
		err = fmt.Errorf("failed to build HTTP request: %w", err)
		return
	}
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("request to GET network config failed: %w", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request to GET network config failed with resp: %s",
			resp.Status)
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read retrieved network config: %w", err)
		return
	}
	config = string(data)
	return
}

// GetSdnStatus : get status of the running Eden-SDN.
func (client *SdnClient) GetSdnStatus() (status model.SDNStatus, err error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("http://localhost:%d/sdn-status.json", client.MgmtPort), nil)
	if err != nil {
		err = fmt.Errorf("failed to build HTTP request: %w", err)
		return
	}
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("request to GET SDN status failed: %w", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request to GET SDN status failed with resp: %s",
			resp.Status)
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read retrieved SDN status data: %w", err)
		return
	}
	err = json.Unmarshal(data, &status)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal retrieved SDN status data: %w", err)
		return
	}
	return
}

func (client *SdnClient) sshArgs(extra ...string) (sshArgs []string) {
	allArgs := fmt.Sprintf("-o ConnectTimeout=5 -o StrictHostKeyChecking=no "+
		"-i %s -p %d root@localhost", sshKeyPath, client.SSHPort)
	return append(strings.Fields(allArgs), extra...)
}

// GetSdnLogs : get all logs from running Eden-SDN VM.
func (client *SdnClient) GetSdnLogs() (string, error) {
	command := exec.Command("ssh", client.sshArgs("cat", "/run/sdn.log")...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command %s failed: %v", command, string(output))
	}
	return string(output), err
}

// SSHIntoSdnVM : ssh into the running Eden-SDN.
func (client *SdnClient) SSHIntoSdnVM() error {
	return utils.RunCommandForeground("ssh", client.sshArgs()...)
}

// SSHPortFowarding : establish port forwarding between the host and the SDN VM using ssh.
// Close the tunnel by running returned "close" function.
func (client *SdnClient) SSHPortForwarding(localPort, targetPort uint16,
	targetIP string) (close func(), err error) {
	fwdArgs := fmt.Sprintf("%d:%s:%d", localPort, targetIP, targetPort)
	cmd := exec.Command("ssh",
		client.sshArgs("-T", "-L", fwdArgs, "tail", "-f", "/dev/null")...)
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	close = func() {
		err = cmd.Process.Kill()
		if err != nil {
			log.Errorf("failed to kill %s: %v", cmd, err)
		} else {
			_ = cmd.Wait()
		}
	}
	// Give tunnel some time to open.
	// TODO: how to determine when the tunnel is ready and avoid sleeping with hard-coded time?
	time.Sleep(2 * time.Second)
	return close, nil
}

// RunCmdFromEndpoint : execute command from inside of an endpoint deployed in Eden-SDN.
func (client *SdnClient) RunCmdFromEndpoint(epLogicalLabel, cmd string, args ...string) error {
	ipNetns := []string{"ip", "netns", "exec", "endpoint-" + epLogicalLabel}
	ipNetns = append(ipNetns, cmd)
	ipNetns = append(ipNetns, args...)
	return utils.RunCommandForeground("ssh", client.sshArgs(ipNetns...)...)
}

// SetLinkState : set the link state of an EVE interface by bringing the other (SDN)
// side UP or DOWN.
func (client *SdnClient) SetLinkState(eveIfName string, up bool) (err error) {
	netModel, err := client.GetNetworkModel()
	if err != nil {
		err = fmt.Errorf("failed to get network model: %v", err)
		return
	}
	var eveIfIndex int
	eveIfIndex, err = client.getEveIfIndex(eveIfName)
	if err != nil {
		return
	}
	if eveIfIndex < 0 || eveIfIndex >= len(netModel.Ports) {
		err = fmt.Errorf("EVE interface index is out-of-range: %d <%d-%d)",
			eveIfIndex, 0, len(netModel.Ports))
		return
	}
	netModel.Ports[eveIfIndex].AdminUP = up
	err = client.ApplyNetworkModel(netModel)
	if err != nil {
		err = fmt.Errorf("failed to apply updated network model: %v", err)
		return
	}
	return nil
}

// GetLinkState : get link state of an EVE interface.
// If the interface name is not specified (empty string), then the link state of every
// EVE interface is returned.
// This is determined by the admin state of the interface on the other (SDN) side.
func (client *SdnClient) GetLinkState(eveIfName string) (linkStates []LinkState, err error) {
	eveIfIndex := -1
	if eveIfName != "" {
		eveIfIndex, err = client.getEveIfIndex(eveIfName)
		if err != nil {
			return
		}
	}
	netModel, err := client.GetNetworkModel()
	if err != nil {
		err = fmt.Errorf("failed to get network model: %v", err)
		return
	}
	for i, port := range netModel.Ports {
		if eveIfIndex == -1 || eveIfIndex == i {
			linkStates = append(linkStates, LinkState{
				EveIfName: fmt.Sprintf("eth%d", i),
				IsUP:      port.AdminUP,
			})
		}
	}
	return
}

// GetEveIfMAC : get MAC address assigned to the given EVE interface.
func (client *SdnClient) GetEveIfMAC(eveIfName string) (mac string, err error) {
	netModel, err := client.GetNetworkModel()
	if err != nil {
		err = fmt.Errorf("failed to get network model: %v", err)
		return
	}
	var eveIfIndex int
	eveIfIndex, err = client.getEveIfIndex(eveIfName)
	if err != nil {
		return
	}
	if eveIfIndex < 0 || eveIfIndex >= len(netModel.Ports) {
		err = fmt.Errorf("EVE interface index is out-of-range: %d <%d-%d)",
			eveIfIndex, 0, len(netModel.Ports))
		return
	}
	return netModel.Ports[eveIfIndex].EVEConnect.MAC, nil
}

// GetEveIfIP : get IP address assigned to the given EVE interface.
func (client *SdnClient) GetEveIfIP(eveIfName string) (ip string, err error) {
	mac, err := client.GetEveIfMAC(eveIfName)
	if err != nil {
		return "", fmt.Errorf("failed to get MAC address for EVE interface %s: %v",
			eveIfName, err)
	}
	command := exec.Command("ssh", client.sshArgs("/bin/get-eve-ip.sh", mac)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("get-eve-ip.sh failed: %v", string(output))
	}
	ip = strings.TrimSpace(string(output))
	return ip, nil
}

func (client *SdnClient) getEveIfIndex(eveIfName string) (eveIfIndex int, err error) {
	if !strings.HasPrefix(eveIfName, "eth") {
		err = fmt.Errorf("unexpected EVE interface name: %s", eveIfName)
		return
	}
	eveIfIndex, err = strconv.Atoi(strings.TrimPrefix(eveIfName, "eth"))
	if err != nil {
		err = fmt.Errorf("failed to parse EVE interface index: %v", err)
	}
	return
}
