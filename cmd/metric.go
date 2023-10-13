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
	var follow bool
	var printFields []string
	var metricTail uint

	var metricCmd = &cobra.Command{
		Use:   "metric [field:regexp ...]",
		Short: "Get metrics from a running EVE device",
		Long: `
Scans the ADAM metrics for correspondence with regular expressions requests to json fields.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdenMetric(outputFormat, follow, metricTail, printFields, args); err != nil {
				log.Fatalf("Metric eden failed: %s", err)
			}
		},
	}

	metricCmd.Flags().UintVar(&metricTail, "tail", 0, "Show only last N lines")
	metricCmd.Flags().StringSliceVarP(&printFields, "out", "o", nil, "Fields to print. Whole message if empty.")
	metricCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Monitor changes in selected metrics")

	metricCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")
	return metricCmd
}
