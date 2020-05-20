package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "status of harness",
	Long:  `Status of harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eserverPidFile = utils.ResolveAbsPath(viper.GetString("eden.eserver.pid"))
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		statusAdam, err := utils.StatusAdam()
		if err != nil {
			log.Errorf("cannot obtain status of adam: %s", err)
		} else {
			fmt.Printf("Adam status: %s\n", statusAdam)
		}
		statusRedis, err := utils.StatusRedis()
		if err != nil {
			log.Errorf("cannot obtain status of redis: %s", err)
		} else {
			fmt.Printf("Redis status: %s\n", statusRedis)
		}
		statusEServer, err := utils.StatusEServer(eserverPidFile)
		if err != nil {
			log.Errorf("cannot obtain status of eserver: %s", err)
		} else {
			fmt.Printf("EServer status: %s\n", statusEServer)
		}
		statusEVE, err := utils.StatusEVEQemu(evePidFile)
		if err != nil {
			log.Errorf("cannot obtain status of EVE: %s", err)
		} else {
			fmt.Printf("EVE status: %s\n", statusEVE)
		}
	},
}

func statusInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	statusCmd.Flags().StringVarP(&eserverPidFile, "eserver-pid", "", filepath.Join(currentPath, "dist", "eserver.pid"), "file with eserver pid")
	statusCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, "dist", "eve.pid"), "file with EVE pid")
}
