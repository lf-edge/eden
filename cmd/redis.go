package cmd

import (
	"fmt"
	"os"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newRedisCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var redisCmd = &cobra.Command{
		Use:               "redis",
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newStartRedisCmd(cfg),
				newStopRedisCmd(cfg),
				newStatusRedisCmd(),
			},
		},
	}

	groups.AddTo(redisCmd)

	return redisCmd
}

func newStartRedisCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var startRedisCmd = &cobra.Command{
		Use:   "start",
		Short: "start redis",
		Long:  `Start redis.`,
		Run: func(cmd *cobra.Command, args []string) {
			command, err := os.Executable()
			if err != nil {
				log.Fatalf("cannot obtain executable path: %s", err)
			}
			log.Infof("Executable path: %s", command)
			if err := eden.StartRedis(cfg.Redis.Port, cfg.Redis.Dist, cfg.Redis.Force, cfg.Redis.Tag); err != nil {
				log.Errorf("cannot start redis: %s", err)
			} else {
				log.Infof("Redis is running and accessible on port %d", cfg.Redis.Port)
			}
		},
	}

	startRedisCmd.Flags().StringVarP(&cfg.Redis.Tag, "redis-tag", "", defaults.DefaultRedisTag, "tag of redis container to pull")
	startRedisCmd.Flags().StringVarP(&cfg.Redis.Dist, "redis-dist", "", cfg.Redis.Dist, "redis dist to start (required)")
	startRedisCmd.Flags().IntVarP(&cfg.Redis.Port, "redis-port", "", defaults.DefaultRedisPort, "redis port to start")
	startRedisCmd.Flags().BoolVarP(&cfg.Redis.Force, "redis-force", "", cfg.Redis.Force, "redis force rebuild")

	return startRedisCmd
}

func newStopRedisCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var stopRedisCmd = &cobra.Command{
		Use:   "stop",
		Short: "stop redis",
		Long:  `Stop redis.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := eden.StopRedis(cfg.Runtime.RedisRm); err != nil {
				log.Errorf("cannot stop redis: %s", err)
			}
		},
	}
	stopRedisCmd.Flags().BoolVarP(&cfg.Runtime.RedisRm, "redis-rm", "", false, "redis rm on stop")

	return stopRedisCmd
}

func newStatusRedisCmd() *cobra.Command {
	var statusRedisCmd = &cobra.Command{
		Use:   "status",
		Short: "status of redis",
		Long:  `Status of redis.`,
		Run: func(cmd *cobra.Command, args []string) {
			statusRedis, err := eden.StatusRedis()
			if err != nil {
				log.Errorf("cannot obtain status of redis: %s", err)
			} else {
				fmt.Printf("Redis status: %s\n", statusRedis)
			}
		},
	}

	return statusRedisCmd
}
