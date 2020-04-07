package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/einfo"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "information for eve",
	Long:  `Get information reports from a running EVE device.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		data, err := ioutil.ReadFile(args[0])
		if err != nil {
			fmt.Println("File reading error", err)
			return
		}

		q := make(map[string]string)
		for _, a := range args[1:] {
			s := strings.Split(a, ":")
			q[s[0]] = s[1]
		}

		im, err := einfo.ParseZInfoMsg(data)
		if err != nil {
			fmt.Println("ParseZInfoMsg error", err)
			return
		}

		ds := einfo.ZInfoFind(&im, q, einfo.ZInfoDevSW)
		if ds != nil {
			einfo.ZInfoPrn(&im, ds, einfo.ZInfoDevSW)
		}
	},
}

func infoInit() {

}
