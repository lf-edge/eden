package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

var (
	configDir   string
	configSaved string
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "clean harness",
	Long:  `Clean harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveDist = utils.ResolveAbsPath(viper.GetString("eve.dist"))
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			certsDir = utils.ResolveAbsPath(viper.GetString("eden.certs-dist"))
			eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))
			qemuFileToSave = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			redisDist = utils.ResolveAbsPath(viper.GetString("redis.dist"))
			context, err := utils.ContextLoad()
			if err != nil {
				log.Fatalf("Load context error: %s", err)
			}
			configSaved = utils.ResolveAbsPath(fmt.Sprintf("%s-%s", context.Current, defaults.DefaultConfigSaved))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		if err := utils.CleanEden(command, eveDist, adamDist, certsDir, filepath.Dir(eveImageFile),
			eserverImageDist, redisDist, configDir, evePidFile,
			configSaved); err != nil {
			log.Fatalf("cannot CleanEden: %s", err)
		}
		log.Infof("CleanEden done")
	},
}

func cleanInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	configDist, err := utils.DefaultEdenDir()
	if err != nil {
		log.Fatal(err)
	}
	cleanCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file with EVE pid")
	cleanCmd.Flags().StringVarP(&eveDist, "eve-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultEVEDist), "directory to save EVE")
	cleanCmd.Flags().StringVarP(&redisDist, "redis-dist", "", "", "redis dist")
	cleanCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", "", "file to save qemu config")
	cleanCmd.Flags().StringVarP(&adamDist, "adam-dist", "", "", "adam dist to start (required)")
	cleanCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", "", "image dist for eserver")

	cleanCmd.Flags().StringVarP(&certsDir, "certs-dist", "o", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultCertsDist), "directory with certs")
	cleanCmd.Flags().StringVarP(&configDir, "config-dist", "", configDist, "directory for config")
}
