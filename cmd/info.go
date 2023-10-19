package cmd

import (
	"github.com/lf-edge/eden/pkg/controller/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

func newInfoCmd() *cobra.Command {
	var outputFormat types.OutputFormat
	var infoTail uint
	var follow bool
	var printFields []string

	var infoCmd = &cobra.Command{
		Use:   "info [field:regexp ...]",
		Short: "Get information reports from a running EVE device",
		Long:  ` Scans the ADAM Info for correspondence with regular expressions requests to json fields.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdenInfo(outputFormat, infoTail, follow, printFields, args); err != nil {
				log.Fatal("Eden info failed ", err)
			}
		},
	}

	infoCmd.Flags().UintVar(&infoTail, "tail", 0, "Show only last N lines")
	infoCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Monitor changes in selected directory")
	infoCmd.Flags().StringSliceVarP(&printFields, "out", "o", nil, "Fields to print. Whole message if empty.")

	infoCmd.Flags().Var(
		enumflag.New(&outputFormat, "format", outputFormatIds, enumflag.EnumCaseInsensitive),
		"format",
		"Format to print logs, supports: lines, json")
	return infoCmd
}
