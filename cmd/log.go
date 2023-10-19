package cmd

import (
	"github.com/lf-edge/eden/pkg/controller/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

var outputFormatIds = map[types.OutputFormat][]string{
	types.OutputFormatLines: {"lines"},
	types.OutputFormatJSON:  {"json"},
}

func newLogCmd() *cobra.Command {
	var outputFormat types.OutputFormat
	var follow bool
	var printFields []string
	var logTail uint

	var logCmd = &cobra.Command{
		Use:   "log [field:regexp ...]",
		Short: "Get logs from a running EVE device",
		Long:  ` Scans the ADAM logs for correspondence with regular expressions requests to json fields.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdenLog(outputFormat, follow, logTail, printFields, args); err != nil {
				log.Fatalf("Log eden failed: %s", err)
			}
		},
	}

	logCmd.Flags().UintVar(&logTail, "tail", 0, "Show only last N lines")
	logCmd.Flags().StringSliceVarP(&printFields, "out", "o", nil, "Fields to print. Whole message if empty.")
	logCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Monitor changes in selected directory")

	logCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")
	return logCmd
}
