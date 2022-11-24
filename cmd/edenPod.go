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
				newPodStartCmd(cfg),
				newPodDeleteCmd(cfg),
				newPodRestartCmd(cfg),
				newPodPurgeCmd(cfg),
				newPodModifyCmd(cfg),
				newPodPublishCmd(cfg),
			},
		},
		{
			Message: "Printing Commands",
			Commands: []*cobra.Command{
				newPodPsCmd(cfg),
				newPodLogsCmd(cfg),
			},
		},
	}

	groups.AddTo(podCmd)

	return podCmd
}

func newPodPublishCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var kernelFile, initrdFile, rootFile, formatStr, arch string
	var disks []string
	var local bool

	var podPublishCmd = &cobra.Command{
		Use:   "publish <image>",
		Short: "Publish pod files into image",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if err := openevec.PodPublish(appName, kernelFile, initrdFile, rootFile, formatStr, arch, local, disks, cfg); err != nil {
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
	var podDeployCmd = &cobra.Command{
		Use:   "deploy (docker|http(s)|file|directory)://(<TAG|PATH>[:<VERSION>] | <URL for qcow2 image> | <path to qcow2 image>)",
		Short: "Deploy app in pod",
		Long:  `Deploy app in pod.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appLink := args[0]
			if err := openevec.PodDeploy(appLink, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	podDeployCmd.Flags().StringVar(&cfg.Runtime.AppMemory, "memory", humanize.Bytes(defaults.DefaultAppMem*1024), "memory for app")
	podDeployCmd.Flags().StringVar(&cfg.Runtime.DiskSize, "disk-size", humanize.Bytes(0), "disk size (empty or 0 - same as in image)")
	podDeployCmd.Flags().StringVar(&cfg.Runtime.VolumeType, "volume-type", "qcow2", "volume type for empty volumes (qcow2, raw, qcow, vmdk, vhdx or oci); set it to none to not use volumes")
	podDeployCmd.Flags().StringSliceVarP(&cfg.Runtime.PortPublish, "publish", "p", nil, "Ports to publish in format EXTERNAL_PORT:INTERNAL_PORT")
	podDeployCmd.Flags().StringVarP(&cfg.Runtime.PodMetadata, "metadata", "", "", "Metadata for pod. If file path provided, will use content of it")
	podDeployCmd.Flags().StringVarP(&cfg.Runtime.PodName, "name", "n", "", "name for pod")
	podDeployCmd.Flags().Uint32Var(&cfg.Runtime.VncDisplay, "vnc-display", 0, "display number for VNC pod (0 - no VNC)")
	podDeployCmd.Flags().StringVar(&cfg.Runtime.VncPassword, "vnc-password", "", "VNC password (empty - no password)")
	podDeployCmd.Flags().Uint32Var(&cfg.Runtime.AppCpus, "cpus", defaults.DefaultAppCPU, "cpu number for app")
	podDeployCmd.Flags().StringSliceVar(&cfg.Runtime.AppAdapters, "adapters", nil, "adapters to assign to the application instance")
	podDeployCmd.Flags().StringSliceVar(&cfg.Runtime.PodNetworks, "networks", nil, "Networks to connect to app (ports will be mapped to first network). May have <name:[MAC address]> notation.")
	podDeployCmd.Flags().StringVar(&cfg.Runtime.ImageFormat, "format", "", "format for image, one of 'container','qcow2','raw','qcow','vmdk','vhdx'; if not provided, defaults to container image for docker and oci transports, qcow2 for file and http/s transports")
	podDeployCmd.Flags().BoolVar(&cfg.Runtime.ACLOnlyHost, "only-host", false, "Allow access only to host and external networks")
	podDeployCmd.Flags().BoolVar(&cfg.Runtime.NoHyper, "no-hyper", false, "Run pod without hypervisor")
	podDeployCmd.Flags().StringVar(&cfg.Runtime.Registry, "registry", "remote", "Select registry to use for containers (remote/local)")
	podDeployCmd.Flags().BoolVar(&cfg.Runtime.DirectLoad, "direct", true, "Use direct download for image instead of eserver")
	podDeployCmd.Flags().BoolVar(&cfg.Runtime.SftpLoad, "sftp", false, "Force use of sftp to load http/file image from eserver")
	podDeployCmd.Flags().StringSliceVar(&cfg.Runtime.Disks, "disks", nil, `Additional disks to use. You can write it in notation <link> or <mount point>:<link>. Deprecated. Please use volumes instead.`)
	podDeployCmd.Flags().StringArrayVar(&cfg.Runtime.Mount, "mount", nil, `Additional volumes to use. You can write it in notation src=<link>,dst=<mount point>.`)
	podDeployCmd.Flags().StringVar(&cfg.Runtime.VolumeSize, "volume-size", humanize.IBytes(defaults.DefaultVolumeSize), "volume size")
	podDeployCmd.Flags().StringSliceVar(&cfg.Runtime.Profiles, "profile", nil, "profile to set for app")
	podDeployCmd.Flags().StringSliceVar(&cfg.Runtime.ACL, "acl", nil, `Allow access only to defined hosts/ips/subnets.
Without explicitly configured ACLs, all traffic is allowed.
You can set ACL for a particular network in format '<network_name[:endpoint[:action]]>', where 'action' is either 'allow' (default) or 'drop'.
With ACLs configured, endpoints not matched by any rule are blocked.
To block all traffic define ACL with no endpoints: '<network_name>:'`)
	podDeployCmd.Flags().StringSliceVar(&cfg.Runtime.VLANs, "vlan", nil, `Connect application to the (switch) network over an access port assigned to the given VLAN.
You can set access VLAN ID (VID) for a particular network in the format '<network_name:VID>'`)
	podDeployCmd.Flags().BoolVar(&cfg.Runtime.OpenStackMetadata, "openstack-metadata", false, "Use OpenStack metadata for VM")
	podDeployCmd.Flags().StringVar(&cfg.Runtime.DatastoreOverride, "datastoreOverride", "", "Override datastore path for disks (when we use different URL for Eden and EVE or for local datastore)")
	podDeployCmd.Flags().Uint32Var(&cfg.Runtime.StartDelay, "start-delay", 0, "The amount of time (in seconds) that EVE waits (after boot finish) before starting application")

	return podDeployCmd
}

func newPodPsCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var podPsCmd = &cobra.Command{
		Use:   "ps",
		Short: "List pods",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.PodPs(cfg); err != nil {
				log.Fatalf("EVE pod deploy failed: %s", err)
			}
		},
	}

	return podPsCmd
}

func newPodStopCmd() *cobra.Command {
	var podStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if err := openevec.PodStop(appName); err != nil {
				log.Fatalf("EVE pod stop failed: %s", err)
			}
		},
	}

	return podStopCmd
}

func newPodPurgeCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var podPurgeCmd = &cobra.Command{
		Use:   "purge",
		Short: "Purge pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			explicitVolumes := cmd.Flags().Changed("volumes")
			if err := openevec.PodPurge(cfg.Runtime.VolumesToPurge, appName, explicitVolumes); err != nil {
				log.Fatalf("EVE pod purge failed: %s", err)
			}
		},
	}

	podPurgeCmd.Flags().StringSliceVar(&cfg.Runtime.VolumesToPurge, "volumes", []string{}, "Explicitly set volume names to purge, purge all if not defined")

	return podPurgeCmd
}

func newPodRestartCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var podRestartCmd = &cobra.Command{
		Use:   "restart",
		Short: "Restart pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if err := openevec.PodRestart(appName); err != nil {
				log.Fatalf("EVE pod restart failed: %s", err)
			}
		},
	}

	return podRestartCmd
}

func newPodStartCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var podStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if err := openevec.PodStart(appName); err != nil {
				log.Fatalf("EVE pod start failed: %s", err)
			}
		},
	}

	return podStartCmd
}

func newPodDeleteCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var podDeleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			if _, err := openevec.PodDelete(appName, cfg.Runtime.DeleteVolumes); err != nil {
				log.Fatalf("EVE pod start failed: %s", err)
			}
		},
	}

	podDeleteCmd.Flags().BoolVar(&cfg.Runtime.DeleteVolumes, "with-volumes", true, "delete volumes of pod")

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
			if err := openevec.PodLogs(appName, outputTail, outputFields, outputFormat); err != nil {
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

func newPodModifyCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var podModifyCmd = &cobra.Command{
		Use:   "modify <app>",
		Short: "Modify pod",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appName := args[0]
			// TODO: might be cfg.Runtime
			if err := openevec.PodModify(appName, cfg); err != nil {
				log.Fatalf("EVE pod start failed: %s", err)
			}
		},
	}

	podModifyCmd.Flags().StringSliceVarP(&cfg.Runtime.PortPublish, "publish", "p", nil, "Ports to publish in format EXTERNAL_PORT:INTERNAL_PORT")
	podModifyCmd.Flags().BoolVar(&cfg.Runtime.ACLOnlyHost, "only-host", false, "Allow access only to host and external networks")
	podModifyCmd.Flags().StringSliceVar(&cfg.Runtime.PodNetworks, "networks", nil, "Networks to connect to app (ports will be mapped to first network). May have <name:[MAC address]> notation.")
	podModifyCmd.Flags().StringSliceVar(&cfg.Runtime.ACL, "acl", nil, `Allow access only to defined hosts/ips/subnets.
Without explicitly configured ACLs, all traffic is allowed.
You can set ACL for a particular network in format '<network_name[:endpoint[:action]]>', where 'action' is either 'allow' (default) or 'drop'.
With ACLs configured, endpoints not matched by any rule are blocked.
To block all traffic define ACL with no endpoints: '<network_name>:'`)
	podModifyCmd.Flags().StringSliceVar(&cfg.Runtime.VLANs, "vlan", nil, `Connect application to the (switch) network over an access port assigned to the given VLAN.
You can set access VLAN ID (VID) for a particular network in the format '<network_name:VID>'`)

	return podModifyCmd
}
