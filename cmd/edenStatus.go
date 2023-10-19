package cmd

import (
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newStatusCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var allConfigs bool
	var vmName string

	var statusCmd = &cobra.Command{
		Use:               "status",
		Short:             "status of harness",
		Long:              `Status of harness.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.Status(vmName, allConfigs); err != nil {
				log.Fatal(err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	statusCmd.Flags().StringVarP(&cfg.Eve.Pid, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file with EVE pid")
	statusCmd.Flags().BoolVar(&allConfigs, "all", true, "show status for all configs")
	statusCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")

	addSdnPidOpt(statusCmd, cfg)
	addSdnPortOpts(statusCmd, cfg)

	return statusCmd

}
