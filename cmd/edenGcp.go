package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/linuxkit"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

var (
	gcpVMName      string
	gcpImageName   string
	gcpProjectName string
	gcpKey         string
	gcpBucketName  string
	gcpZone        string
	gcpMachineType string

	gcpFirewallRuleName    string
	gcpFirewallRuleSources []string
)

var gcpCmd = &cobra.Command{
	Use:   "gcp",
	Short: `Manage images and VMs in Google Cloud Platform`,
	Long:  `Manage images and VMs in Google Cloud Platform (you need to provide a key, set it in config in gcp.key or use a gcloud login)`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)

		viperLoaded, err := utils.LoadConfigFile(configFile)

		if viperLoaded && err == nil {
			gcpKey = viper.GetString("gcp.key") //use variable from config
		}

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

var gcpImageCmd = &cobra.Command{
	Use:   "image",
	Short: `Manage images in gcp`,
}

var gcpVMCmd = &cobra.Command{
	Use:   "vm",
	Short: `Manage VMs in gcp`,
}

var gcpImageList = &cobra.Command{
	Use:   "list",
	Short: "list images uploaded to gcp",
	Long:  `Show list of images from gcp project.`,
	Run: func(cmd *cobra.Command, args []string) {
		gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
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

var gcpImageUpload = &cobra.Command{
	Use:   "upload",
	Short: "upload image to gcp",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}

		if viperLoaded {
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
		if err != nil {
			log.Fatalf("Unable to connect to GCP: %v", err)
		}
		fileName := fmt.Sprintf("%s.img.tar.gz", gcpImageName)
		if err := gcpClient.UploadFile(eveImageFile, fileName, gcpBucketName, false); err != nil {
			log.Fatalf("Error copying to Google Storage: %v", err)
		}
		err = gcpClient.CreateImage(gcpImageName, "https://storage.googleapis.com/"+gcpBucketName+"/"+fileName, "", true, true)
		if err != nil {
			log.Fatalf("Error creating Google Compute Image: %v", err)
		}
	},
}

var gcpImageDelete = &cobra.Command{
	Use:   "delete",
	Short: "delete image from gcp",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)

		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}

		if viperLoaded {
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
		if err != nil {
			log.Fatalf("Unable to connect to GCP: %v", err)
		}
		fileName := fmt.Sprintf("%s.img.tar.gz", gcpImageName)
		err = gcpClient.DeleteImage(gcpImageName)
		if err != nil {
			log.Fatalf("Error in delete of Google Compute Image: %v", err)
		}
		if err := gcpClient.RemoveFile(fileName, gcpBucketName); err != nil {
			log.Fatalf("Error id delete from Google Storage: %v", err)
		}
	},
}

var gcpRun = &cobra.Command{
	Use:   "run",
	Short: "run vm in gcp",
	Run: func(cmd *cobra.Command, args []string) {
		gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
		if err != nil {
			log.Fatalf("Unable to connect to GCP: %v", err)
		}
		if err := gcpClient.CreateInstance(gcpVMName, gcpImageName, gcpZone, gcpMachineType, nil, nil, true, true); err != nil {
			log.Fatal(err)
		}
	},
}

var gcpDelete = &cobra.Command{
	Use:   "delete",
	Short: "delete vm from gcp",
	Run: func(cmd *cobra.Command, args []string) {
		gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
		if err != nil {
			log.Fatalf("Unable to connect to GCP: %v", err)
		}
		if err := gcpClient.DeleteInstance(gcpVMName, gcpZone, true); err != nil {
			log.Fatalf("")
		}
	},
}

var gcpConsole = &cobra.Command{
	Use:   "console",
	Short: "connect to vm console gcp",
	Run: func(cmd *cobra.Command, args []string) {
		gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
		if err != nil {
			log.Fatalf("Unable to connect to GCP: %v", err)
		}
		if err := gcpClient.ConnectToInstanceSerialPort(gcpVMName, gcpZone); err != nil {
			log.Fatalf("")
		}
	},
}

var gcpGetIP = &cobra.Command{
	Use:   "get-ip",
	Short: "print IP of VM ",
	Run: func(cmd *cobra.Command, args []string) {
		gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
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
		gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
		if err != nil {
			log.Fatalf("Unable to connect to GCP: %v", err)
		}
		if err := gcpClient.SetFirewallAllowRule(gcpFirewallRuleName, gcpFirewallRuleSources); err != nil {
			log.Fatal(err)
		}
		log.Info("Rules added")
	},
}

func gcpInit() {
	gcpCmd.AddCommand(gcpImageCmd)
	gcpCmd.AddCommand(gcpVMCmd)
	gcpCmd.PersistentFlags().StringVarP(&gcpProjectName, "project", "p", defaults.DefaultGcpProjectName, "project name on gcp")
	gcpCmd.PersistentFlags().StringVarP(&gcpKey, "key", "k", "", "gcp key file")
	gcpImageCmd.AddCommand(gcpImageList)
	gcpImageCmd.AddCommand(gcpImageUpload)
	gcpImageUpload.Flags().StringVar(&gcpImageName, "image-name", defaults.DefaultGcpImageName, "image name")
	gcpImageUpload.Flags().StringVar(&eveImageFile, "image-file", "", "image file to upload")
	gcpImageUpload.Flags().StringVar(&gcpBucketName, "bucket-name", defaults.DefaultGcpBucketName, "bucket name to upload into")
	gcpImageCmd.AddCommand(gcpImageDelete)
	gcpImageDelete.Flags().StringVar(&gcpImageName, "image-name", defaults.DefaultGcpImageName, "image name")
	gcpImageDelete.Flags().StringVar(&gcpBucketName, "bucket-name", defaults.DefaultGcpBucketName, "bucket name to upload into")
	gcpVMCmd.AddCommand(gcpRun)
	gcpRun.Flags().StringVar(&gcpImageName, "image-name", defaults.DefaultGcpImageName, "image name")
	gcpRun.Flags().StringVar(&gcpVMName, "vm-name", defaults.DefaultGcpImageName, "vm name")
	gcpRun.Flags().StringVar(&gcpZone, "zone", defaults.DefaultGcpZone, "gcp zone")
	gcpRun.Flags().StringVar(&gcpMachineType, "machine-type", defaults.DefaultGcpMachineType, "gcp machine type")
	gcpVMCmd.AddCommand(gcpDelete)
	gcpDelete.Flags().StringVar(&gcpVMName, "vm-name", defaults.DefaultGcpImageName, "vm name")
	gcpDelete.Flags().StringVar(&gcpZone, "zone", defaults.DefaultGcpZone, "gcp zone")
	gcpVMCmd.AddCommand(gcpConsole)
	gcpConsole.Flags().StringVar(&gcpVMName, "vm-name", defaults.DefaultGcpImageName, "vm name")
	gcpConsole.Flags().StringVar(&gcpZone, "zone", defaults.DefaultGcpZone, "gcp zone")
	gcpVMCmd.AddCommand(gcpGetIP)
	gcpGetIP.Flags().StringVar(&gcpVMName, "vm-name", defaults.DefaultGcpImageName, "vm name")
	gcpGetIP.Flags().StringVar(&gcpZone, "zone", defaults.DefaultGcpZone, "gcp zone")
	gcpCmd.AddCommand(gcpAddFirewallRule)
	gcpAddFirewallRule.Flags().StringVar(&gcpFirewallRuleName, "name", fmt.Sprintf("%s-rule", defaults.DefaultGcpImageName), "firewall rule name")
	gcpAddFirewallRule.Flags().StringSliceVar(&gcpFirewallRuleSources, "source-range", nil, "source ranges to allow")
}
