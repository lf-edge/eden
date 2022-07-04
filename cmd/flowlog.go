package cmd

import (
	"fmt"
	"strings"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/eflowlog"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/flowlog"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thediveo/enumflag"
)

var netStatCmd = &cobra.Command{
	Use:   "netstat [field:regexp ...]",
	Short: "Get logs of network packets from a running EVE device",
	Long: `Scans the ADAM flow messages for correspondence with regular expressions to show network flow statistics
(TCP and UDP flows with IP addresses, port numbers, counters, whether dropped or accepted)`,
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
			log.Fatalf("Error in get param 'follow'")
		}

		q := make(map[string]string)

		for _, a := range args[0:] {
			s := strings.Split(a, ":")
			q[s[0]] = s[1]
		}

		handleFunc := func(le *flowlog.FlowMessage) bool {
			if printFields == nil {
				eflowlog.FlowLogPrn(le, outputFormat)
			} else {
				eflowlog.FlowLogItemPrint(le, printFields).Print()
			}
			return false
		}

		if logTail > 0 {
			if err = ctrl.FlowLogChecker(devUUID, q, handleFunc, eflowlog.FlowLogTail(logTail), 0); err != nil {
				log.Fatalf("FlowLogChecker: %s", err)
			}
		} else {
			if follow {
				// Monitoring of new files
				if err = ctrl.FlowLogChecker(devUUID, q, handleFunc, eflowlog.FlowLogNew, 0); err != nil {
					log.Fatalf("FlowLogChecker: %s", err)
				}
			} else {
				if err = ctrl.FlowLogLastCallback(devUUID, q, handleFunc); err != nil {
					log.Fatalf("FlowLogLastCallback: %s", err)
				}
			}
		}
	},
}

func netStatInit() {
	netStatCmd.Flags().UintVar(&logTail, "tail", 0, "Show only last N lines")
	netStatCmd.Flags().StringSliceVarP(&printFields, "out", "o", nil, "Fields to print. Whole message if empty.")
	netStatCmd.Flags().BoolP("follow", "f", false, "Monitor changes in selected directory")
	netStatCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")
}
