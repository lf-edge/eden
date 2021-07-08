package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/linuxkit"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
)

var (
	asbddsModel   string
	asbddsIPXELink string
	asbddsDeviceUUID string
)

var asbddsCmd = &cobra.Command{
	Use:   "asbdds",
	Short: `Manage devices in ARM Single Board Devices Deployment System`,
}

var asbddsDeviceCmd = &cobra.Command{
	Use:   "device",
	Short: `Manage devices`,
}

var asbddsDeviceCreateCmd = &cobra.Command{
	Use:   "create",
	Short: `Create device`,
	Run: func(cmd *cobra.Command, args []string) {
		asbddsClient, err := linuxkit.NewASBDDSClient()
		if err != nil {
			log.Fatalf("ASBDS: unable to create rest client: %v", err)
		}
		resp, err := asbddsClient.CreateDevice(asbddsModel, asbddsIPXELink)
		if err != nil {
			log.Fatalf("ASBDS: unable to create device:  %v", err)
		}
		jsonStr,_ := resp.MarshalJSON()
		fmt.Println(string(jsonStr[:]))
	},
}

var asbddsDeviceDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete device`,
	Run: func(cmd *cobra.Command, args []string) {
		asbddsClient, err := linuxkit.NewASBDDSClient()
		if err != nil {
			log.Fatalf("ASBDS: unable to create rest client: %v", err)
		}
		resp, err := asbddsClient.DeleteDevice(asbddsDeviceUUID)
		if err != nil {
			log.Fatalf("ASBDS: unable to delete device: %v", err)
		}
		jsonStr,_ := resp.MarshalJSON()
		fmt.Println(string(jsonStr[:]))
	},
}
func asbddsInit() {
	// device
	asbddsCmd.AddCommand(asbddsDeviceCmd)
	// device -> create
	asbddsDeviceCmd.AddCommand(asbddsDeviceCreateCmd)
	asbddsDeviceCreateCmd.Flags().StringVarP(&asbddsModel, "model", "m","", "device model")
	asbddsDeviceCreateCmd.Flags().StringVarP(&asbddsIPXELink, "ipxe_url", "i","", "link to ipxe config")
	_ = asbddsDeviceCreateCmd.MarkFlagRequired("model")
	_ = asbddsDeviceCreateCmd.MarkFlagRequired("ipxe_url")
	// device -> delete
	asbddsDeviceCmd.AddCommand(asbddsDeviceDeleteCmd)
	asbddsDeviceDeleteCmd.Flags().StringVarP(&asbddsDeviceUUID, "uuid", "i","", "device uuid")
	_ = asbddsDeviceDeleteCmd.MarkFlagRequired("uuid")
}