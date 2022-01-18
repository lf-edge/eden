package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/Insei/rolgo"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"path"
)

var (
	rolProjectID    string
	rolRentName     string
	rolModel        string
	rolManufacturer string
	rolRentID		string
	rolIPXEUrl		string
)

var rolCmd = &cobra.Command{
	Use:   "rol",
	Short: `Manage devices in Rack Of Labs`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := rootCmd.PersistentPreRunE(cmd, args); err != nil {
			return err
		}
		assignCobraToViper(cmd)

		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return err
		}
		return nil
	},
}

var rolRentCmd = &cobra.Command{
	Use:   "rent",
	Short: "Manage device rents",
	Long:  `Manage device rents`,
}

var createRentCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new device rent",
	Long:  `Create a new device rent`,

	Run: func(cmd *cobra.Command, args []string) {
		assignCobraToViper(cmd)

		client, err := rolgo.NewClient()
		if err != nil {
			log.Fatalf(err.Error())
		}
		if rolIPXEUrl == "" {
			certsEVEIP = viper.GetString("adam.eve-ip")
			eServerPort := viper.GetString("eden.eserver.port")
			configPrefix := configName
			if configName == defaults.DefaultContext {
				configPrefix = ""
			}
			rolIPXEUrl = fmt.Sprintf("http://%s:%s/%s/ipxe.efi.cfg", certsEVEIP, eServerPort, path.Join("eserver", configPrefix))
			log.Debugf("ipxe-url is empty, will use default one: %s", packetIPXEUrl)
		}
		r := &rolgo.DeviceRentCreateRequest{Model: rolModel, Manufacturer: rolManufacturer, Name: rolRentName,
			IpxeUrl: rolIPXEUrl}
		rent, err := client.Rents.Create(rolProjectID, r)
		if err == nil {
			fmt.Println(rent.Id)
		} else {
			log.Fatalf("unable to create device rent: %v", err)
		}

	},
}

var getRentCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the device rent",
	Long:  `Get the device rent`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rolgo.NewClient()
		if err != nil {
			log.Fatalf(err.Error())
		}
		rent, err := client.Rents.Get(rolProjectID, rolRentID)
		if err != nil {
			log.Fatalf(err.Error())
		}
		rentJSON, err := json.Marshal(rent)
		if err != nil {
			log.Fatalf(err.Error())
		}
		fmt.Println(string(rentJSON))
	},
}

var closeRentCmd = &cobra.Command{
	Use:   "close",
	Short: "Close the device rent",
	Long:  `Close the device rent`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := rolgo.NewClient()
		if err != nil {
			log.Fatalf(err.Error())
		}
		err = client.Rents.Release(rolProjectID, rolRentID)
		if err != nil {
			log.Fatalf(err.Error())
		}
	},
}

func rolInit() {
	// rol -> rent
	rolCmd.AddCommand(rolRentCmd)
	rolRentCmd.PersistentFlags().StringVarP(&rolProjectID, "project-id", "p", "", "project id")
	_ = rolRentCmd.MarkFlagRequired("project-id")
	// rol -> rent -> create
	createRentCmd.Flags().StringVarP(&rolRentName, "name", "n", "", "rent name")
	createRentCmd.Flags().StringVar(&rolModel, "model", "", "device model")
	createRentCmd.Flags().StringVarP(&rolManufacturer, "manufacturer", "m", "", "device manufacturer")
	createRentCmd.Flags().StringVarP(&rolIPXEUrl, "ipxe-cfg-url", "i", "", "url to IPXE cfg file")
	_ = createRentCmd.MarkFlagRequired("name")
	_ = createRentCmd.MarkFlagRequired("model")
	_ = createRentCmd.MarkFlagRequired("manufacturer")
	rolRentCmd.AddCommand(createRentCmd)
	// rol -> rent -> close
	closeRentCmd.Flags().StringVarP(&rolRentID, "id", "i", "", "rent id")
	_ = closeRentCmd.MarkFlagRequired("id")
	rolRentCmd.AddCommand(closeRentCmd)
	// rol -> rent -> get
	getRentCmd.Flags().StringVarP(&rolRentID, "id", "i", "", "rent id")
	_ = getRentCmd.MarkFlagRequired("id")
	rolRentCmd.AddCommand(getRentCmd)
}
