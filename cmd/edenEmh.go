package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/linuxkit"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"encoding/json"
)

var (
	emhProjectID		string
	emhPortName			string
	emhPlan				string
	emhPortID			string
	emhNetworkID		string
	emhDeviceID			string
	emhFacility			string
	emhOS				string
	emhIpxeURL			string
	emhEVETag			string
	emhPrefixDeviceName	string
)

var emhCmd = &cobra.Command{
	Use:   "emh",
	Short: `Manage devices in Equinix Metal Hosting`,
	Long:  `Manage devices in Equinix Metal Hosting
			(you need to provide a api key, set it by export PACKET_AUTH_TOKEN=<your's token'>)`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := rootCmd.PersistentPreRunE(cmd, args); err != nil {
			return err
		}
		assignCobraToViper(cmd)

		viperLoaded, err := utils.LoadConfigFile(configFile)

		if viperLoaded && err == nil {
			emhEVETag = viper.GetString("eve.tag") //use variable from config
		}
		return nil
	},
}

var emhDeviceCmd = &cobra.Command{
	Use:   "device",
	Short: `Manage devices in EMH`,
}

var emhDevicePortCmd = &cobra.Command{
	Use:   "port",
	Short: `Read eth ports configurations on EMH switches per device`,
}

var emhPortCmd = &cobra.Command{
	Use:   "port",
	Short: `Manage ethernet ports configurations on EMH switches`,
}

var emhCreateDevice = &cobra.Command{
	Use:   "create",
	Short: "Create device in EMH",
	Long:  `Create device in Equinix Metal Hosting`,
	Run: func(cmd *cobra.Command, args []string) {
		emhClient, err := linuxkit.NewEMHClient()
		if err != nil {
			log.Fatalf("Unable to connect to EMH: %v", err)
		}
		device, err := emhClient.CreateDevice(emhProjectID, emhPrefixDeviceName + "test-eden-eve-" + emhEVETag + "-" + emhFacility, emhFacility, emhPlan, emhOS, emhIpxeURL)
		if err != nil {
			log.Fatalf("Unable to create device in EMH: %v", err)
		}
		deviceJSON, err := json.Marshal(device)
		if err != nil {
			log.Fatalf("Failed get device as json: %v", err)
		}
		fmt.Println(string(deviceJSON))
	},
}

var emhGetDevice = &cobra.Command{
	Use:   "get",
	Short: "Get device details",
	Long:  `Get device details`,
	Run: func(cmd *cobra.Command, args []string) {
		emhClient, err := linuxkit.NewEMHClient()
		if err != nil {
			log.Fatalf("Unable to connect to EMH: %v", err)
		}
		device, err := emhClient.GetDevice(emhDeviceID)
		if err != nil {
			log.Fatalf("Unable to get device: %v", err)
		}
		deviceJSON, err := json.Marshal(device)
		if err != nil {
			log.Fatalf("Failed get device as json: %v", err)
		}
		fmt.Println(string(deviceJSON))
	},
}

var emhDeleteDevice = &cobra.Command{
	Use:   "delete",
	Short: "Delete device",
	Run: func(cmd *cobra.Command, args []string) {
		emhClient, err := linuxkit.NewEMHClient()
		if err != nil {
			log.Fatalf("Unable to connect to EMH: %v", err)
		}
		err = emhClient.DeleteDevice(emhDeviceID)
		if err != nil {
			log.Fatalf("Unable to delete device: %v", err)
		}
	},
}

var emhGetDevicePort = &cobra.Command{
	Use:   "get",
	Short: "Get device network port",
	Long:  `Get device network port`,
	Run: func(cmd *cobra.Command, args []string) {
		emhClient, err := linuxkit.NewEMHClient()
		if err != nil {
			log.Fatalf("Unable to connect to EMH: %v", err)
		}
		port, err := emhClient.GetDevicePortByName(emhDeviceID, emhPortName)
		if err != nil {
			log.Fatalf(err.Error())
		}
		portJSON, err := json.Marshal(port)
		if err != nil {
			log.Fatalf("Failed get port as json: %v", err)
		}
		fmt.Println(string(portJSON))
	},
}

var emhAssignPort = &cobra.Command{
	Use:   "assign",
	Short: "Add a VLAN to a port",
	Long:  `Add a VLAN to a port`,
	Run: func(cmd *cobra.Command, args []string) {
		emhClient, err := linuxkit.NewEMHClient()
		if err != nil {
			log.Fatalf("Unable to connect to EMH: %v", err)
		}
		err = emhClient.AssignPort(emhPortID, emhNetworkID)
		if err != nil {
			log.Fatalf(err.Error())
		}
	},
}

