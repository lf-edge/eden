package cmd

import (
	"fmt"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/linuxkit"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newGcpImageCmd(cfg *openevec.EdenSetupArgs, gcpKey, gcpProjectName *string) *cobra.Command {
	var gcpImageCmd = &cobra.Command{
		Use:   "image",
		Short: `Manage images in gcp`,
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newGcpImageListCmd(gcpKey, gcpProjectName),
				newGcpImageUploadCmd(cfg, gcpKey, gcpProjectName),
				newGcpImageDelete(gcpKey, gcpProjectName),
			},
		},
	}

	groups.AddTo(gcpImageCmd)

	return gcpImageCmd
}

func newGcpImageDelete(gcpKey, gcpProjectName *string) *cobra.Command {
	var gcpImageName, gcpBucketName string

	var gcpImageDelete = &cobra.Command{
		Use:   "delete",
		Short: "delete image from gcp",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.GcpImageDelete(*gcpKey, *gcpProjectName, gcpImageName, gcpBucketName); err != nil {
				log.Fatal(err)
			}
		},
	}

	gcpImageDelete.Flags().StringVar(&gcpImageName, "image-name", defaults.DefaultGcpImageName, "image name")
	gcpImageDelete.Flags().StringVar(&gcpBucketName, "bucket-name", defaults.DefaultGcpBucketName, "bucket name to upload into")

	return gcpImageDelete
}

func newGcpImageListCmd(gcpKey, gcpProjectName *string) *cobra.Command {
	var gcpImageList = &cobra.Command{
		Use:   "list",
		Short: "list images uploaded to gcp",
		Long:  `Show list of images from gcp project.`,
		Run: func(cmd *cobra.Command, args []string) {
			gcpClient, err := linuxkit.NewGCPClient(*gcpKey, *gcpProjectName)
			if err != nil {
				log.Fatalf("Unable to connect to GCP: %v", err)
			}
			imageList, err := gcpClient.ListImages()
			if err != nil {
				log.Fatal(err)
			}
			for _, el := range imageList {
				fmt.Println(el)
			}
		},
	}

	return gcpImageList
}
func newGcpImageUploadCmd(cfg *openevec.EdenSetupArgs, gcpKey, gcpProjectName *string) *cobra.Command {
	var gcpImageName, gcpBucketName string

	var gcpImageUpload = &cobra.Command{
		Use:   "upload",
		Short: "upload image to gcp",
		Run: func(cmd *cobra.Command, args []string) {
			err := openevec.GcpImageUpload(*gcpKey, *gcpProjectName, gcpImageName, gcpBucketName, cfg.Eve.ImageFile, cfg.Eve.TPM)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	gcpImageUpload.Flags().StringVar(&gcpImageName, "image-name", defaults.DefaultGcpImageName, "image name")
	gcpImageUpload.Flags().StringVar(&cfg.Eve.ImageFile, "image-file", "", "image file to upload")
	gcpImageUpload.Flags().StringVar(&gcpBucketName, "bucket-name", defaults.DefaultGcpBucketName, "bucket name to upload into")
	gcpImageUpload.Flags().BoolVar(&cfg.Eve.TPM, "tpm", defaults.DefaultTPMEnabled, "enable UEFI to support vTPM for image")

	return gcpImageUpload
}

func newGcpVMCmd(cfg *openevec.EdenSetupArgs, gcpKey, gcpProjectName *string) *cobra.Command {
	var gcpVMCmd = &cobra.Command{
		Use:   "vm",
		Short: `Manage VMs in gcp`,
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newGcpRunCmd(cfg, gcpKey, gcpProjectName),
				newGcpDeleteCmd(gcpKey, gcpProjectName),
				newGcpConsoleCmd(gcpKey, gcpProjectName),
				newGcpLogCmd(gcpKey, gcpProjectName),
				newGcpGetIPCmd(gcpKey, gcpProjectName),
			},
		},
	}

	groups.AddTo(gcpVMCmd)

	return gcpVMCmd
}

