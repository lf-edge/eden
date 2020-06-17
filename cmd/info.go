package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/spf13/cobra"
)

var infoType string

var infoCmd = &cobra.Command{
	Use:   "info [field:regexp ...]",
	Short: "Get information reports from a running EVE device",
	Long: `
Scans the ADAM Info for correspondence with regular expressions requests to json fields.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			certsIP = viper.GetString("adam.ip")
			adamPort = viper.GetInt("adam.port")
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
		devFirst, err := ctrl.GetDeviceFirst()
		if err != nil {
			log.Fatalf("GetDeviceFirst error: %s", err)
		}
		devUUID := devFirst.GetID()
		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			fmt.Printf("Error in get param 'follow'")
			return
		}
		zInfoType, err := einfo.GetZInfoType(infoType)
		if err != nil {
			fmt.Printf("Error in get param 'type': %s", err)
			return
		}
		q := make(map[string]string)
		for _, a := range args[0:] {
			s := strings.Split(a, ":")
			q[s[0]] = s[1]
		}

		if follow {
			if err = ctrl.InfoChecker(devUUID, q, zInfoType, einfo.HandleAll, einfo.InfoNew, 0); err != nil {
				log.Fatalf("InfoChecker: %s", err)
			}
		} else {
			if err = ctrl.InfoLastCallback(devUUID, q, zInfoType, einfo.HandleAll); err != nil {
				log.Fatalf("InfoChecker: %s", err)
			}
		}
	},
}

func infoInit() {
	infoCmd.Flags().BoolP("follow", "f", false, "Monitor changes in selected directory")
	infoCmd.PersistentFlags().StringVarP(&infoType, "type", "", "all", fmt.Sprintf("info type (%s)", strings.Join(einfo.ListZInfoType(), ",")))
}
