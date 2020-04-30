package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/adam"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
)

var adamCA string
var getConfig bool

var reconfCmd = &cobra.Command{
	Use:   "reconf <file>",
	Short: "reconf harness",
	Long:  `Reconf harness.`,
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			certsIP = viper.GetString("adam.ip")
			adamPort = viper.GetString("adam.port")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			adamCA = utils.ResolveAbsPath(viper.GetString("adam.ca"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var ctrl controller.Cloud = &controller.CloudCtx{Controller: &adam.Ctx{
			Dir:         adamDist,
			URL:         fmt.Sprintf("https://%s:%s", certsIP, adamPort),
			InsecureTLS: true,
		}}
		if len(adamCA) != 0 {
			ctrl = &controller.CloudCtx{Controller: &adam.Ctx{
				Dir:         adamDist,
				URL:         fmt.Sprintf("https://%s:%s", certsIP, adamPort),
				InsecureTLS: false,
				ServerCA:    adamCA,
			}}
		}
		devices, err := ctrl.DeviceList()
		if err != nil {
			log.Fatalf("DeviceList: %s", err)
		}
		for _, devID := range devices {
			devUUID, err := uuid.FromString(devID)
			if err != nil {
				log.Fatalf("uuidGet: %s", err)
			}
			if getConfig {
				data, err := ctrl.ConfigGet(devUUID)
				if err != nil {
					log.Fatalf("ConfigSet: %s", err)
				}
				if err = ioutil.WriteFile(args[0], []byte(data), 0755); err != nil {
					log.Fatalf("WriteFile: %s", err)
				}
			} else {
				data, err := ioutil.ReadFile(args[0])
				if err != nil {
					fmt.Println("File reading error:", err)
					return
				}
				if err = ctrl.ConfigSet(devUUID, data); err != nil {
					log.Fatalf("ConfigSet: %s", err)
				}
			}
			break
		}
	},
}

func reconfInit() {
	reconfCmd.Flags().StringVar(&config, "config", "", "path to config file")
	reconfCmd.Flags().BoolVar(&getConfig, "get", false, "get config")
}
