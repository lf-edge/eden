package openevec

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/edensdn"
	"github.com/lf-edge/eden/pkg/utils"
	sdnapi "github.com/lf-edge/eden/sdn/vm/api"
	log "github.com/sirupsen/logrus"
)

func (openEVEC *OpenEVEC) SdnForwardSSHToEve(commandToRun string) error {
	cfg := openEVEC.cfg
	arguments := fmt.Sprintf("-o IdentitiesOnly=yes -o ConnectTimeout=5 -o StrictHostKeyChecking=no -i %s "+
		"-p FWD_PORT root@FWD_IP %s", sdnSSSHKeyPrivate(cfg.Eden.SSHKey), commandToRun)
	return openEVEC.SdnForwardCmd("", "eth0", 22, "ssh", strings.Fields(arguments)...)
}

func (openEVEC *OpenEVEC) SdnForwardSCPFromEve(remoteFilePath, localFilePath string) error {
	cfg := openEVEC.cfg
	arguments := fmt.Sprintf("-o IdentitiesOnly=yes -o ConnectTimeout=5 -o StrictHostKeyChecking=no -i %s "+
		"-P FWD_PORT root@FWD_IP:%s %s", sdnSSSHKeyPrivate(cfg.Eden.SSHKey), remoteFilePath, localFilePath)
	return openEVEC.SdnForwardCmd("", "eth0", 22, "scp", strings.Fields(arguments)...)
}

func sdnSSSHKeyPrivate(sshKeyPub string) string {
	extension := filepath.Ext(sshKeyPub)
	// we store the pub key in config
	if extension == ".pub" {
		return strings.TrimRight(sshKeyPub, extension)
	}
	return sshKeyPub
}

func sdnSSHKeyPath(sdnSourceDir string) string {
	return filepath.Join(sdnSourceDir, "vm/cert/ssh/id_rsa")
}

func (openEVEC *OpenEVEC) SdnForwardCmd(fromEp string, eveIfName string, targetPort int, cmd string, args ...string) error {
	cfg := openEVEC.cfg
	const fwdIPLabel = "FWD_IP"
	const fwdPortLabel = "FWD_PORT"

	// Case 1: EVE is running remotely (on Raspberry Pi, Gcp, etc.)
	if cfg.Eve.Remote {
		// Get IP address used by the target EVE interface.
		// (look at network info published by EVE)
		ip := openEVEC.GetEveIP(eveIfName)
		if ip == "" {
			return fmt.Errorf("failed to obtain IP address for EVE interface %s", eveIfName)
		}
		for i := range args {
			args[i] = strings.ReplaceAll(args[i], fwdIPLabel, ip)
			args[i] = strings.ReplaceAll(args[i], fwdPortLabel, strconv.Itoa(targetPort))
		}
		err := utils.RunCommandForeground(cmd, args...)
		if err != nil {
			return fmt.Errorf("command %s failed: %w", cmd, err)
		}
		return nil
	}

	// Case 2: EVE is running inside VM on this host, but without SDN in between
	if !cfg.IsSdnEnabled() {
		if fromEp != "" {
			log.Warnf("Cannot execute command from an endpoint without SDN running, " +
				"argument \"from-ep\" will be ignored")
		}
		// Network model is static and consists of two EVE interfaces.
		if eveIfName != "eth0" && eveIfName != "eth1" {
			return fmt.Errorf("unknown EVE interface: %s", eveIfName)
		}
		// Find out what the targetPort is (statically) mapped to in the host.
		targetHostPort := -1
		for k, v := range cfg.Eve.HostFwd {
			hostPort, err := strconv.Atoi(k)
			if err != nil {
				log.Errorf("failed to parse host port from eve.hostfwd: %s", err.Error())
				continue
			}
			guestPort, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("failed to parse guest port from eve.hostfwd: %s", err.Error())
				continue
			}
			if eveIfName == "eth1" {
				// For eth1 numbers of forwarded ports are shifted by 10.
				hostPort += 10
				guestPort += 10
			}
			if guestPort == targetPort {
				targetHostPort = hostPort
				break
			}
		}
		if targetHostPort == -1 {
			return fmt.Errorf("target EVE interface and port (%s, %d) are not port-forwarded "+
				"by config (see eve.hostfwd)", eveIfName, targetPort)
		}
		// Redirect command to localhost and the forwarded port.
		fwdPort := strconv.Itoa(targetHostPort)
		for i := range args {
			args[i] = strings.ReplaceAll(args[i], fwdIPLabel, "127.0.0.1")
			args[i] = strings.ReplaceAll(args[i], fwdPortLabel, fwdPort)
		}
		err := utils.RunCommandForeground(cmd, args...)
		if err != nil {
			return fmt.Errorf("command %s failed: %w", cmd, err)
		}
		return nil
	}

	// Case 3: EVE is running inside VM on this host, with networking provided by SDN VM
	// TODO: Port forwarding with SDN only works for TCP for now.

	// Get IP address used by the target EVE interface.
	// (look at the ARP tables inside SDN VM)
	targetIP := openEVEC.GetEveIP(eveIfName)
	if targetIP == "" {
		return fmt.Errorf("no IP address found to be assigned to EVE interface %s",
			eveIfName)
	}
	client := &edensdn.SdnClient{
		SSHPort:    uint16(cfg.Sdn.SSHPort),
		SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
		MgmtPort:   uint16(cfg.Sdn.MgmtPort),
	}
	if fromEp != "" {
		// Running command from an endpoint inside SDN VM, no tunneling is needed.
		fwdPort := strconv.Itoa(targetPort)
		for i := range args {
			args[i] = strings.ReplaceAll(args[i], fwdIPLabel, targetIP)
			args[i] = strings.ReplaceAll(args[i], fwdPortLabel, fwdPort)
		}
		err := client.RunCmdFromEndpoint(fromEp, cmd, args...)
		if err != nil {
			return fmt.Errorf("command %s %s run inside endpoint %s failed: %w",
				cmd, strings.Join(args, " "), fromEp, err)
		}
		return nil
	}
	// Temporarily establish port forwarding using SSH.
	localPort, err := utils.FindUnusedPort()
	if err != nil {
		return fmt.Errorf("failed to find unused port number: %w", err)
	}
	closeTunnel, err := client.SSHPortForwarding(localPort, uint16(targetPort), targetIP)
	if err != nil {
		return fmt.Errorf("failed to establish SSH port forwarding: %w", err)
	}
	defer closeTunnel()
	// Redirect command to localhost and the forwarded port.
	fwdPort := strconv.Itoa(int(localPort))
	for i := range args {
		args[i] = strings.ReplaceAll(args[i], fwdIPLabel, "127.0.0.1")
		args[i] = strings.ReplaceAll(args[i], fwdPortLabel, fwdPort)
	}
	err = utils.RunCommandForeground(cmd, args...)
	if err != nil {
		return fmt.Errorf("command %s %s failed: %w", cmd, strings.Join(args, " "), err)
	}
	return nil
}