func newGcpRunCmd(cfg *openevec.EdenSetupArgs, gcpKey, gcpProjectName *string) *cobra.Command {
	var gcpVMName, gcpImageName, gcpZone, gcpMachineType string

	var gcpRun = &cobra.Command{
		Use:   "run",
		Short: "run vm in gcp",
		Run: func(cmd *cobra.Command, args []string) {
			err := openevec.GcpRun(*gcpKey, *gcpProjectName, gcpImageName, gcpVMName, gcpZone, gcpMachineType, cfg.Eve.TPM, cfg.Eve.Disks, cfg.Eve.ImageSizeMB)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	gcpRun.Flags().StringVar(&gcpImageName, "image-name", defaults.DefaultGcpImageName, "image name")
	gcpRun.Flags().StringVar(&gcpVMName, "vm-name", defaults.DefaultGcpImageName, "vm name")
	gcpRun.Flags().StringVar(&gcpZone, "zone", defaults.DefaultGcpZone, "gcp zone")
	gcpRun.Flags().StringVar(&gcpMachineType, "machine-type", defaults.DefaultGcpMachineType, "gcp machine type")
	gcpRun.Flags().BoolVar(&cfg.Eve.TPM, "tpm", defaults.DefaultTPMEnabled, "enable vTPM for VM")

	return gcpRun
}

func newGcpDeleteCmd(gcpKey, gcpProjectName *string) *cobra.Command {
	var gcpVMName, gcpZone string

	var gcpDelete = &cobra.Command{
		Use:   "delete",
		Short: "delete vm from gcp",
		Run: func(cmd *cobra.Command, args []string) {
			err := openevec.GcpDelete(*gcpKey, *gcpProjectName, gcpVMName, gcpZone)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	gcpDelete.Flags().StringVar(&gcpVMName, "vm-name", defaults.DefaultGcpImageName, "vm name")
	gcpDelete.Flags().StringVar(&gcpZone, "zone", defaults.DefaultGcpZone, "gcp zone")

	return gcpDelete
}

func newGcpConsoleCmd(gcpKey, gcpProjectName *string) *cobra.Command {
	var gcpVMName, gcpZone string

	var gcpConsole = &cobra.Command{
		Use:   "console",
		Short: "connect to vm console gcp",
		Run: func(cmd *cobra.Command, args []string) {
			gcpClient, err := linuxkit.NewGCPClient(*gcpKey, *gcpProjectName)
			if err != nil {
				log.Fatalf("Unable to connect to GCP: %v", err)
			}
			if err := gcpClient.ConnectToInstanceSerialPort(gcpVMName, gcpZone); err != nil {
				log.Fatalf("ConnectToInstanceSerialPort: %s", err)
			}
		},
	}

	gcpConsole.Flags().StringVar(&gcpVMName, "vm-name", defaults.DefaultGcpImageName, "vm name")
	gcpConsole.Flags().StringVar(&gcpZone, "zone", defaults.DefaultGcpZone, "gcp zone")

	return gcpConsole
}

func newGcpLogCmd(gcpKey, gcpProjectName *string) *cobra.Command {
	var gcpVMName, gcpZone string
	var follow bool

	var gcpLog = &cobra.Command{
		Use:   "log",
		Short: "show vm console log from gcp",
		Run: func(cmd *cobra.Command, args []string) {
			gcpClient, err := linuxkit.NewGCPClient(*gcpKey, *gcpProjectName)
			if err != nil {
				log.Fatalf("Unable to connect to GCP: %v", err)
			}
			if err := gcpClient.GetInstanceSerialOutput(gcpVMName, gcpZone, follow); err != nil {
				log.Fatalf("GetInstanceSerialOutput: %s", err)
			}
		},
	}

	gcpLog.Flags().StringVar(&gcpVMName, "vm-name", defaults.DefaultGcpImageName, "vm name")
	gcpLog.Flags().StringVar(&gcpZone, "zone", defaults.DefaultGcpZone, "gcp zone")
	gcpLog.Flags().BoolVarP(&follow, "follow", "f", false, "follow log")

	return gcpLog
}

func newGcpGetIPCmd(gcpKey, gcpProjectName *string) *cobra.Command {
	var gcpVMName, gcpZone string

	var gcpGetIP = &cobra.Command{
		Use:   "get-ip",
		Short: "print IP of VM ",
		Run: func(cmd *cobra.Command, args []string) {
			gcpClient, err := linuxkit.NewGCPClient(*gcpKey, *gcpProjectName)
			if err != nil {
				log.Fatalf("Unable to connect to GCP: %v", err)
			}
			natIP, err := gcpClient.GetInstanceNatIP(gcpVMName, gcpZone)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(natIP)
		},
	}

	gcpGetIP.Flags().StringVar(&gcpVMName, "vm-name", defaults.DefaultGcpImageName, "vm name")
	gcpGetIP.Flags().StringVar(&gcpZone, "zone", defaults.DefaultGcpZone, "gcp zone")

	return gcpGetIP
}

func newGcpAddFirewallRule(gcpKey, gcpProjectName *string) *cobra.Command {
	var (
		gcpFirewallRuleName     string
		gcpFirewallRuleSources  []string
		gcpFirewallRulePriority int64
	)

	var gcpAddFirewallRule = &cobra.Command{
		Use:   "firewall",
		Short: "add firewall rule for access",
		Run: func(cmd *cobra.Command, args []string) {
			if gcpFirewallRuleSources == nil {
				log.Fatal("Please define source-range")
			}
			for ind, el := range gcpFirewallRuleSources {
				if !strings.Contains(el, "/") {
					gcpFirewallRuleSources[ind] = fmt.Sprintf("%s/32", el)
				}
			}
			gcpClient, err := linuxkit.NewGCPClient(*gcpKey, *gcpProjectName)
			if err != nil {
				log.Fatalf("Unable to connect to GCP: %v", err)
			}
			if err := gcpClient.DeleteFirewallAllowRule(gcpFirewallRuleName); err != nil {
				log.Warning(err)
			}
			if err := gcpClient.SetFirewallAllowRule(gcpFirewallRuleName, gcpFirewallRulePriority, gcpFirewallRuleSources); err != nil {
				log.Fatal(err)
			}
			log.Info("Rules added")
		},
	}

	gcpAddFirewallRule.Flags().StringVar(&gcpFirewallRuleName, "name", fmt.Sprintf("%s-rule", defaults.DefaultGcpImageName), "firewall rule name")
	gcpAddFirewallRule.Flags().StringSliceVar(&gcpFirewallRuleSources, "source-range", nil, "source ranges to allow")
	gcpAddFirewallRule.Flags().Int64Var(&gcpFirewallRulePriority, "priority", defaults.DefaultGcpRulePriority, "priority of firewall rule")

	return gcpAddFirewallRule
}

func newGcpCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var gcpProjectName, gcpKey string

	var gcpCmd = &cobra.Command{
		Use:   "gcp",
		Short: `Manage images and VMs in Google Cloud Platform`,
		Long:  `Manage images and VMs in Google Cloud Platform (you need to provide a key, set it in config in gcp.key or use a gcloud login)`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// gcpKey = cfg.Gcp.Key

			if gcpKey == "" {
				context, err := utils.ContextLoad()
				if err != nil { //we have no current context
					log.Warn(`You didn't specify the '--key' argument. Something might break`)
				} else {
					log.Warnf(`You didn't specify the '--key' argument, or set it wia 'eden config set %s --key gcp.key --value YOUR_PATH_TO_KEY'. Something might break`, context.Current)
				}
			}

			return nil
		},
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newGcpImageCmd(cfg, &gcpKey, &gcpProjectName),
				newGcpVMCmd(cfg, &gcpKey, &gcpProjectName),
				newGcpAddFirewallRule(&gcpKey, &gcpProjectName),
			},
		},
	}

	groups.AddTo(gcpCmd)

	gcpCmd.PersistentFlags().StringVarP(&gcpProjectName, "project", "p", defaults.DefaultGcpProjectName, "project name on gcp")
	gcpCmd.PersistentFlags().StringVarP(&gcpKey, "key", "k", "", "gcp key file")

	return gcpCmd
}
