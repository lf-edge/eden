package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/eden"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	adamRm    bool
	eserverRm bool
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop harness",
	Long:  `Stop harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveRemote = viper.GetBool("eve.remote")
			devModel = viper.GetString("eve.devmodel")
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			gcpvTPM = viper.GetBool("eve.tpm")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		eden.StopEden(adamRm, redisRm, registryRm, eserverRm, eveRemote, evePidFile,
			swtpmPidFile(), sdnPidFile, devModel, vmName)
	},
}

func stopInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	stopCmd.Flags().BoolVarP(&adamRm, "adam-rm", "", false, "adam rm on stop")
	stopCmd.Flags().BoolVarP(&registryRm, "registry-rm", "", false, "registry rm on stop")
	stopCmd.Flags().BoolVarP(&redisRm, "redis-rm", "", false, "redis rm on stop")
	stopCmd.Flags().BoolVarP(&eserverRm, "eserver-rm", "", false, "eserver rm on stop")
	stopCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file with EVE pid")
	stopCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")
	addSdnPidOpt(stopCmd)
}
