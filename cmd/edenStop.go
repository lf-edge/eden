package cmd

import (
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newStopCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var vmName string
	var adamRm, registryRm, redisRm, eServerRm bool

	var stopCmd = &cobra.Command{
		Use:               "stop",
		Short:             "stop harness",
		Long:              `Stop harness.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.StopEden(adamRm, redisRm, registryRm, eServerRm, vmName); err != nil {
				log.Fatal(err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	stopCmd.Flags().BoolVarP(&adamRm, "adam-rm", "", false, "adam rm on stop")
	stopCmd.Flags().BoolVarP(&registryRm, "registry-rm", "", false, "registry rm on stop")
	stopCmd.Flags().BoolVarP(&redisRm, "redis-rm", "", false, "redis rm on stop")
	stopCmd.Flags().BoolVarP(&eServerRm, "eserver-rm", "", false, "eserver rm on stop")
	stopCmd.Flags().StringVarP(&cfg.Eve.Pid, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file with EVE pid")
	stopCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")

	addSdnPidOpt(stopCmd, cfg)

	return stopCmd
}