func (openEVEC *OpenEVEC) SdnStatus() error {
	cfg := openEVEC.cfg
	if !cfg.IsSdnEnabled() {
		return fmt.Errorf("Sdn is not enabled")
	}
	processStatus, err := utils.StatusCommandWithPid(cfg.Sdn.PidFile)
	if err != nil {
		log.Errorf("%s cannot obtain status of SDN Qemu process: %s",
			statusWarn(), err)
	} else {
		fmt.Printf("%s SDN on Qemu status: %s\n",
			representProcessStatus(processStatus), processStatus)
		fmt.Printf("\tConsole logs for SDN at: %s\n", cfg.Sdn.ConsoleLogFile)
	}
	client := &edensdn.SdnClient{
		SSHPort:    uint16(cfg.Sdn.SSHPort),
		SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
		MgmtPort:   uint16(cfg.Sdn.MgmtPort),
	}
	status, err := client.GetSdnStatus()
	if err != nil {
		return fmt.Errorf("failed to get SDN status: %w", err)
	}
	if len(status.ConfigErrors) == 0 {
		fmt.Printf("\tNo configuration errors.\n")
	} else {
		fmt.Printf("\tHave configuration errors: %v\n", status.ConfigErrors)
	}
	fmt.Printf("\tManagement IPs: %v\n", strings.Join(status.MgmtIPs, ", "))
	return nil
}

