package cmd

import (
	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	edgeRegistry "github.com/lf-edge/edge-containers/pkg/registry"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

func newPodCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var podCmd = &cobra.Command{
		Use:               "pod",
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	groups := CommandGroups{
		{
			Message: "Control Commands",
			Commands: []*cobra.Command{
				newPodDeployCmd(cfg),
				newPodStopCmd(),
				newPodStartCmd(),
				newPodDeleteCmd(),
				newPodRestartCmd(),
				newPodPurgeCmd(),
				newPodModifyCmd(),
				newPodPublishCmd(),
			},
		},
		{
			Message: "Printing Commands",
			Commands: []*cobra.Command{
				newPodPsCmd(),
				newPodLogsCmd(cfg),
			},
		},
	}

	groups.AddTo(podCmd)

	return podCmd
}

func newPodPublishCmd() *cobra.Command {
	var kernelFile, initrdFile, rootFile, formatStr, arch string
	var disks []string
	var local bool

	var podPublishCmd = &cobra.Command{
		Use:   "publish <image>",
		Short: "Publish pod files into image",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if err := openEVEC.PodPublish(appName, kernelFile, initrdFile, rootFile, formatStr, arch, local, disks); err != nil {
				log.Fatal(err)
			}
		},
	}

	podPublishCmd.Flags().StringVar(&kernelFile, "kernel", "", "path to kernel file, optional")
	podPublishCmd.Flags().StringVar(&initrdFile, "initrd", "", "path to initrd file, optional")
	podPublishCmd.Flags().StringVar(&rootFile, "root", "", "path to root disk file and format (for example: image.img:qcow2)")
	podPublishCmd.Flags().StringSliceVar(&disks, "disks", []string{}, "disks to add into image")
	podPublishCmd.Flags().BoolVar(&local, "local", false, "push to local registry")
	podPublishCmd.Flags().StringVar(&formatStr, "format", "artifacts", "which format to use, one of: artifacts, legacy")
	podPublishCmd.Flags().StringVar(&arch, "arch", edgeRegistry.DefaultArch, "arch to deploy")

	return podPublishCmd
}

func newPodDeployCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var pc openevec.PodConfig

	var podDeployCmd = &cobra.Command{
		Use:   "deploy (docker|http(s)|file|directory)://(<TAG|PATH>[:<VERSION>] | <URL for qcow2 image> | <path to qcow2 image>)",
		Short: "Deploy app in pod",
		Long:  `Deploy app in pod.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appLink := args[0]
			if err := openEVEC.PodDeploy(appLink, pc, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	podDeployCmd.Flags().StringVar(&pc.AppMemory, "memory", humanize.Bytes(defaults.DefaultAppMem*1024), "memory for app")
	podDeployCmd.Flags().StringVar(&pc.DiskSize, "disk-size", humanize.Bytes(0), "disk size (empty or 0 - same as in image)")
	podDeployCmd.Flags().StringVar(&pc.VolumeType, "volume-type", "qcow2", "volume type for empty volumes (qcow2, raw, qcow, vmdk, vhdx, iso or oci); set it to none to not use volumes")
	podDeployCmd.Flags().StringSliceVarP(&pc.PortPublish, "publish", "p", nil, "Ports to publish in format EXTERNAL_PORT:INTERNAL_PORT")
	podDeployCmd.Flags().StringVarP(&pc.Metadata, "metadata", "", "", "Metadata for pod. If file path provided, will use content of it")
	podDeployCmd.Flags().StringVarP(&pc.Name, "name", "n", "", "name for pod")
	podDeployCmd.Flags().IntVar(&pc.VncDisplay, "vnc-display", -1, "display number for VNC pod")
	podDeployCmd.Flags().StringVar(&pc.VncPassword, "vnc-password", "", "VNC password (empty - no password)")
	podDeployCmd.Flags().BoolVar(&pc.VncForShimVM, "vnc-for-shim-vm", false, "Enable VNC for a shim VM")
	podDeployCmd.Flags().Uint32Var(&pc.AppCpus, "cpus", defaults.DefaultAppCPU, "cpu number for app")
	podDeployCmd.Flags().StringSliceVar(&pc.AppAdapters, "adapters", nil, "adapters to assign to the application instance")
	podDeployCmd.Flags().StringSliceVar(&pc.Networks, "networks", nil, "Networks to connect to app (ports will be mapped to first network). May have <name:[MAC address]> notation.")
	podDeployCmd.Flags().StringVar(&pc.ImageFormat, "format", "", "format for image, one of 'container','qcow2','raw','qcow','vmdk','vhdx','iso'; if not provided, defaults to container image for docker and oci transports, qcow2 for file and http/s transports")
	podDeployCmd.Flags().BoolVar(&pc.ACLOnlyHost, "only-host", false, "Allow access only to host and external networks")
	podDeployCmd.Flags().BoolVar(&pc.NoHyper, "no-hyper", false, "Run pod without hypervisor")
	podDeployCmd.Flags().StringVar(&pc.Registry, "registry", "remote", "Select registry to use for containers (remote/local)")
	podDeployCmd.Flags().BoolVar(&pc.DirectLoad, "direct", true, "Use direct download for image instead of eserver")
	podDeployCmd.Flags().BoolVar(&pc.SftpLoad, "sftp", false, "Force use of sftp to load http/file image from eserver")
	podDeployCmd.Flags().StringSliceVar(&pc.Disks, "disks", nil, `Additional disks to use. You can write it in notation <link> or <mount point>:<link>. Deprecated. Please use volumes instead.`)
	podDeployCmd.Flags().StringArrayVar(&pc.Mount, "mount", nil, `Additional volumes to use. You can write it in notation src=<link>,dst=<mount point>.`)
	podDeployCmd.Flags().StringVar(&pc.VolumeSize, "volume-size", humanize.IBytes(defaults.DefaultVolumeSize), "volume size")
	podDeployCmd.Flags().StringSliceVar(&pc.Profiles, "profile", nil, "profile to set for app")
	podDeployCmd.Flags().StringSliceVar(&pc.ACL, "acl", nil, `Allow access only to defined hosts/ips/subnets.
Without explicitly configured ACLs, all traffic is allowed.
You can set ACL for a particular network in format '<network_name[:endpoint[:action]]>', where 'action' is either 'allow' (default) or 'drop'.
With ACLs configured, endpoints not matched by any rule are blocked.
To block all traffic define ACL with no endpoints: '<network_name>:'`)
	podDeployCmd.Flags().StringSliceVar(&pc.Vlans, "vlan", nil, `Connect application to the (switch) network over an access port assigned to the given VLAN.
You can set access VLAN ID (VID) for a particular network in the format '<network_name:VID>'`)
	podDeployCmd.Flags().BoolVar(&pc.OpenStackMetadata, "openstack-metadata", false, "Use OpenStack metadata for VM")
	podDeployCmd.Flags().StringVar(&pc.DatastoreOverride, "datastoreOverride", "", "Override datastore path for disks (when we use different URL for Eden and EVE or for local datastore)")
	podDeployCmd.Flags().Uint32Var(&pc.StartDelay, "start-delay", 0, "The amount of time (in seconds) that EVE waits (after boot finish) before starting application")
	podDeployCmd.Flags().BoolVar(&pc.PinCpus, "pin-cpus", false, "Pin the CPUs used by the pod")

	return podDeployCmd
}

func newPodPsCmd() *cobra.Command {
	var outputFormat types.OutputFormat
	var podPsCmd = &cobra.Command{
		Use:   "ps",
		Short: "List pods",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.PodPs(outputFormat); err != nil {
				log.Fatalf("EVE pod deploy failed: %s", err)
			}
		},
	}
	podPsCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")

	return podPsCmd
}

func newPodStopCmd() *cobra.Command {
	var podStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if err := openEVEC.PodStop(appName); err != nil {
				log.Fatalf("EVE pod stop failed: %s", err)
			}
		},
	}

	return podStopCmd
}

func newPodPurgeCmd() *cobra.Command {
	var volumesToPurge []string

	var podPurgeCmd = &cobra.Command{
		Use:   "purge",
		Short: "Purge pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			explicitVolumes := cmd.Flags().Changed("volumes")
			if err := openEVEC.PodPurge(volumesToPurge, appName, explicitVolumes); err != nil {
				log.Fatalf("EVE pod purge failed: %s", err)
			}
		},
	}

	podPurgeCmd.Flags().StringSliceVar(&volumesToPurge, "volumes", []string{}, "Explicitly set volume names to purge, purge all if not defined")

	return podPurgeCmd
}

func newPodRestartCmd() *cobra.Command {
	var podRestartCmd = &cobra.Command{
		Use:   "restart",
		Short: "Restart pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if err := openEVEC.PodRestart(appName); err != nil {
				log.Fatalf("EVE pod restart failed: %s", err)
			}
		},
	}

	return podRestartCmd
}

func newPodStartCmd() *cobra.Command {
	var podStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if err := openEVEC.PodStart(appName); err != nil {
				log.Fatalf("EVE pod start failed: %s", err)
			}
		},
	}

	return podStartCmd
}

func newPodDeleteCmd() *cobra.Command {
	var deleteVolumes bool

	var podDeleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if _, err := openEVEC.PodDelete(appName, deleteVolumes); err != nil {
				log.Fatalf("EVE pod start failed: %s", err)
			}
		},
	}

	podDeleteCmd.Flags().BoolVar(&deleteVolumes, "with-volumes", true, "delete volumes of pod")

	return podDeleteCmd
}

func newPodLogsCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var (
		outputTail   uint
		outputFields []string
		outputFormat types.OutputFormat
	)

	var podLogsCmd = &cobra.Command{
		Use:   "logs <name>",
		Short: "Logs of pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if err := openEVEC.PodLogs(appName, outputTail, outputFields, outputFormat); err != nil {
				log.Fatalf("EVE pod start failed: %s", err)
			}
		},
	}

	podLogsCmd.Flags().UintVar(&outputTail, "tail", 0, "Show only last N lines")
	podLogsCmd.Flags().StringSliceVar(&outputFields, "fields", []string{"log", "info", "metric", "netstat", "app"}, "Show defined elements")
	podLogsCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")

	return podLogsCmd
}

func newPodModifyCmd() *cobra.Command {
	var podNetworks, portPublish, acl, vlans []string
	var startDelay uint32

	var podModifyCmd = &cobra.Command{
		Use:   "modify <app>",
		Short: "Modify pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if err := openEVEC.PodModify(appName, podNetworks, portPublish, acl, vlans, startDelay); err != nil {
				log.Fatalf("EVE pod start failed: %s", err)
			}
		},
	}

	podModifyCmd.Flags().StringSliceVarP(&portPublish, "publish", "p", nil, "Ports to publish in format EXTERNAL_PORT:INTERNAL_PORT")
	podModifyCmd.Flags().StringSliceVar(&podNetworks, "networks", nil, "Networks to connect to app (ports will be mapped to first network). May have <name:[MAC address]> notation.")
	podModifyCmd.Flags().StringSliceVar(&acl, "acl", nil, `Allow access only to defined hosts/ips/subnets.
Without explicitly configured ACLs, all traffic is allowed.
You can set ACL for a particular network in format '<network_name[:endpoint[:action]]>', where 'action' is either 'allow' (default) or 'drop'.
With ACLs configured, endpoints not matched by any rule are blocked.
To block all traffic define ACL with no endpoints: '<network_name>:'`)
	podModifyCmd.Flags().StringSliceVar(&vlans, "vlan", nil, `Connect application to the (switch) network over an access port assigned to the given VLAN.
You can set access VLAN ID (VID) for a particular network in the format '<network_name:VID>'`)
	podModifyCmd.Flags().Uint32Var(&startDelay, "start-delay", 0, "The amount of time (in seconds) that EVE waits (after boot finish) before starting application")

	return podModifyCmd
}
