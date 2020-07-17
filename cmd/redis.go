package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path"
	"path/filepath"
)

var (
	redisTag   string
	redisPort  int
	redisDist  string
	redisForce bool
	redisRm    bool
)

var redisCmd = &cobra.Command{
	Use: "redis",
}

var startRedisCmd = &cobra.Command{
	Use:   "start",
	Short: "start redis",
	Long:  `Start redis.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			redisTag = viper.GetString("redis.tag")
			redisPort = viper.GetInt("redis.port")
			redisDist = utils.ResolveAbsPath(viper.GetString("redis.dist"))
			redisForce = viper.GetBool("redis.force")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		redisPath, err := filepath.Abs(redisDist)
		if err != nil {
			log.Fatalf("redis-dist problems: %s", err)
		}
		if err = os.MkdirAll(redisPath, 0755); err != nil {
			log.Fatalf("Cannot create directory for redis (%s): %s", redisPath, err)
		}
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		log.Infof("Executable path: %s", command)
		if err := utils.StartRedis(redisPort, redisPath, redisForce, redisTag); err != nil {
			log.Errorf("cannot start redis: %s", err)
		} else {
			log.Infof("Redis is running and accessible on port %d", redisPort)
		}
	},
}

var stopRedisCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop redis",
	Long:  `Stop redis.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			redisRm = viper.GetBool("redis-rm")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := utils.StopRedis(redisRm); err != nil {
			log.Errorf("cannot stop redis: %s", err)
		}
	},
}

var statusRedisCmd = &cobra.Command{
	Use:   "status",
	Short: "status of redis",
	Long:  `Status of redis.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		statusRedis, err := utils.StatusRedis()
		if err != nil {
			log.Errorf("cannot obtain status of redis: %s", err)
		} else {
			fmt.Printf("Redis status: %s\n", statusRedis)
		}
	},
}

func redisInit() {
	redisCmd.AddCommand(startRedisCmd)
	redisCmd.AddCommand(stopRedisCmd)
	redisCmd.AddCommand(statusRedisCmd)
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	startRedisCmd.Flags().StringVarP(&redisTag, "redis-tag", "", defaults.DefaultRedisTag, "tag of redis container to pull")
	startRedisCmd.Flags().StringVarP(&redisDist, "redis-dist", "", path.Join(currentPath, defaults.DefaultDist, defaults.DefaultRedisDist), "redis dist to start (required)")
	startRedisCmd.Flags().IntVarP(&redisPort, "redis-port", "", defaults.DefaultRedisPort, "redis port to start")
	startRedisCmd.Flags().BoolVarP(&redisForce, "redis-force", "", false, "redis force rebuild")
	stopRedisCmd.Flags().BoolVarP(&redisRm, "redis-rm", "", false, "redis rm on stop")
}