func (openEVEC *OpenEVEC) SdnNetModelGet() (string, error) {
	cfg := openEVEC.cfg
	if !cfg.IsSdnEnabled() {
		return "", fmt.Errorf("SDN is not enabled")
	}
	client := &edensdn.SdnClient{
		SSHPort:    uint16(cfg.Sdn.SSHPort),
		SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
		MgmtPort:   uint16(cfg.Sdn.MgmtPort),
	}
	netModel, err := client.GetNetworkModel()
	if err != nil {
		return "", fmt.Errorf("failed to get network model: %w", err)
	}
	jsonBytes, err := json.MarshalIndent(netModel, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal net modem to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

func (openEVEC *OpenEVEC) SdnNetModelApply(ref string) error {
	cfg := openEVEC.cfg
	if !cfg.IsSdnEnabled() {
		return fmt.Errorf("SDN is not enabled")
	}
	var err error
	var newNetModel sdnapi.NetworkModel
	if ref == "default" {
		newNetModel, err = edensdn.GetDefaultNetModel()
		if err != nil {
			return err
		}
	} else {
		newNetModel, err = edensdn.LoadNetModeFromFile(ref)
		if err != nil {
			return fmt.Errorf("failed to load network model from file '%s': %w", ref, err)
		}
	}
	newNetModel.Host.ControllerPort = uint16(cfg.Adam.Port)
	client := &edensdn.SdnClient{
		SSHPort:    uint16(cfg.Sdn.SSHPort),
		SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
		MgmtPort:   uint16(cfg.Sdn.MgmtPort),
	}
	oldNetModel, err := client.GetNetworkModel()
	if err != nil {
		return fmt.Errorf("failed to get current network model: %w", err)
	}
	vmRunner, err := edensdn.GetSdnVMRunner(cfg.Eve.DevModel, edensdn.SdnVMConfig{})
	if err != nil {
		return fmt.Errorf("failed to get SDN VM runner: %w", err)
	}
	if vmRunner.RequiresVmRestart(oldNetModel, newNetModel) {
		if ref != "default" && !filepath.IsAbs(ref) {
			ref = "$(pwd)/" + ref
		}
		return fmt.Errorf("Network model change requires to restart SDN and EVE VMs.\n" +
			"Run instead:\n" +
			"  eden eve stop\n" +
			"  eden eve start --sdn-network-model " + ref + "\n")
	}
	err = client.ApplyNetworkModel(newNetModel)
	if err != nil {
		return fmt.Errorf("failed to apply network model: %w", err)
	}
	fmt.Printf("Submitted network model: %s", ref)
	return nil
}

func (openEVEC *OpenEVEC) SdnNetConfigGraph() (string, error) {
	cfg := openEVEC.cfg
	if !cfg.IsSdnEnabled() {
		return "", fmt.Errorf("SDN is not enabled")
	}
	client := &edensdn.SdnClient{
		SSHPort:    uint16(cfg.Sdn.SSHPort),
		SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
		MgmtPort:   uint16(cfg.Sdn.MgmtPort),
	}
	netConfig, err := client.GetNetworkConfigGraph()
	if err != nil {
		return "", fmt.Errorf("failed to get network config: %w", err)
	}
	return netConfig, nil
}

func (openEVEC *OpenEVEC) SdnSsh() error {
	cfg := openEVEC.cfg
	if !cfg.IsSdnEnabled() {
		return fmt.Errorf("SDN is not enabled")
	}
	client := &edensdn.SdnClient{
		SSHPort:    uint16(cfg.Sdn.SSHPort),
		SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
		MgmtPort:   uint16(cfg.Sdn.MgmtPort),
	}
	if err := client.SSHIntoSdnVM(); err != nil {
		return fmt.Errorf("failed to SSH into SDN VM: %w", err)
	}
	return nil
}

func (openEVEC *OpenEVEC) SdnLogs() (string, error) {
	cfg := openEVEC.cfg
	if !cfg.IsSdnEnabled() {
		return "", fmt.Errorf("SDN is not enabled")
	}
	client := &edensdn.SdnClient{
		SSHPort:    uint16(cfg.Sdn.SSHPort),
		SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
		MgmtPort:   uint16(cfg.Sdn.MgmtPort),
	}
	sdnLogs, err := client.GetSdnLogs()
	if err != nil {
		return "", fmt.Errorf("failed to get SDN logs: %w", err)
	}
	return sdnLogs, nil
}

func (openEVEC *OpenEVEC) SdnMgmtIp() (string, error) {
	cfg := openEVEC.cfg
	if !cfg.IsSdnEnabled() {
		return "", fmt.Errorf("SDN is not enabled")
	}
	client := &edensdn.SdnClient{
		SSHPort:    uint16(cfg.Sdn.SSHPort),
		SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
		MgmtPort:   uint16(cfg.Sdn.MgmtPort),
	}
	status, err := client.GetSdnStatus()
	if err != nil {
		return "", fmt.Errorf("failed to get SDN status: %w", err)
	}
	if len(status.MgmtIPs) == 0 {
		return "", fmt.Errorf("no management IP reported by SDN: %w", err)
	}
	return status.MgmtIPs[0], nil
}

func (openEVEC *OpenEVEC) SdnEpExec(epName, command string, args []string) error {
	cfg := openEVEC.cfg
	if !cfg.IsSdnEnabled() {
		return fmt.Errorf("SDN is not enabled")
	}
	client := &edensdn.SdnClient{
		SSHPort:    uint16(cfg.Sdn.SSHPort),
		SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
		MgmtPort:   uint16(cfg.Sdn.MgmtPort),
	}
	err := client.RunCmdFromEndpoint(epName, command, args...)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}
	return nil
}
