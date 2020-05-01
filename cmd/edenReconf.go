package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
)

var adamCA string
var getConfig bool

var reconfCmd = &cobra.Command{
	Use:   "reconf <file>",
	Short: "reconf EVE",
	Long:  `Reconf EVE.`,
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
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
		ctrl, err := controller.CloudPrepare()
		if err != nil {
			log.Fatalf("CloudPrepare: %s", err)
		}
		if err := ctrl.OnBoard(); err != nil {
			log.Fatalf("OnBoard: %s", err)
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
				log.Infof("File saved: %s", args[0])
			} else {
				data, err := ioutil.ReadFile(args[0])
				if err != nil {
					log.Fatalf("File reading error:", err)
					return
				}
				if err = ctrl.ConfigSet(devUUID, data); err != nil {
					log.Fatalf("ConfigSet: %s", err)
				}
				log.Infof("File loaded: %s", args[0])
			}
			break
		}
	},
}

func reconfInit() {
	reconfCmd.Flags().StringVar(&configFile, "config", "", "path to config file")
	reconfCmd.Flags().BoolVar(&getConfig, "get", false, "get config instead of set")
}
