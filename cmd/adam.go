package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newAdamCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var adamCmd = &cobra.Command{
		Use:               "adam",
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newStartAdamCmd(cfg),
				newStopAdamCmd(cfg),
				newStatusAdamCmd(),
			},
		},
	}

	groups.AddTo(adamCmd)

	return adamCmd
}

func newStartAdamCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var startAdamCmd = &cobra.Command{
		Use:   "start",
		Short: "start adam",
		Long:  `Start adam.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.AdamStart(cfg); err != nil {
				log.Fatalf("Adam start failed: %s", err)
			}
		},
	}

	startAdamCmd.Flags().StringVarP(&cfg.Adam.Tag, "adam-tag", "", defaults.DefaultAdamTag, "tag on adam container to pull")
	startAdamCmd.Flags().StringVarP(&cfg.Adam.Dist, "adam-dist", "", cfg.Adam.Dist, "adam dist to start (required)")
	startAdamCmd.Flags().IntVarP(&cfg.Adam.Port, "adam-port", "", defaults.DefaultAdamPort, "adam port to start")
	startAdamCmd.Flags().BoolVarP(&cfg.Adam.Force, "adam-force", "", cfg.Adam.Force, "adam force rebuild")
	startAdamCmd.Flags().StringVarP(&cfg.Adam.Redis.RemoteURL, "adam-redis-url", "", cfg.Adam.Redis.RemoteURL, "adam remote redis url")
	startAdamCmd.Flags().BoolVarP(&cfg.Adam.Remote.Redis, "adam-redis", "", cfg.Adam.Remote.Redis, "use adam remote redis")

	return startAdamCmd
}

func newStopAdamCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var stopAdamCmd = &cobra.Command{
		Use:   "stop",
		Short: "stop adam",
		Long:  `Stop adam.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := eden.StopAdam(cfg.Runtime.AdamRm); err != nil {
				log.Errorf("cannot stop adam: %s", err)
			}
		},
	}

	stopAdamCmd.Flags().BoolVarP(&cfg.Runtime.AdamRm, "adam-rm", "", false, "adam rm on stop")

	return stopAdamCmd
}

func newStatusAdamCmd() *cobra.Command {
	var statusAdamCmd = &cobra.Command{
		Use:   "status",
		Short: "status of adam",
		Long:  `Status of adam.`,
		Run: func(cmd *cobra.Command, args []string) {
			statusAdam, err := eden.StatusAdam()
			if err != nil {
				log.Errorf("cannot obtain status of adam: %s", err)
			} else {
				fmt.Printf("Adam status: %s\n", statusAdam)
			}
		},
	}

	return statusAdamCmd
}
