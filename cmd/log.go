package cmd

import (
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

var outputFormatIds = map[types.OutputFormat][]string{
	types.OutputFormatLines: {"lines"},
	types.OutputFormatJSON:  {"json"},
}

func newLogCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var outputFormat types.OutputFormat

	var logCmd = &cobra.Command{
		Use:   "log [field:regexp ...]",
		Short: "Get logs from a running EVE device",
		Long: `
Scans the ADAM logs for correspondence with regular expressions requests to json fields.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: add outputFormat to code
			if err := openevec.EdenLog(cfg, outputFormat, args); err != nil {
				log.Fatalf("Log eden failed: %s", err)
			}
		},
	}

	logCmd.Flags().UintVar(&cfg.Runtime.LogTail, "tail", 0, "Show only last N lines")
	logCmd.Flags().StringSliceVarP(&cfg.Runtime.PrintFields, "out", "o", nil, "Fields to print. Whole message if empty.")
	logCmd.Flags().BoolVarP(&cfg.Runtime.Follow, "follow", "f", false, "Monitor changes in selected directory")

	logCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")
	return logCmd
}
