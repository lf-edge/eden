package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get information reports from a running EVE device",
	Long:  `
Scans the ADAM Info files for correspondence with regular expressions requests to json fields. For example:

eden info file [field:regexp ...]
eden info -f directory [field:regexp ...]
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
				_ = einfo.InfoWatch(args[0], q, einfo.ZInfoFind, einfo.HandleAll, einfo.ZInfoDevSW)
			} else {
				fmt.Printf("'%s' is not a directory.\n",args[0])
				return
			}
		} else {
			// Just look to selected file
			data, err := ioutil.ReadFile(args[0])
			if err != nil {
				fmt.Println("File reading error:", err)
				return
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
		}
	},
}

func infoInit() {
	infoCmd.Flags().BoolP("follow", "f", false, "Monitor changes in selected directory")
}
