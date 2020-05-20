package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path"
	"path/filepath"
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
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			adamTag = viper.GetString("adam.tag")
			adamPort = viper.GetString("adam.port")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			adamForce = viper.GetBool("adam.force")
			adamRemoteRedisURL = viper.GetString("adam.redis.adam")
			adamRemoteRedis = viper.GetBool("adam.remote.redis")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		adamPath, err := filepath.Abs(adamDist)
		if err != nil {
			log.Fatalf("adam-dist problems: %s", err)
		}
		if _, err = os.Lstat(fmt.Sprintf("%s/run", adamPath)); os.IsNotExist(err) {
			log.Fatalf("%s not found. Please run ./eden setup before start to generate certs", fmt.Sprintf("%s/run", adamPath))
		}
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		log.Infof("Executable path: %s", command)
		if !adamRemoteRedis {
			adamRemoteRedisURL = ""
		}
		if err := utils.StartAdam(adamPort, adamPath, adamForce, adamTag, adamRemoteRedisURL); err != nil {
			log.Errorf("cannot start adam: %s", err)
		} else {
			log.Infof("Adam is running and accessible on port %s", adamPort)
		}
	},
}

var stopAdamCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop adam",
	Long:  `Stop adam.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
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
		if err := utils.StopAdam(adamRm); err != nil {
			log.Errorf("cannot stop adam: %s", err)
		}
	},
}

var statusAdamCmd = &cobra.Command{
	Use:   "status",
	Short: "status of adam",
	Long:  `Status of adam.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
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
	},
}

func adamInit() {
	adamCmd.AddCommand(startAdamCmd)
	adamCmd.AddCommand(stopAdamCmd)
	adamCmd.AddCommand(statusAdamCmd)
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	startAdamCmd.Flags().StringVarP(&adamTag, "adam-tag", "", defaultAdamTag, "tag on adam container to pull")
	startAdamCmd.Flags().StringVarP(&adamDist, "adam-dist", "", path.Join(currentPath, "dist", "adam"), "adam dist to start (required)")
	startAdamCmd.Flags().StringVarP(&adamPort, "adam-port", "", defaultAdamPort, "adam port to start")
	startAdamCmd.Flags().BoolVarP(&adamForce, "adam-force", "", false, "adam force rebuild")
	startAdamCmd.Flags().StringVarP(&adamRemoteRedisURL, "adam-redis-url", "", "", "adam remote redis url")
	startAdamCmd.Flags().BoolVarP(&adamRemoteRedis, "adam-redis", "", true, "use adam remote redis")
	stopAdamCmd.Flags().BoolVarP(&adamRm, "adam-rm", "", false, "adam rm on stop")
}
