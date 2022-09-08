package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/edensdn"
	"github.com/lf-edge/eden/pkg/utils"
	sdnapi "github.com/lf-edge/eden/sdn/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	sdnFwdFromEp string
)

var sdnCmd = &cobra.Command{
	Use:   "sdn",
	Short: "Emulate and manage networks surrounding EVE VM using Eden-SDN",
}

var sdnNetModelCmd = &cobra.Command{
	Use:   "net-model",
	Short: "Manage network model submitted to Eden-SDN",
}

var sdnNetModelGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get currently applied network model (in JSON)",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !isSdnEnabled() {
			log.Fatalf("SDN is not enabled")
		}
		client := &edensdn.SdnClient{
			SSHPort:  uint16(sdnSSHPort),
			MgmtPort: uint16(sdnMgmtPort),
		}
		netModel, err := client.GetNetworkModel()
		if err != nil {
			log.Fatalf("Failed to get network model: %v", err)
		}
		jsonBytes, err := json.MarshalIndent(netModel, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal net modem to JSON: %v", err)
		}
		fmt.Println(string(jsonBytes))
	},
}

var sdnNetModelApplyCmd = &cobra.Command{
	Use:   "apply <filepath.json|default>",
	Short: "submit network model into Eden-SDN",
	Long: `Load network model from a JSON file and submit it to Eden-SDN.
Use string \"default\" instead of a file path to apply the default network model
(two eth interfaces inside the same network with DHCP, see DefaultNetModel in pkg/edensdn/netModel.go).`,
	Args: cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !isSdnEnabled() {
			log.Fatalf("SDN is not enabled")
		}
		ref := args[0]
		var err error
		var newNetModel sdnapi.NetworkModel
		if ref == "default" {
			newNetModel, err = edensdn.GetDefaultNetModel()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			newNetModel, err = edensdn.LoadNetModeFromFile(ref)
			if err != nil {
				log.Fatalf("Failed to load network model from file '%s': %v", ref, err)
			}
		}
		client := &edensdn.SdnClient{
			SSHPort:  uint16(sdnSSHPort),
			MgmtPort: uint16(sdnMgmtPort),
		}
		oldNetModel, err := client.GetNetworkModel()
		if err != nil {
			log.Fatalf("Failed to get current network model: %v", err)
		}
		vmRunner, err := edensdn.GetSdnVMRunner(devModel, edensdn.SdnVMConfig{})
		if err != nil {
			log.Fatalf("Failed to get SDN VM runner: %v", err)
		}
		if vmRunner.RequiresVmRestart(oldNetModel, newNetModel) {
			log.Fatalf("Network model change requires to restart SDN and EVE VMs.\n" +
				"Run instead:\n" +
				"  eden eve stop\n" +
				"  eden eve start --sdn-network-model " + ref + "\n")
		}
		err = client.ApplyNetworkModel(newNetModel)
		if err != nil {
			log.Fatalf("Failed to apply network model: %v", err)
		}
		fmt.Printf("Submitted network model: %s", ref)
	},
}

var sdnNetConfigGraphCmd = &cobra.Command{
	Use:   "net-config-graph",
	Short: "get network config applied by Eden-SDN, visualized using a dependency graph",
	Long: `Get network config applied by Eden-SDN.
Network config items and their dependencies are depicted using a DOT graph.
To generate graph image, run: eden sdn net-config-graph | dot -Tsvg > output.svg
This requires to have Graphviz installed.
Alternatively, visualize using the online tool: https://dreampuf.github.io/GraphvizOnline/`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !isSdnEnabled() {
			log.Fatalf("SDN is not enabled")
		}
		client := &edensdn.SdnClient{
			SSHPort:  uint16(sdnSSHPort),
			MgmtPort: uint16(sdnMgmtPort),
		}
		netConfig, err := client.GetNetworkConfigGraph()
		if err != nil {
			log.Fatalf("Failed to get network config: %v", err)
		}
		fmt.Println(netConfig)
	},
}

var sdnStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get status of the running Eden-SDN",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !isSdnEnabled() {
			log.Fatalf("SDN is not enabled")
		}
		sdnStatus()
	},
}

