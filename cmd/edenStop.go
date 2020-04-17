package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	PreRunE: func(cmd *cobra.Command, args []string) error {
		viperLoaded, err := loadViperConfig()
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eserverPidFile = viper.GetString("eserver-pid")
			evePidFile = viper.GetString("eve-pid")
			adamRm = viper.GetBool("adam-rm")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := utils.StopAdam(adamRm)
		if err != nil {
			log.Errorf("cannot stop adam: %s", err)
		}
		err = utils.StopEServer(eserverPidFile)
		if err != nil {
			log.Errorf("cannot stop eserver: %s", err)
		}
		err = utils.StopEVEQemu(evePidFile)
		if err != nil {
			log.Errorf("cannot stop EVE: %s", err)
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
	err = viper.BindPFlags(stopCmd.Flags())
	if err != nil {
		log.Fatal(err)
	}
	stopCmd.Flags().StringVar(&config, "config", "", "path to config file")
}
