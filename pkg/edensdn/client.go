package edensdn

import (
	"bufio"
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

//LinkState of an EVE uplink interface.
type LinkState struct {
	EveIfName string
	IsUP      bool
}

// SdnClient is a client for talking to Eden-SDN management agent.
// It also allows to SSH into SDN VM, establish SSH port forwarding with SDN VM
// and to run command from inside of an endpoint deployed in Eden-SDN.
type SdnClient struct {
	SSHPort    uint16
	SSHKeyPath string
	MgmtPort   uint16
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
	if client.SSHKeyPath == "" {
		log.Fatal("SDN client with undefined SSHKeyPath")
	}
	allArgs := fmt.Sprintf("-o ConnectTimeout=5 -o StrictHostKeyChecking=no "+
		"-i %s -p %d root@localhost", client.SSHKeyPath, client.SSHPort)
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
	args := client.sshArgs("-v", "-T", "-L", fwdArgs, "tail", "-f", "/dev/null")
	cmd := exec.Command("ssh", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	tunnelReadyCh := make(chan bool, 1)
	go func(tunnelReadyCh chan<- bool) {
		var listenerReady, fwdReady, sshReady bool
		fwdMsg := fmt.Sprintf("Local connections to LOCALHOST:%d " +
			"forwarded to remote address %s:%d", localPort, targetIP, targetPort)
		listenMsg := fmt.Sprintf("Local forwarding listening on 127.0.0.1 port %d",
			localPort)
		sshReadyMsg := "Entering interactive session"
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, fwdMsg) {
				fwdReady = true
			}
			if strings.Contains(line, listenMsg) {
				listenerReady = true
			}
			if strings.Contains(line, sshReadyMsg) {
				sshReady = true
			}
			if listenerReady && fwdReady && sshReady {
				tunnelReadyCh <- true
				return
			}
		}
	}(tunnelReadyCh)
	close = func() {
		err = cmd.Process.Kill()
		if err != nil {
			log.Errorf("failed to kill %s: %v", cmd, err)
		} else {
			_ = cmd.Wait()
		}
	}
	// Give tunnel some time to open.
	select {
	case <-tunnelReadyCh:
		// Just an extra cushion for the tunnel to establish.
		time.Sleep(500 * time.Millisecond)
		return close, nil
	case <-time.After(10 * time.Second):
		close()
		return nil, fmt.Errorf("failed to create SSH tunnel %s in time", fwdArgs)
	}
}

// RunCmdFromEndpoint : execute command from inside of an endpoint deployed in Eden-SDN.
func (client *SdnClient) RunCmdFromEndpoint(epLogicalLabel, cmd string, args ...string) error {
	ipNetns := []string{"ip", "netns", "exec", "endpoint-" + epLogicalLabel}
	ipNetns = append(ipNetns, cmd)
	ipNetns = append(ipNetns, args...)
	return utils.RunCommandForeground("ssh", client.sshArgs(ipNetns...)...)
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
