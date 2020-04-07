package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/elog"

	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "logs for eve",
	Long:  `Get logs from a running EVE device.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		data, err := ioutil.ReadFile(args[0])
		if err != nil {
			fmt.Println("File reading error", err)
			return
		}

		lb, err := elog.ParseLogBundle(data)
		if err != nil {
			fmt.Println("ParseLogBundle error", err)
			return
		}

		q := make(map[string]string)

		for _, a := range args[1:] {
			s := strings.Split(a, ":")
			q[s[0]] = s[1]
		}

		for _, n := range lb.Log {
			//fmt.Println(n.Content)
			s := string(n.Content)
			le, err := elog.ParseLogItem(s)
			if err != nil {
				fmt.Println("ParseLogItem error", err)
				return
			}
			if elog.LogItemFind(le, q) == 1 {
				elog.LogPrn(&le)
			}
		}
	},
}

func logInit() {

}
