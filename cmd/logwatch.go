package cmd

import (
	"strings"

	"github.com/lf-edge/eden/pkg/controller/elog"

	"github.com/spf13/cobra"
)

var logwatchCmd = &cobra.Command{
	Use:   "logwatch",
	Short: "retrieve log entries and keep watch",
	Long:  `Retrieve log entries and keep watch.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		q := make(map[string]string)

		for _, a := range args[1:] {
			s := strings.Split(a, ":")
			q[s[0]] = s[1]
		}

		_ = elog.LogWatch(args[0], q, elog.HandleAll)
	},
}

func logwatchInit() {
}