func sdnStatus() {
	if !isSdnEnabled() {
		return
	}
	processStatus, err := utils.StatusCommandWithPid(sdnPidFile)
	if err != nil {
		log.Errorf("%s cannot obtain status of SDN Qemu process: %s",
			statusWarn(), err)
	} else {
		fmt.Printf("%s SDN on Qemu status: %s\n",
			representProcessStatus(processStatus), processStatus)
		fmt.Printf("\tConsole logs for SDN at: %s\n", sdnConsoleLogFile)
	}
	client := &edensdn.SdnClient{
		SSHPort:  uint16(sdnSSHPort),
		MgmtPort: uint16(sdnMgmtPort),
	}
	status, err := client.GetSdnStatus()
	if err != nil {
		log.Fatalf("Failed to get SDN status: %v", err)
	}
	if len(status.ConfigErrors) == 0 {
		fmt.Printf("\tNo configuration errors.\n")
	} else {
		fmt.Printf("\tHave configuration errors: %v\n", status.ConfigErrors)
	}
	fmt.Printf("\tManagement IPs: %v\n", strings.Join(status.MgmtIPs, ", "))
}

var sdnSshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "SSH into the running Eden-SDN VM",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !isSdnEnabled() {
			log.Fatalf("SDN is not enabled")
		}
		client := &edensdn.SdnClient{
			SSHPort:  uint16(sdnSSHPort),
			MgmtPort: uint16(sdnMgmtPort),
		}
		err := client.SSHIntoSdnVM()
		if err != nil {
			log.Fatalf("Failed to SSH into SDN VM: %v", err)
		}
	},
}

var sdnLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Get all logs from running Eden-SDN VM",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !isSdnEnabled() {
			log.Fatalf("SDN is not enabled")
		}
		client := &edensdn.SdnClient{
			SSHPort:  uint16(sdnSSHPort),
			MgmtPort: uint16(sdnMgmtPort),
		}
		sdnLogs, err := client.GetSdnLogs()
		if err != nil {
			log.Fatalf("Failed to get SDN logs: %v", err)
		}
		fmt.Println(sdnLogs)
	},
}

var sdnMgmtIpCmd = &cobra.Command{
	Use:   "mgmt-ip",
	Short: "Get IP address assigned to Eden-SDN VM for management",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !isSdnEnabled() {
			log.Fatalf("SDN is not enabled")
		}
		client := &edensdn.SdnClient{
			SSHPort:  uint16(sdnSSHPort),
			MgmtPort: uint16(sdnMgmtPort),
		}
		status, err := client.GetSdnStatus()
		if err != nil {
			log.Fatalf("Failed to get SDN status: %v", err)
		}
		if len(status.MgmtIPs) == 0 {
			log.Fatalf("No management IP reported by SDN: %v", err)
		}
		fmt.Printf(status.MgmtIPs[0])
	},
}

var sdnEndpointCmd = &cobra.Command{
	Use:   "endpoint",
	Short: "Interact with endpoints deployed inside Eden-SDN",
	Long: `Interact with endpoints deployed inside Eden-SDN.
Endpoints emulate "remote" clients and servers.
Used to test connectivity between apps running on EVE and remote hosts.
Endpoint can be for example:
	- HTTP client (accessing server running as app inside EVE),
	- HTTP server (being accessed by app inside EVE),
	- DNS server (used by EVE/app),
	- NTP server (used by EVE/app),
	- etc.
See sdn/api/endpoints.go to learn about all kinds of supported endpoints.`,
}

var sdnEpExecCmd = &cobra.Command{
	Use:   "exec <endpoint-name> -- <command> [args...]",
	Short: "Execute command from inside of the given endpoint",
	Long: `Execute command from inside of the given endpoint.
Remember that the command is running inside the Eden-SDN VM and the net namespace of the endpoint.
References to files present only on the host are therefore not valid!
Command must be installed in Eden-SDN!
Also consider using "eden sdn fwd" command instead, especially if you are intending to use
the EVE's port forwarding capability.`,
	Args: cobra.MinimumNArgs(2),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !isSdnEnabled() {
			log.Fatalf("SDN is not enabled")
		}
		epName := args[0]
		command := args[1]
		args = args[2:]
		client := &edensdn.SdnClient{
			SSHPort:  uint16(sdnSSHPort),
			MgmtPort: uint16(sdnMgmtPort),
		}
		err := client.RunCmdFromEndpoint(epName, command, args...)
		if err != nil {
			log.Fatalf("Failed to execute command: %v", err)
		}
	},
}

