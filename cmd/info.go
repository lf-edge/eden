package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/spf13/cobra"
)

var printFields []string

var infoCmd = &cobra.Command{
	Use:   "info [field:regexp ...]",
	Short: "Get information reports from a running EVE device",
	Long: `
Scans the ADAM Info for correspondence with regular expressions requests to json fields.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
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
		devFirst, err := ctrl.GetDeviceCurrent()
		if err != nil {
			log.Fatalf("GetDeviceCurrent error: %s", err)
		}
		devUUID := devFirst.GetID()
		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			fmt.Printf("Error in get param 'follow'")
			return
		}
		q := make(map[string]string)
		for _, a := range args[0:] {
			s := strings.Split(a, ":")
			q[s[0]] = s[1]
		}

		handleInfo := func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
			if printFields == nil {
				einfo.ZInfoPrn(im, ds)
			} else {
				einfo.ZInfoPrint(im, printFields).Print()
			}
			return false
		}

		if follow {
			if err = ctrl.InfoChecker(devUUID, q, handleInfo, einfo.InfoNew, 0); err != nil {
				log.Fatalf("InfoChecker: %s", err)
			}
		} else {
			if err = ctrl.InfoLastCallback(devUUID, q, handleInfo); err != nil {
				log.Fatalf("InfoChecker: %s", err)
			}
		}
	},
}

func infoInit() {
	infoCmd.Flags().BoolP("follow", "f", false, "Monitor changes in selected directory")
	infoCmd.Flags().StringSliceVarP(&printFields, "out", "o", nil, "Fields to print. Whole message if empty.")
}
