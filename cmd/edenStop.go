package cmd

import (
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newStopCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var stopCmd = &cobra.Command{
		Use:               "stop",
		Short:             "stop harness",
		Long:              `Stop harness.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
		Run: func(cmd *cobra.Command, args []string) {
			eden.StopEden(
				cfg.Runtime.AdamRm, cfg.Runtime.RedisRm,
				cfg.Runtime.RegistryRm, cfg.Runtime.EServerRm,
				cfg.Eve.Remote, cfg.Eve.Pid,
				swtpmPidFile(cfg), cfg.Sdn.PidFile,
				cfg.Eve.DevModel, cfg.Runtime.VmName,
			)
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	stopCmd.Flags().BoolVarP(&cfg.Runtime.AdamRm, "adam-rm", "", false, "adam rm on stop")
	stopCmd.Flags().BoolVarP(&cfg.Runtime.RegistryRm, "registry-rm", "", false, "registry rm on stop")
	stopCmd.Flags().BoolVarP(&cfg.Runtime.RedisRm, "redis-rm", "", false, "redis rm on stop")
	stopCmd.Flags().BoolVarP(&cfg.Runtime.EServerRm, "eserver-rm", "", false, "eserver rm on stop")
	stopCmd.Flags().StringVarP(&cfg.Eve.Pid, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file with EVE pid")
	stopCmd.Flags().StringVarP(&cfg.Runtime.VmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")

	addSdnPidOpt(stopCmd, cfg)

	return stopCmd
}
