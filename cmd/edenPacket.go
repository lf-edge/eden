package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/packet"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	packetVMName      string
	packetProjectName string
	packetKey         string
	packetZone        string
	packetMachineType string
	packetIPXEUrl     string
)

var packetCmd = &cobra.Command{
	Use:   "packet",
	Short: `Manage VMs in Equinix Metal Platform`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := rootCmd.PersistentPreRunE(cmd, args); err != nil {
			return err
		}
		assignCobraToViper(cmd)
		if flag := cmd.Flag("key"); flag != nil {
			_ = viper.BindPFlag("packet.key", flag)
		}

		viperLoaded, err := utils.LoadConfigFile(configFile)

		if viperLoaded && err == nil {
			packetKey = viper.GetString("packet.key") //use variable from config
		}

		if packetKey == "" {
			context, err := utils.ContextLoad()
			if err != nil { //we have no current context
				log.Warn(`You didn't specify the '--key' argument. Something might break`)
			} else {
				log.Warnf(`You didn't specify the '--key' argument, or set it wia 'eden config set %s --key packet.key --value YOUR_PATH_TO_KEY'. Something might break`, context.Current)
			}
		}

		return nil
	},
}

var packetVMCmd = &cobra.Command{
	Use:   "vm",
	Short: `Manage VMs in packet`,
}

var packetRun = &cobra.Command{
	Use:   "run",
	Short: "run vm in packet",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}

		if viperLoaded {
			gcpvTPM = viper.GetBool("eve.tpm")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if packetIPXEUrl == "" {
			certsEVEIP = viper.GetString("adam.eve-ip")
			eServerPort := viper.GetString("eden.eserver.port")
			packetIPXEUrl = fmt.Sprintf("http://%s:%s/%s/%s/ipxe.efi.cfg", certsEVEIP, eServerPort, "eserver", configName)
			log.Debugf("ipxe-url is empty, will use default one: %s", packetIPXEUrl)
		}
		packetClient, err := packet.NewPacketClient(packetKey, packetProjectName)
		if err != nil {
			log.Fatalf("Unable to connect to create packet client: %v", err)
		}
		if err := packetClient.CreateInstance(packetVMName, packetZone, packetMachineType, packetIPXEUrl); err != nil {
			log.Fatalf("CreateInstance: %s", err)
		}
	},
}

var packetDelete = &cobra.Command{
	Use:   "delete",
	Short: "delete vm from packet",
	Run: func(cmd *cobra.Command, args []string) {
		packetClient, err := packet.NewPacketClient(packetKey, packetProjectName)
		if err != nil {
			log.Fatalf("Unable to connect to create packet client: %v", err)
		}
		if err := packetClient.DeleteInstance(packetVMName); err != nil {
			log.Fatalf("DeleteInstance: %s", err)
		}
	},
}

var packetGetIP = &cobra.Command{
	Use:   "get-ip",
	Short: "print IP of VM in packet",
	Run: func(cmd *cobra.Command, args []string) {
		packetClient, err := packet.NewPacketClient(packetKey, packetProjectName)
		if err != nil {
			log.Fatalf("Unable to connect to create packet client: %v", err)
		}
		natIP, err := packetClient.GetInstanceNatIP(packetVMName)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(natIP)
	},
}

func packetInit() {
	packetCmd.AddCommand(packetVMCmd)
	packetCmd.PersistentFlags().StringVarP(&packetProjectName, "project", "p", defaults.DefaultPacketProjectName, "project name on packet")
	packetCmd.PersistentFlags().StringVarP(&packetKey, "key", "k", "", "packet key file")
	packetVMCmd.AddCommand(packetRun)
	packetRun.Flags().StringVar(&packetVMName, "vm-name", defaults.DefaultVMName, "vm name")
	packetRun.Flags().StringVar(&packetZone, "zone", defaults.DefaultPacketZone, "packet zone")
	packetRun.Flags().StringVar(&packetMachineType, "machine-type", defaults.DefaultPacketMachineType, "packet machine type")
	packetRun.Flags().StringVar(&packetIPXEUrl, "ipxe-url", "", "packet ipxe url")
	packetVMCmd.AddCommand(packetDelete)
	packetDelete.Flags().StringVar(&packetVMName, "vm-name", defaults.DefaultVMName, "vm name")
	packetVMCmd.AddCommand(packetGetIP)
	packetGetIP.Flags().StringVar(&packetVMName, "vm-name", defaults.DefaultVMName, "vm name")
}
