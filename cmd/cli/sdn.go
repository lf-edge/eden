package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	sdnFwdFromEp string
)

func newSdnCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var sdnCmd = &cobra.Command{
		Use:               "sdn",
		Short:             "Emulate and manage networks surrounding EVE VM using Eden-SDN",
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newSdnNetModelCmd(cfg),
				newSdnNetConfigGraphCmd(cfg),
				newSdnStatusCmd(cfg),
				newSdnSshCmd(cfg),
				newSdnLogsCmd(cfg),
				newSdnMgmtIPCmd(cfg),
				newSdnEndpointCmd(cfg),
				newSdnFwdCmd(cfg),
			},
		},
	}

	groups.AddTo(sdnCmd)

	return sdnCmd
}

func newSdnNetModelCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var sdnNetModelCmd = &cobra.Command{
		Use:   "net-model",
		Short: "Manage network model submitted to Eden-SDN",
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newSdnNetModelApplyCmd(cfg),
				newSdnModelGetCmd(cfg),
			},
		},
	}

	groups.AddTo(sdnNetModelCmd)

	return sdnNetModelCmd
}

func newSdnModelGetCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var sdnNetModelGetCmd = &cobra.Command{
		Use:   "get",
		Short: "Get currently applied network model (in JSON)",
		Run: func(cmd *cobra.Command, args []string) {
			model, err := openevec.SdnNetModelGet(cfg)
			if err != nil {
				log.Fatal(err)
			} else {
				fmt.Println(model)
			}
		},
	}

	return sdnNetModelGetCmd
}

func newSdnNetModelApplyCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var sdnNetModelApplyCmd = &cobra.Command{
		Use:   "apply <filepath.json|default>",
		Short: "submit network model into Eden-SDN",
		Long: `Load network model from a JSON file and submit it to Eden-SDN.
Use string \"default\" instead of a file path to apply the default network model
(two eth interfaces inside the same network with DHCP, see DefaultNetModel in pkg/edensdn/netModel.go).`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ref := args[0]
			if err := openevec.SdnNetModelApply(ref, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}
	addSdnPortOpts(sdnNetModelApplyCmd, cfg)

	return sdnNetModelApplyCmd
}

func newSdnNetConfigGraphCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var sdnNetConfigGraphCmd = &cobra.Command{
		Use:   "net-config-graph",
		Short: "get network config applied by Eden-SDN, visualized using a dependency graph",
		Long: `Get network config applied by Eden-SDN.
Network config items and their dependencies are depicted using a DOT graph.
To generate graph image, run: eden sdn net-config-graph | dot -Tsvg > output.svg
This requires to have Graphviz installed.
Alternatively, visualize using the online tool: https://dreampuf.github.io/GraphvizOnline/`,
		Run: func(cmd *cobra.Command, args []string) {
			graph, err := openevec.SdnNetConfigGraph(cfg)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(graph)
		},
	}
	addSdnPortOpts(sdnNetConfigGraphCmd, cfg)
	return sdnNetConfigGraphCmd
}

func newSdnStatusCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var sdnStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Get status of the running Eden-SDN",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.SdnStatus(cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	addSdnPortOpts(sdnStatusCmd, cfg)
	addSdnPidOpt(sdnStatusCmd, cfg)

	return sdnStatusCmd
}

func newSdnSshCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var sdnSshCmd = &cobra.Command{
		Use:   "ssh",
		Short: "SSH into the running Eden-SDN VM",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.SdnSsh(cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	addSdnPortOpts(sdnSshCmd, cfg)

	return sdnSshCmd
}

func newSdnLogsCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var sdnLogsCmd = &cobra.Command{
		Use:   "logs",
		Short: "Get all logs from running Eden-SDN VM",
		Run: func(cmd *cobra.Command, args []string) {
			if logs, err := openevec.SdnLogs(cfg); err != nil {
				log.Fatal(err)
			} else {
				fmt.Println(logs)
			}
		},
	}

	addSdnPortOpts(sdnLogsCmd, cfg)

	return sdnLogsCmd
}

func newSdnMgmtIPCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var sdnMgmtIpCmd = &cobra.Command{
		Use:   "mgmt-ip",
		Short: "Get IP address assigned to Eden-SDN VM for management",
		Run: func(cmd *cobra.Command, args []string) {
			if mgmtIp, err := openevec.SdnMgmtIp(cfg); err != nil {
				log.Fatal(err)
			} else {
				fmt.Println(mgmtIp)
			}
		},
	}
	addSdnPortOpts(sdnMgmtIpCmd, cfg)
	return sdnMgmtIpCmd
}

func newSdnEndpointCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
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

	sdnEndpointCmd.AddCommand(newSdnEpExecCmd(cfg))
	addSdnPortOpts(sdnEndpointCmd, cfg)

	return sdnEndpointCmd
}

func newSdnEpExecCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
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
		Run: func(cmd *cobra.Command, args []string) {
			epName := args[0]
			command := args[1]
			args = args[2:]

			if err := openevec.SdnEpExec(epName, command, args, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	addSdnPortOpts(sdnEpExecCmd, cfg)

	return sdnEpExecCmd
}

func newSdnFwdCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
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
		Run: func(cmd *cobra.Command, args []string) {
			eveIfName := args[0]
			targetPort, err := strconv.Atoi(args[1])
			if err != nil {
				log.Fatalf("Failed to parse target port: %v", err)
			}
			command := args[2]
			args = args[3:]

			err = openevec.SdnForwardCmd(sdnFwdFromEp, eveIfName, targetPort, command, cfg, args...)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	addSdnPortOpts(sdnFwdCmd, cfg)
	sdnFwdCmd.Flags().StringVarP(&sdnFwdFromEp, "from-ep", "", "",
		"run port-forwarded command from inside of the given Eden-SDN endpoint "+
			"(referenced by logical label)")

	return sdnFwdCmd
}

func addSdnPidOpt(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	parentCmd.Flags().StringVarP(&cfg.Sdn.PidFile, "sdn-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "sdn.pid"), "file for saving SDN pid")
}

func addSdnNetModelOpt(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	parentCmd.Flags().StringVarP(&cfg.Sdn.NetModelFile, "sdn-network-model", "", "", "path to JSON file with network model to apply into SDN")
}

func addSdnVmOpts(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	parentCmd.Flags().IntVarP(&cfg.Sdn.RAM, "sdn-ram", "", defaults.DefaultSdnMemory, "memory (MB) for SDN VM")
	parentCmd.Flags().IntVarP(&cfg.Sdn.CPU, "sdn-cpu", "", defaults.DefaultSdnCpus, "CPU count for SDN VM")
}

func addSdnPortOpts(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	parentCmd.Flags().IntVarP(&cfg.Sdn.TelnetPort, "sdn-telnet-port", "", defaults.DefaultSdnTelnetPort, "port for telnet (console access) to SDN VM")
	parentCmd.Flags().IntVarP(&cfg.Sdn.MgmtPort, "sdn-mgmt-port", "", defaults.DefaultSdnMgmtPort, "port for access to the management agent running inside SDN VM")
	parentCmd.Flags().IntVarP(&cfg.Sdn.SSHPort, "sdn-ssh-port", "", defaults.DefaultSdnSSHPort, "port for SSH access to SDN VM")
}

func addSdnLogOpt(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	parentCmd.Flags().StringVarP(&cfg.Sdn.ConsoleLogFile, "sdn-console-log", "", filepath.Join(currentPath, defaults.DefaultDist, "sdn-console.log"), "log file for SDN console output")
}

func addSdnImageOpt(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	parentCmd.Flags().StringVarP(&cfg.Sdn.ImageFile, "sdn-image-file", "", "", "path to SDN image drive, overrides default setting")
}

func addSdnDisableOpt(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	parentCmd.Flags().BoolVarP(&cfg.Sdn.Disable, "sdn-disable", "", false, "disable Eden-SDN (do not run SDN VM)")
}

func addSdnSourceDirOpt(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	parentCmd.Flags().StringVarP(&cfg.Sdn.SourceDir, "sdn-source-dir", "", "", "directory with SDN source code")
}

func addSdnConfigDirOpt(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	parentCmd.Flags().StringVarP(&cfg.Sdn.ConfigDir, "sdn-config-dir", "", "", "directory where to put generated SDN-related config files")
}

func addSdnLinuxkitOpt(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	parentCmd.Flags().StringVarP(&cfg.Sdn.LinuxkitBin, "sdn-linuxkit-bin", "", "", "path to linuxkit binary used to build SDN VM")
}
