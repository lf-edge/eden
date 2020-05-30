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
		eveUUID := viper.GetString("eve.uuid")
		edenDir, err := utils.DefaultEdenDir()
		if err != nil {
			log.Fatal(err)
		}
		fi, err := os.Stat(filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", eveUUID)))
		if err != nil {
			fmt.Printf("EVE state: not onboarded\n")
		} else {
			size := fi.Size()
			if size > 0 {
				fmt.Printf("EVE state: registered\n")
			} else {
				fmt.Printf("EVE state: onboarding\n")
			}
		}
		fmt.Println()
		statusAdam, err := utils.StatusAdam()
		if err != nil {
			log.Errorf("cannot obtain status of adam: %s", err)
		} else {
			fmt.Printf("Adam status: %s\n", statusAdam)
			fmt.Printf("\tAdam is expected at https://%s:%d\n", viper.GetString("adam.ip"), viper.GetInt("adam.port"))
			fmt.Printf("\tFor local Adam you can run 'docker logs %s' to see logs\n", defaults.DefaultAdamContainerName)
		}
		statusRedis, err := utils.StatusRedis()
		if err != nil {
			log.Errorf("cannot obtain status of redis: %s", err)
		} else {
			fmt.Printf("Redis status: %s\n", statusRedis)
			fmt.Printf("\tRedis is expected at %s\n", viper.GetString("adam.redis.eden"))
			fmt.Printf("\tFor local Redis you can run 'docker logs %s' to see logs\n", defaults.DefaultRedisContainerName)
		}
		statusEServer, err := utils.StatusEServer(eserverPidFile)
		if err != nil {
			log.Errorf("cannot obtain status of EServer process: %s", err)
		} else {
			fmt.Printf("EServer process status: %s\n", statusEServer)
			fmt.Printf("\tEServer is expected at http://%s:%d from EVE\n", viper.GetString("eden.eserver.ip"), viper.GetInt("eden.eserver.port"))
			fmt.Printf("\tLogs for local EServer at: %s\n", utils.ResolveAbsPath("eserver.log"))
		}
		statusEVE, err := utils.StatusEVEQemu(evePidFile)
		if err != nil {
			log.Errorf("cannot obtain status of EVE process: %s", err)
		} else {
			fmt.Printf("EVE process status: %s\n", statusEVE)
			fmt.Printf("\tLogs for local EVE at: %s\n", utils.ResolveAbsPath("eve.log"))
		}
	},
}

func statusInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	statusCmd.Flags().StringVarP(&eserverPidFile, "eserver-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eserver.pid"), "file with eserver pid")
	statusCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file with EVE pid")
}