var sdnFwdCmd = &cobra.Command{
	Use:   "fwd <target-eve-interface> <target-port> -- <command> [args...]",
	Short: "Execute command aimed at a given EVE interface and a port",
	Long: `Execute command aimed at a given EVE interface and a port.
Use for example to ssh into EVE, or to access HTTP server running as EVE app, etc.
Note that you cannot run commands against EVE interfaces from the host directly - there is
Eden-SDN VM in the way. Even without SDN (i.e. legacy mode with SLIRP networking on QEMU),
EVE interfaces and ports are not directly accessible from the host but must be port forwarded by QEMU.
You cannot therefore reference the target EVE IP address and port number directly, they could
be mapped to different values on the host (and forwarding may need to be established beforehand).
The command should therefore reference destination IP address and port number symbolically
with labels FWD_IP and FWD_PORT, and let Eden to establish forwarding, replace symbolic names
with an actual IP address and a port number, and run command from the host or from an endpoint (--from-ep).

When the command is run from the host, it can reference files present on the host, for example an SSH key:
	eden sdn fwd eth0 2222 ssh -I ./dist/tests/eclient/image/cert/id_rsa root@FWD_IP FWD_PORT
The command can also be run from an endpoint deployed inside Eden-SDN, for example:
	eden sdn fwd --from-ep my-client eth0 2222 nc FWD_IP FWD_PORT
Note that in this case the command must be installed in Eden-SDN VM (see sdn/Dockerfile)!

The target interface should be referenced by its name inside the kernel of EVE VM (e.g. "eth0").
This is currently limited to TCP port forwarding (i.e. not working with UDP)!`,
	Args: cobra.MinimumNArgs(3),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			hostFwd = viper.GetStringMapString("eve.hostfwd")
			devModel = viper.GetString("eve.devmodel")
			eveRemote = viper.GetBool("eve.remote")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if eveRemote {
			log.Fatal("Eden command is not supported in the Remote mode")
		}
		eveIfName := args[0]
		targetPort, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatalf("Failed to parse target port: %v", err)
		}
		command := args[2]
		args = args[3:]
		sdnForwardCmd(sdnFwdFromEp, eveIfName, targetPort, command, args...)
	},
}

