package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

var (
	adamRm    bool
	eserverRm bool
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop harness",
	Long:  `Stop harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveRemote = viper.GetBool("eve.remote")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := utils.StopAdam(adamRm); err != nil {
			log.Infof("cannot stop adam: %s", err)
		} else {
			log.Infof("adam stopped")
		}
		if err := utils.StopRedis(redisRm); err != nil {
			log.Infof("cannot stop redis: %s", err)
		} else {
			log.Infof("redis stopped")
		}
		if err := utils.StopEServer(eserverRm); err != nil {
			log.Infof("cannot stop eserver: %s", err)
		} else {
			log.Infof("eserver stopped")
		}
		if eveRemote {
			return
		}
		if err := utils.StopEVEQemu(evePidFile); err != nil {
			log.Infof("cannot stop EVE: %s", err)
		} else {
			log.Infof("EVE stopped")
		}
	},
}

func stopInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	stopCmd.Flags().BoolVarP(&adamRm, "adam-rm", "", false, "adam rm on stop")
	stopCmd.Flags().BoolVarP(&redisRm, "redis-rm", "", false, "redis rm on stop")
	stopCmd.Flags().BoolVarP(&eserverRm, "eserver-rm", "", false, "eserver rm on stop")
	stopCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file with EVE pid")
}
