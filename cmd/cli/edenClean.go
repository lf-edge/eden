package cmd

import (
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newCleanCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var configDist, vmName string
	var currentContext bool

	var cleanCmd = &cobra.Command{
		Use:               "clean",
		Short:             "clean harness",
		Long:              `Clean harness.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.EdenClean(*cfg, *configName, configDist, vmName, currentContext); err != nil {
				log.Fatalf("Setup eden failed: %s", err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	configDist, err = utils.DefaultEdenDir()
	if err != nil {
		log.Fatal(err)
	}

	cleanCmd.Flags().StringVarP(&cfg.Eve.Pid, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file with EVE pid")
	cleanCmd.Flags().StringVarP(&cfg.Eve.Dist, "eve-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultEVEDist), "directory to save EVE")
	cleanCmd.Flags().StringVarP(&cfg.Adam.Redis.Dist, "redis-dist", "", cfg.Adam.Redis.Dist, "redis dist")
	cleanCmd.Flags().StringVarP(&cfg.Eve.QemuFileToSave, "qemu-config", "", "", "file to save qemu config")
	cleanCmd.Flags().StringVarP(&cfg.Adam.Dist, "adam-dist", "", cfg.Adam.Dist, "adam dist to start (required)")
	cleanCmd.Flags().StringVarP(&cfg.Eden.Images.EServerImageDist, "image-dist", "", "", "image dist for eserver")

	cleanCmd.Flags().StringVarP(&cfg.Eden.CertsDir, "certs-dist", "o", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultCertsDist), "directory with certs")
	cleanCmd.Flags().StringVarP(&configDist, "config-dist", "", configDist, "directory with eden config to cleanup")
	cleanCmd.Flags().BoolVar(&currentContext, "current-context", true, "clean only current context")
	cleanCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")

	addSdnPidOpt(cleanCmd, cfg)

	return cleanCmd
}
