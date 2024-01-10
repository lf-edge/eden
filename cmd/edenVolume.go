package cmd

import (
	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

func newVolumeCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var volumeCmd = &cobra.Command{
		Use:               "volume",
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newVolumeLsCmd(),
				newVolumeCreateCmd(),
				newVolumeDeleteCmd(),
				newVolumeDetachCmd(),
				newVolumeAttachCmd(),
			},
		},
	}

	groups.AddTo(volumeCmd)

	return volumeCmd
}

func newVolumeLsCmd() *cobra.Command {
	var outputFormat types.OutputFormat
	//volumeLsCmd is a command to list deployed volumes
	var volumeLsCmd = &cobra.Command{
		Use:   "ls",
		Short: "List volumes",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.VolumeLs(outputFormat); err != nil {
				log.Fatal(err)
			}
		},
	}
	volumeLsCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")
	return volumeLsCmd
}

func newVolumeCreateCmd() *cobra.Command {
	var registry, diskSize, volumeName, volumeType, datastoreOverride string
	var sftpLoad, directLoad bool

	//volumeCreateCmd is a command to create volume
	var volumeCreateCmd = &cobra.Command{
		Use:   "create <(docker|http(s)|file)://(<TAG>[:<VERSION>] | <URL for qcow2 image> | <path to qcow2 image>| blank)>",
		Short: "Create volume",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appLink := args[0]
			err := openEVEC.VolumeCreate(appLink, registry, diskSize, volumeName,
				volumeType, datastoreOverride, sftpLoad, directLoad)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	volumeCreateCmd.Flags().StringVar(&registry, "registry", "remote", "Select registry to use for containers (remote/local)")
	volumeCreateCmd.Flags().StringVar(&diskSize, "disk-size", humanize.Bytes(0), "disk size (empty or 0 - same as in image)")
	volumeCreateCmd.Flags().StringVarP(&volumeName, "name", "n", "", "name of volume, random if empty")
	volumeCreateCmd.Flags().StringVar(&volumeType, "format", "", "volume type (qcow2, raw, qcow, vmdk, vhdx, iso or oci)")
	volumeCreateCmd.Flags().BoolVar(&sftpLoad, "sftp", false, "force eserver to use sftp")
	volumeCreateCmd.Flags().BoolVar(&directLoad, "direct", true, "Use direct download for image instead of eserver")
	volumeCreateCmd.Flags().StringVar(&datastoreOverride, "datastoreOverride", "", "Override datastore path for volume (when we use different URL for Eden and EVE or for local datastore)")

	return volumeCreateCmd
}

func newVolumeDeleteCmd() *cobra.Command {
	//volumeDeleteCmd is a command to delete volume
	var volumeDeleteCmd = &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete volume",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			volumeName := args[0]
			if err := openEVEC.VolumeDelete(volumeName); err != nil {
				log.Fatal(err)
			}
		},
	}

	return volumeDeleteCmd
}

func newVolumeDetachCmd() *cobra.Command {
	//volumeDetachCmd is a command to detach volume
	var volumeDetachCmd = &cobra.Command{
		Use:   "detach <name>",
		Short: "Detach volume",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			volumeName := args[0]
			if err := openEVEC.VolumeDetach(volumeName); err != nil {
				log.Fatal(err)
			}
		},
	}

	return volumeDetachCmd
}

func newVolumeAttachCmd() *cobra.Command {
	//volumeAttachCmd is a command to attach volume to app instance
	var volumeAttachCmd = &cobra.Command{
		Use:   "attach <volume name> <app name> [mount point]",
		Short: "Attach volume to app",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			volumeName := args[0]
			appName := args[1]
			mountPoint := ""
			if len(args) > 2 {
				mountPoint = args[2]
			}

			if err := openEVEC.VolumeAttach(appName, volumeName, mountPoint); err != nil {
				log.Fatal(err)
			}
		},
	}
	return volumeAttachCmd
}
