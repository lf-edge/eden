package cmd

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/spf13/cobra"
)

var infoType string

var infoCmd = &cobra.Command{
	Use:   "info <directory> [field:regexp ...]",
	Short: "Get information reports from a running EVE device",
	Long: `
Scans the ADAM Info files for correspondence with regular expressions requests to json fields.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			fmt.Printf("Error in get param 'follow'")
			return
		}
		zInfoType, err := einfo.GetZInfoType(infoType)
		if err != nil {
			fmt.Printf("Error in get param 'type': %s", err)
			return
		}
		q := make(map[string]string)
		for _, a := range args[1:] {
			s := strings.Split(a, ":")
			q[s[0]] = s[1]
		}

		if follow {
			// Monitoring of new files
			s, err := os.Stat(args[0])
			if os.IsNotExist(err) {
				fmt.Println("Directory reading error:", err)
				return
			}
			if s.IsDir() {
				pathBuilder := func(devUUID uuid.UUID) (dir string) {
					return args[0]
				}
				loader := einfo.FileLoader(pathBuilder)
				_ = loader.InfoWatch(q, einfo.ZInfoFind, einfo.HandleAll, zInfoType, 0)
			} else {
				fmt.Printf("'%s' is not a directory.\n", args[0])
				return
			}
		} else {
			// Just look to selected direcory
			var files []string

			root := args[0]
			err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				files = append(files, path)
				return nil
			})
			if err != nil {
				panic(err)
			}
			for _, file := range files[1:] {
				data, err := ioutil.ReadFile(file)
				if err != nil {
					fmt.Println("File reading error:", err)
					return
				}

				im, err := einfo.ParseZInfoMsg(data)
				if err != nil {
					fmt.Println("ParseZInfoMsg error", err)
					return
				}

				ds := einfo.ZInfoFind(&im, q, zInfoType)
				if ds != nil {
					einfo.ZInfoPrn(&im, ds, zInfoType)
				}
			}
		}
	},
}

func infoInit() {
	infoCmd.Flags().BoolP("follow", "f", false, "Monitor changes in selected directory")
	infoCmd.PersistentFlags().StringVarP(&infoType, "type", "", "all", fmt.Sprintf("info type (%s)", strings.Join(einfo.ListZInfoType(), ",")))
}
