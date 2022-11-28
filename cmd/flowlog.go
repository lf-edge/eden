package cmd

import (
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

func newNetStatCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var outputFormat types.OutputFormat
	var netStatCmd = &cobra.Command{
		Use:   "netstat [field:regexp ...]",
		Short: "Get logs of network packets from a running EVE device",
		Long: `Scans the ADAM flow messages for correspondence with regular expressions to show network flow statistics
(TCP and UDP flows with IP addresses, port numbers, counters, whether dropped or accepted)`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.EdenNetStat(cfg, outputFormat, args); err != nil {
				log.Fatalf("Setup eden failed: %s", err)
			}
		},
	}

	netStatCmd.Flags().UintVar(&cfg.Runtime.LogTail, "tail", 0, "Show only last N lines")
	netStatCmd.Flags().StringSliceVarP(&cfg.Runtime.PrintFields, "out", "o", nil, "Fields to print. Whole message if empty.")
	netStatCmd.Flags().BoolVarP(&cfg.Runtime.Follow, "follow", "f", false, "Monitor changes in selected directory")
	netStatCmd.Flags().Var(enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive), "format", "Format to print logs, supports: lines, json")

	return netStatCmd
}
