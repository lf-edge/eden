package cmd

import (
	"strings"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/spf13/cobra"
)

var infowatchCmd = &cobra.Command{
	Use:   "infowatch",
	Short: "retrieve and parse information reports",
	Long:  `Retrieve and parse information reports.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		q := make(map[string]string)

		for _, a := range args[1:] {
			s := strings.Split(a, ":")
			q[s[0]] = s[1]
		}

		_ = einfo.InfoWatch(args[0], q, einfo.ZInfoFind, einfo.HandleAll, einfo.ZInfoDevSW)
	},
}

func infowatchInit() {
}
