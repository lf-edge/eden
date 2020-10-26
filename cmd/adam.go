package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var (
	adamTag            string
	adamRemoteRedisURL string
	adamRemoteRedis    bool
)

var adamCmd = &cobra.Command{
	Use: "adam",
}

var startAdamCmd = &cobra.Command{
	Use:   "start",
	Short: "start adam",
	Long:  `Start adam.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			adamTag = viper.GetString("adam.tag")
			adamPort = viper.GetInt("adam.port")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			adamForce = viper.GetBool("adam.force")
			adamRemoteRedisURL = viper.GetString("adam.redis.adam")
			adamRemoteRedis = viper.GetBool("adam.remote.redis")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		log.Infof("Executable path: %s", command)
		if !adamRemoteRedis {
			adamRemoteRedisURL = ""
		}
		if err := eden.StartAdam(adamPort, adamDist, adamForce, adamTag, adamRemoteRedisURL); err != nil {
			log.Errorf("cannot start adam: %s", err)
		} else {
			log.Infof("Adam is running and accessible on port %d", adamPort)
		}
	},
}

var stopAdamCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop adam",
	Long:  `Stop adam.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			adamRm = viper.GetBool("adam-rm")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := eden.StopAdam(adamRm); err != nil {
			log.Errorf("cannot stop adam: %s", err)
		}
	},
}

var statusAdamCmd = &cobra.Command{
	Use:   "status",
	Short: "status of adam",
	Long:  `Status of adam.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		statusAdam, err := eden.StatusAdam()
		if err != nil {
			log.Errorf("cannot obtain status of adam: %s", err)
		} else {
			fmt.Printf("Adam status: %s\n", statusAdam)
		}
	},
}

func adamInit() {
	adamCmd.AddCommand(startAdamCmd)
	adamCmd.AddCommand(stopAdamCmd)
	adamCmd.AddCommand(statusAdamCmd)
	startAdamCmd.Flags().StringVarP(&adamTag, "adam-tag", "", defaults.DefaultAdamTag, "tag on adam container to pull")
	startAdamCmd.Flags().StringVarP(&adamDist, "adam-dist", "", "", "adam dist to start (required)")
	startAdamCmd.Flags().IntVarP(&adamPort, "adam-port", "", defaults.DefaultAdamPort, "adam port to start")
	startAdamCmd.Flags().BoolVarP(&adamForce, "adam-force", "", false, "adam force rebuild")
	startAdamCmd.Flags().StringVarP(&adamRemoteRedisURL, "adam-redis-url", "", "", "adam remote redis url")
	startAdamCmd.Flags().BoolVarP(&adamRemoteRedis, "adam-redis", "", true, "use adam remote redis")
	stopAdamCmd.Flags().BoolVarP(&adamRm, "adam-rm", "", false, "adam rm on stop")
}