var emhDisbondPort = &cobra.Command{
	Use:   "disbond",
	Short: "Disable bonding for port",
	Long:  `Disable bonding for port`,
	Run: func(cmd *cobra.Command, args []string) {
		emhClient, err := linuxkit.NewEMHClient()
		if err != nil {
			log.Fatalf("Unable to connect to EMH: %v", err)
		}
		err = emhClient.DisbondPort(emhPortID)
		if err != nil {
			log.Fatalf(err.Error())
		}
	},
}

var emhAssignPortNative = &cobra.Command{
	Use:   "assign-native",
	Short: "Assign a virtual network to the port as a \"native VLAN\"",
	Long:  `Assign a virtual network to the port as a "native VLAN"`,
	Run: func(cmd *cobra.Command, args []string) {
		emhClient, err := linuxkit.NewEMHClient()
		if err != nil {
			log.Fatalf("Unable to connect to EMH: %v", err)
		}
		err = emhClient.AssignNativePort(emhPortID, emhNetworkID)
		if err != nil {
			log.Fatalf(err.Error())
		}
	},
}

func emhInit() {
	// device
	emhCmd.AddCommand(emhDeviceCmd)
	// device -> create
	emhDeviceCmd.AddCommand(emhCreateDevice)
	emhCreateDevice.Flags().StringVarP(&emhProjectID, "project-id", "p", "", "project id")
	emhCreateDevice.Flags().StringVarP(&emhFacility, "facility","f", "", "facility code")
	emhCreateDevice.Flags().StringVarP(&emhPrefixDeviceName, "name-prefix","n", "", "device name prefix")
	emhCreateDevice.Flags().StringVarP(&emhOS, "operating-system", "o", "", "operation system")
	emhCreateDevice.Flags().StringVarP(&emhIpxeURL, "ipxe","i", "", "ipxe cfg url")
	emhCreateDevice.Flags().StringVarP(&emhPlan, "plan", "P", "", "plan (configuration)")
	_ = emhCreateDevice.MarkFlagRequired("project-id")
	_ = emhCreateDevice.MarkFlagRequired("facility")
	_ = emhCreateDevice.MarkFlagRequired("plan")
	_ = emhCreateDevice.MarkFlagRequired("operating-system")
	// device -> get
	emhDeviceCmd.AddCommand(emhGetDevice)
	emhGetDevice.Flags().StringVarP(&emhDeviceID, "device-id","i", "", "device id")
	_ = emhGetDevice.MarkFlagRequired("device-id")
	// device -> delete
	emhDeviceCmd.AddCommand(emhDeleteDevice)
	emhDeleteDevice.Flags().StringVarP(&emhDeviceID, "device-id","i", "", "device id")
	_ = emhGetDevice.MarkFlagRequired("device-id")
	// device -> port
	emhDeviceCmd.AddCommand(emhDevicePortCmd)
	// device -> port -> get
	emhDevicePortCmd.AddCommand(emhGetDevicePort)
	emhGetDevicePort.Flags().StringVarP(&emhDeviceID, "device-id","i", "", "device id")
	emhGetDevicePort.Flags().StringVarP(&emhPortName, "port-name","p", "", "device port name")
	_ = emhGetDevicePort.MarkFlagRequired("device-id")
	_ = emhGetDevicePort.MarkFlagRequired("port-name")
	// port
	emhCmd.AddCommand(emhPortCmd)
	// port -> assign
	emhPortCmd.AddCommand(emhAssignPort)
	emhAssignPort.Flags().StringVarP(&emhPortID, "port-id", "i", "", "port id")
	emhAssignPort.Flags().StringVarP(&emhPortName, "network-id", "n", "", "network id")
	_ = emhAssignPort.MarkFlagRequired("port-id")
	_ = emhAssignPort.MarkFlagRequired("network-id")
	// port -> disbond
	emhPortCmd.AddCommand(emhDisbondPort)
	emhDisbondPort.Flags().StringVarP(&emhPortID,"port-id", "i", "", "port id")
	_ = emhAssignPort.MarkFlagRequired("port-id")
	// port -> assign-native
	emhPortCmd.AddCommand(emhAssignPortNative)
	emhAssignPortNative.Flags().StringVarP(&emhPortID, "port-id", "i", "", "port id")
	emhAssignPortNative.Flags().StringVarP(&emhPortName, "network-id", "n", "", "network id")
	_ = emhAssignPortNative.MarkFlagRequired("port-id")
	_ = emhAssignPortNative.MarkFlagRequired("network-id")
}