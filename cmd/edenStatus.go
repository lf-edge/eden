package cmd

import (
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "status of harness",
	Long:  `Status of harness.`,
	Run: func(cmd *cobra.Command, args []string) {
		statusAdam, err := utils.StatusAdam()
		if err != nil {
			log.Fatalf("cannot obtain status of adam: %s", err)
		}
		log.Printf("Adam status: %s", statusAdam)
		statusEServer, err := utils.StatusEServer(eserverPidFile)
		if err != nil {
			log.Fatalf("cannot obtain status of eserver: %s", err)
		}
		log.Printf("EServer status: %s", statusEServer)
		statusEVE, err := utils.StatusEVEQemu(evePidFile)
		if err != nil {
			log.Fatalf("cannot obtain status of EVE: %s", err)
		}
		log.Printf("EVE status: %s", statusEVE)
	},
}

func statusInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	statusCmd.Flags().StringVarP(&eserverPidFile, "eserver-pid", "", path.Join(currentPath, "dist", "eserver.pid"), "file with eserver pid")
	statusCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", path.Join(currentPath, "dist", "eve.pid"), "file with EVE pid")
}
