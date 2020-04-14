package cmd

import (
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path"
)

var (
	adamRm bool
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop harness",
	Long:  `Stop harness.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := utils.StopAdam(adamRm)
		if err != nil {
			log.Printf("cannot stop adam: %s", err)
		}
		err = utils.StopEServer(eserverPidFile)
		if err != nil {
			log.Printf("cannot stop eserver: %s", err)
		}
		err = utils.StopEVEQemu(evePidFile)
		if err != nil {
			log.Printf("cannot stop EVE: %s", err)
		}
	},
}

func stopInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	stopCmd.Flags().BoolVarP(&adamRm, "adam-rm", "", false, "adam rm on stop")
	stopCmd.Flags().StringVarP(&eserverPidFile, "eserver-pid", "", path.Join(currentPath, "dist", "eserver.pid"), "file with eserver pid")
	stopCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", path.Join(currentPath, "dist", "eve.pid"), "file with EVE pid")
}