func sdnForwardCmd(fromEp string, eveIfName string, targetPort int, cmd string, args ...string) {
	const fwdIPLabel = "FWD_IP"
	const fwdPortLabel = "FWD_PORT"
	if !isSdnEnabled() {
		if fromEp != "" {
			log.Warnf("Cannot execute command from an endpoint without SDN running, " +
				"argument \"from-ep\" will be ignored")
		}
		// Network model is static and consists of two Eve interfaces.
		if eveIfName != "eth0" && eveIfName != "eth1" {
			log.Fatalf("Unknown EVE interface: %s", eveIfName)
		}
		// Find out what the targetPort is (statically) mapped to in the host.
		targetHostPort := -1
		for k, v := range hostFwd {
			hostPort, err := strconv.Atoi(k)
			if err != nil {
				log.Errorf("failed to parse host port from eve.hostfwd: %v", err)
				continue
			}
			guestPort, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("failed to parse guest port from eve.hostfwd: %v", err)
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
			log.Fatalf("Target EVE interface and port (%s, %d) are not port-forwarded "+
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
			log.Fatalf("command %s failed: %v", cmd, err)
		}
		return
	}
	// TODO: Port forwarding with SDN only works for TCP for now.

	// Get IP address used by the target EVE interface.
	targetIP := getEVEIP(eveIfName)
	if targetIP == "" {
		log.Fatalf("no IP address found to be assigned to EVE interface %s",
			eveIfName)
	}
	// Temporarily establish port forwarding using SSH.
	localPort, err := utils.FindUnusedPort()
	if err != nil {
		log.Fatalf("failed to find unused port number: %v", err)
	}
	client := &edensdn.SdnClient{
		SSHPort:  uint16(sdnSSHPort),
		MgmtPort: uint16(sdnMgmtPort),
	}
	closeTunnel, err := client.SSHPortForwarding(localPort, uint16(targetPort), targetIP)
	if err != nil {
		log.Fatalf("failed to establish SSH port forwarding: %v", err)
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
		log.Fatalf("command %s %s failed: %v", cmd, strings.Join(args, " "), err)
	}
}

func sdnForwardSSHToEve(commandToRun string) {
	arguments := fmt.Sprintf("-o ConnectTimeout=5 -o StrictHostKeyChecking=no -i %s "+
		"-p FWD_PORT root@FWD_IP %s", eveSSHKey, commandToRun)
	sdnForwardCmd("", "eth0", 22, "ssh", strings.Fields(arguments)...)
}

func sdnForwardSCPFromEve(remoteFilePath, localFilePath string) {
	arguments := fmt.Sprintf("-o ConnectTimeout=5 -o StrictHostKeyChecking=no -i %s "+
		"-P FWD_PORT root@FWD_IP:%s %s", eveSSHKey, remoteFilePath, localFilePath)
	sdnForwardCmd("", "eth0", 22, "scp", strings.Fields(arguments)...)
}

// Run after loading these options from config:
//   - devModel = viper.GetString("eve.devmodel")
//   - loadSdnOptsFromViper()
func isSdnEnabled() bool {
	// Only supported with QEMU for now.
	return !sdnDisable && devModel == defaults.DefaultQemuModel
}

func loadSdnOptsFromViper() {
	sdnImageFile = utils.ResolveAbsPath(viper.GetString("sdn.image-file"))
	sdnNetModelFile = utils.ResolveAbsPath(viper.GetString("sdn.network-model"))
	sdnConsoleLogFile = utils.ResolveAbsPath(viper.GetString("sdn.console-log"))
	sdnDisable = viper.GetBool("sdn.disable")
	sdnTelnetPort = viper.GetInt("sdn.telnet-port")
	sdnSSHPort = viper.GetInt("sdn.ssh-port")
	sdnMgmtPort = viper.GetInt("sdn.mgmt-port")
	sdnPidFile = utils.ResolveAbsPath(viper.GetString("sdn.pid"))
}

func addSdnStartOpts(parentCmd *cobra.Command) {
	addSdnPidOpt(parentCmd)
	addSdnNetModelOpt(parentCmd)
	addSdnPortOpts(parentCmd)
	addSdnLogOpt(parentCmd)
	addSdnImageOpt(parentCmd)
	addSdnDisableOpt(parentCmd)
}

func addSdnPidOpt(parentCmd *cobra.Command) {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	parentCmd.Flags().StringVarP(&sdnPidFile, "sdn-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "sdn.pid"), "file for saving SDN pid")
}

func addSdnNetModelOpt(parentCmd *cobra.Command) {
	parentCmd.Flags().StringVarP(&sdnNetModelFile, "sdn-network-model", "", "", "path to JSON file with network model to apply into SDN")
}

func addSdnPortOpts(parentCmd *cobra.Command) {
	parentCmd.Flags().IntVarP(&sdnTelnetPort, "sdn-telnet-port", "", defaults.DefaultSdnTelnetPort, "port for telnet (console access) to SDN VM")
	parentCmd.Flags().IntVarP(&sdnMgmtPort, "sdn-mgmt-port", "", defaults.DefaultSdnMgmtPort, "port for access to the management agent running inside SDN VM")
	parentCmd.Flags().IntVarP(&sdnSSHPort, "sdn-ssh-port", "", defaults.DefaultSdnSSHPort, "port for SSH access to SDN VM")
}

func addSdnLogOpt(parentCmd *cobra.Command) {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	parentCmd.Flags().StringVarP(&sdnConsoleLogFile, "sdn-console-log", "", filepath.Join(currentPath, defaults.DefaultDist, "sdn-console.log"), "log file for SDN console output")
}

func addSdnImageOpt(parentCmd *cobra.Command) {
	parentCmd.Flags().StringVarP(&sdnImageFile, "sdn-image-file", "", "", "path to SDN image drive, overrides default setting")
}

func addSdnDisableOpt(parentCmd *cobra.Command) {
	parentCmd.Flags().BoolVarP(&sdnDisable, "sdn-disable", "", false, "disable Eden-SDN (do not run SDN VM)")
}

func sdnInit() {
	sdnCmd.AddCommand(sdnNetModelCmd)
	sdnCmd.AddCommand(sdnNetConfigGraphCmd)
	sdnCmd.AddCommand(sdnStatusCmd)
	sdnCmd.AddCommand(sdnSshCmd)
	sdnCmd.AddCommand(sdnLogsCmd)
	sdnCmd.AddCommand(sdnMgmtIpCmd)
	sdnCmd.AddCommand(sdnEndpointCmd)
	sdnCmd.AddCommand(sdnFwdCmd)
	sdnNetModelCmd.AddCommand(sdnNetModelApplyCmd)
	sdnNetModelCmd.AddCommand(sdnNetModelGetCmd)
	sdnEndpointCmd.AddCommand(sdnEpExecCmd)

	addSdnPidOpt(sdnStatusCmd)
	addSdnPortOpts(sdnStatusCmd)
	addSdnPortOpts(sdnNetConfigGraphCmd)
	addSdnPortOpts(sdnSshCmd)
	addSdnPortOpts(sdnLogsCmd)
	addSdnPortOpts(sdnMgmtIpCmd)
	addSdnPortOpts(sdnEndpointCmd)
	addSdnPortOpts(sdnNetModelApplyCmd)
	addSdnPortOpts(sdnNetModelGetCmd)
	addSdnPortOpts(sdnEpExecCmd)
	addSdnPortOpts(sdnFwdCmd)
	sdnFwdCmd.Flags().StringVarP(&sdnFwdFromEp, "from-ep", "", "",
		"run port-forwarded command from inside of the given Eden-SDN endpoint "+
			"(referenced by logical label)")
}
