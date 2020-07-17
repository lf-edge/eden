package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"

	"github.com/spf13/cobra"
)

var metricCmd = &cobra.Command{
	Use:   "metric [field:regexp ...]",
	Short: "Get metrics from a running EVE device",
	Long: `
Scans the ADAM metrics for correspondence with regular expressions requests to json fields.`,
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
		devFirst, err := ctrl.GetDeviceFirst()
		if err != nil {
			log.Fatalf("GetDeviceFirst error: %s", err)
		}
		devUUID := devFirst.GetID()
		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			log.Fatalf("Error in get param 'follow'")
		}

		q := make(map[string]string)

		for _, a := range args[0:] {
			s := strings.Split(a, ":")
			q[s[0]] = s[1]
		}

		if follow {
			// Monitoring of new files
			if err = ctrl.MetricChecker(devUUID, q, emetric.HandleAll, emetric.MetricNew, 0); err != nil {
				log.Fatalf("MetricChecker: %s", err)
			}
		} else {
			if err = ctrl.MetricLastCallback(devUUID, q, emetric.HandleAll); err != nil {
				log.Fatalf("MetricChecker: %s", err)
			}
		}
	},
}

func metricInit() {
	metricCmd.Flags().BoolP("follow", "f", false, "Monitor changes in selected metrics")
}
