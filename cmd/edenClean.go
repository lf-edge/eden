package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configDir   string
	configSaved string

	currentContext bool
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
			adamTag = viper.GetString("adam.tag")
			adamPort = viper.GetInt("adam.port")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			adamRemoteRedisURL = viper.GetString("adam.redis.adam")
			adamRemoteRedis = viper.GetBool("adam.remote.redis")
			redisTag = viper.GetString("redis.tag")
			redisPort = viper.GetInt("redis.port")
			redisDist = utils.ResolveAbsPath(viper.GetString("redis.dist"))
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveDist = utils.ResolveAbsPath(viper.GetString("eve.dist"))
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			certsDir = utils.ResolveAbsPath(viper.GetString("eden.certs-dist"))
			eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))
			qemuFileToSave = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			redisDist = utils.ResolveAbsPath(viper.GetString("redis.dist"))
			registryDist = utils.ResolveAbsPath(viper.GetString("registry.dist"))
			configSaved = utils.ResolveAbsPath(fmt.Sprintf("%s-%s", configName, defaults.DefaultConfigSaved))
			eveRemote = viper.GetBool("eve.remote")
			devModel = viper.GetString("eve.devmodel")
			apiV1 = viper.GetBool("adam.v1")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if currentContext {
			log.Info("Cleanup current context")
			// we need to delete information about EVE from adam
			if err := eden.StartRedis(redisPort, redisDist, redisForce, redisTag); err != nil {
				log.Errorf("cannot start redis: %s", err)
			} else {
				log.Infof("Redis is running and accessible on port %d", redisPort)
			}
			if err := eden.StartAdam(adamPort, adamDist, adamForce, adamTag, adamRemoteRedisURL, apiV1); err != nil {
				log.Errorf("cannot start adam: %s", err)
			} else {
				log.Infof("Adam is running and accessible on port %d", adamPort)
			}
			eveUUID := viper.GetString("eve.uuid")
			if err := eden.CleanContext(eveDist, certsDir, filepath.Dir(eveImageFile), evePidFile, eveUUID, vmName, configSaved, eveRemote); err != nil {
				log.Fatalf("cannot CleanContext: %s", err)
			}
		} else {
			if err := eden.CleanEden(eveDist, adamDist, certsDir, filepath.Dir(eveImageFile),
				eserverImageDist, redisDist, registryDist, configDir, evePidFile,
				sdnPidFile, configSaved, eveRemote, devModel, vmName); err != nil {
				log.Fatalf("cannot CleanEden: %s", err)
			}
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
	cleanCmd.Flags().BoolVar(&currentContext, "current-context", true, "clean only current context")
	cleanCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")
	addSdnPidOpt(cleanCmd)
}
