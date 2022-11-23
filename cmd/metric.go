package cmd

import (
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

func newMetricCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var outputFormat types.OutputFormat
	var metricCmd = &cobra.Command{
		Use:   "metric [field:regexp ...]",
		Short: "Get metrics from a running EVE device",
		Long: `
Scans the ADAM metrics for correspondence with regular expressions requests to json fields.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Add ouputFormat to the func
			if err := openevec.EdenMetric(cfg, outputFormat, args); err != nil {
				log.Fatalf("Metric eden failed: %s", err)
			}
		},
	}

	metricCmd.Flags().UintVar(&cfg.Runtime.MetricTail, "tail", 0, "Show only last N lines")
	metricCmd.Flags().StringSliceVarP(&cfg.Runtime.PrintFields, "out", "o", nil, "Fields to print. Whole message if empty.")
	metricCmd.Flags().BoolVarP(&cfg.Runtime.Follow, "follow", "f", false, "Monitor changes in selected metrics")

	metricCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")
	return metricCmd
}
