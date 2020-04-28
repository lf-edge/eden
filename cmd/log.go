package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Get logs from a running EVE device",
	Long:  `
Scans the ADAM log files for correspondence with regular expressions requests to json fields. For example:

eden log file [field:regexp ...]
eden log -f directory [field:regexp ...]
`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			fmt.Printf("Error in get param 'follow'")
			return
		}

		q := make(map[string]string)

		for _, a := range args[1:] {
			s := strings.Split(a, ":")
			q[s[0]] = s[1]
		}

		if follow {
			// Monitoring of new files
			s, err := os.Stat(args[0]);
			if os.IsNotExist(err) {
				fmt.Println("Directory reading error:", err)
				return
			}
			if s.IsDir() {
				_ = elog.LogWatch(args[0], q, elog.HandleAll)
			} else {
				fmt.Printf("'%s' is not a directory.\n",args[0])
				return
			}
		} else {
			// Just look to selected file
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
		}
	},
}

func logInit() {
	logCmd.Flags().BoolP("follow", "f", false, "Monitor changes in selected directory")
}
