package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	edenRoot    string
	rewriteRoot bool
)

var exportCmd = &cobra.Command{
	Use:   "export <filename>",
	Short: "export harness",
	Long:  `Export certificates and configs of harness into tar.gz file.`,
	Args:  cobra.ExactArgs(1),
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
			certsDir = utils.ResolveAbsPath(viper.GetString("eden.certs-dist"))
			apiV1 = viper.GetBool("adam.v1")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		changer := &adamChanger{}
		// we need to obtain information about EVE from Adam
		if err := eden.StartRedis(redisPort, redisDist, false, redisTag); err != nil {
			log.Errorf("cannot start redis: %s", err)
		} else {
			log.Infof("Redis is running and accessible on port %d", redisPort)
		}
		if err := eden.StartAdam(adamPort, adamDist, false, adamTag, adamRemoteRedisURL, apiV1); err != nil {
			log.Errorf("cannot start adam: %s", err)
		} else {
			log.Infof("Adam is running and accessible on port %d", adamPort)
		}
		ctrl, err := changer.getController()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		dev, err := ctrl.GetDeviceCurrent()
		if err == nil {
			deviceCert, err := ctrl.GetDeviceCert(dev)
			if err != nil {
				log.Warn(err)
			} else {
				if err = ioutil.WriteFile(ctrl.GetVars().EveDeviceCert, deviceCert.Cert, 0777); err != nil {
					log.Warn(err)
				}
			}
		} else {
			log.Info("Device not registered, will not save device cert")
		}
		edenDir, err := utils.DefaultEdenDir()
		if err != nil {
			log.Fatal(err)
		}
		tarFile := args[0]
		files := []utils.FileToSave{
			{Location: certsDir, Destination: filepath.Join("dist", filepath.Base(certsDir))},
			{Location: utils.ResolveAbsPath(defaults.DefaultCertsDist), Destination: filepath.Join("dist", defaults.DefaultCertsDist)},
			{Location: edenDir, Destination: "eden"},
		}
		if err := utils.CreateTarGz(tarFile, files); err != nil {
			log.Fatal(err)
		}
		log.Infof("Export Eden done")
	},
}

var importCmd = &cobra.Command{
	Use:   "import <filename>",
	Short: "import harness",
	Long:  `Import certificates and configs of harness from tar.gz file.`,
	Args:  cobra.ExactArgs(1),
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
			certsDir = utils.ResolveAbsPath(viper.GetString("eden.certs-dist"))
			edenRoot = viper.GetString("eden.root")
			apiV1 = viper.GetBool("adam.v1")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		edenDir, err := utils.DefaultEdenDir()
		if err != nil {
			log.Fatal(err)
		}
		files := []utils.FileToSave{
			{Location: filepath.Join("dist", filepath.Base(certsDir)), Destination: certsDir},
			{Location: filepath.Join("dist", defaults.DefaultCertsDist), Destination: utils.ResolveAbsPath(defaults.DefaultCertsDist)},
			{Location: "eden", Destination: edenDir},
		}
		if err := utils.UnpackTarGz(args[0], files); err != nil {
			log.Fatal(err)
		}
		if rewriteRoot {
			// we need to rewrite eden root to match with local
			viperLoaded, err := utils.LoadConfigFile(configFile)
			if err != nil {
				log.Fatalf("error reading config: %s", err.Error())
			}
			if viperLoaded {
				if edenRoot != viper.GetString("eden.root") {
					viper.Set("eve.root", edenRoot)
					if err = utils.GenerateConfigFileFromViper(); err != nil {
						log.Fatalf("error writing config: %s", err)
					}
				}
			}
		}
		// we need to put information about EVE into Adam
		if err := eden.StartRedis(redisPort, redisDist, false, redisTag); err != nil {
			log.Errorf("cannot start redis: %s", err)
		} else {
			log.Infof("Redis is running and accessible on port %d", redisPort)
		}
		if err := eden.StartAdam(adamPort, adamDist, false, adamTag, adamRemoteRedisURL, apiV1); err != nil {
			log.Errorf("cannot start adam: %s", err)
		} else {
			log.Infof("Adam is running and accessible on port %d", adamPort)
		}
		changer := &adamChanger{}
		ctrl, err := changer.getController()
		if err != nil {
			log.Fatal(err)
		}
		devUUID, err := ctrl.DeviceGetByOnboard(ctrl.GetVars().EveCert)
		if err != nil {
			log.Debug(err)
		}
		if devUUID == uuid.Nil {
			if _, err := os.Stat(ctrl.GetVars().EveDeviceCert); os.IsNotExist(err) {
				log.Warnf("No device cert %s, you device was not registered", ctrl.GetVars().EveDeviceCert)
			} else {
				if _, err := os.Stat(ctrl.GetVars().EveCert); os.IsNotExist(err) {
					log.Fatalf("No onboard cert in %s, you need to run 'eden setup' first", ctrl.GetVars().EveCert)
				}
				deviceCert, err := ioutil.ReadFile(ctrl.GetVars().EveDeviceCert)
				if err != nil {
					log.Fatal(err)
				}
				onboardCert, err := ioutil.ReadFile(ctrl.GetVars().EveCert)
				if err != nil {
					log.Warn(err)
				}
				dc := types.DeviceCert{
					Cert:   deviceCert,
					Serial: ctrl.GetVars().EveSerial,
				}
				if onboardCert != nil {
					dc.Onboard = onboardCert
				}
				err = ctrl.UploadDeviceCert(dc)
				if err != nil {
					log.Fatal(err)
				}
			}
			log.Info("You need to run 'eden eve onboard")
		} else {
			log.Info("Device already exists")
		}
	},
}

func exportImportInit() {
	utilsCmd.AddCommand(importCmd)
	importCmd.Flags().BoolVar(&rewriteRoot, "rewrite-root", true, "Rewrite eve.root with local value")
	utilsCmd.AddCommand(exportCmd)
}
