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

var configDir string

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "clean harness",
	Long:  `Clean harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveDist = utils.ResolveAbsPath(viper.GetString("eve.dist"))
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			certsDir = utils.ResolveAbsPath(viper.GetString("eden.certs-dist"))
			eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))
			qemuFileToSave = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			redisDist = utils.ResolveAbsPath(viper.GetString("redis.dist"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		if err := utils.CleanEden(command, eveDist, adamDist, certsDir, eserverImageDist, redisDist,
			configDir, evePidFile); err != nil {
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
	cleanCmd.Flags().StringVarP(&redisDist, "redis-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultRedisDist), "redis dist")
	cleanCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultQemuFileToSave), "file to save qemu config")
	cleanCmd.Flags().StringVarP(&adamDist, "adam-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultAdamDist), "adam dist to start (required)")
	cleanCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultImageDist), "image dist for eserver")

	cleanCmd.Flags().StringVarP(&certsDir, "certs-dist", "o", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultCertsDist), "directory with certs")
	cleanCmd.Flags().StringVarP(&configDir, "config-dist", "", configDist, "directory for config")
}
