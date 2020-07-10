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
	"runtime"
)

var (
	adamDist         string
	adamPort         int
	adamForce        bool
	eserverForce     bool
	eserverImageDist string
	eserverPort      int
	eserverTag       string
	evePidFile       string
	eveLogFile       string
	eveRemote        bool
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start harness",
	Long:  `Start harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			adamTag = viper.GetString("adam.tag")
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			adamPort = viper.GetInt("adam.port")
			adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
			adamForce = viper.GetBool("adam.force")
			adamRemoteRedisURL = viper.GetString("adam.redis.adam")
			adamRemoteRedis = viper.GetBool("adam.remote.redis")
			redisTag = viper.GetString("redis.tag")
			redisPort = viper.GetInt("redis.port")
			redisDist = utils.ResolveAbsPath(viper.GetString("redis.dist"))
			redisForce = viper.GetBool("redis.force")
			eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))
			eserverPort = viper.GetInt("eden.eserver.port")
			eserverForce = viper.GetBool("eden.eserver.force")
			eserverTag = viper.GetString("eden.eserver.tag")
			adamForce = viper.GetBool("adam.force")
			qemuARCH = viper.GetString("eve.arch")
			qemuOS = viper.GetString("eve.os")
			qemuAccel = viper.GetBool("eve.accel")
			qemuSMBIOSSerial = viper.GetString("eve.serial")
			qemuConfigFile = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveLogFile = utils.ResolveAbsPath(viper.GetString("eve.log"))
			eveRemote = viper.GetBool("eve.remote")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		adamPath, err := filepath.Abs(adamDist)
		if err != nil {
			log.Fatalf("adam-dist problems: %s", err)
		}
		redisPath, err := filepath.Abs(redisDist)
		if err != nil {
			log.Fatalf("redis-dist problems: %s", err)
		}
		if err = os.MkdirAll(redisPath, 0755); err != nil {
			log.Fatalf("Cannot create directory for redis (%s): %s", redisPath, err)
		}
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		if err := utils.StartRedis(redisPort, redisPath, redisForce, redisTag); err != nil {
			log.Errorf("cannot start redis: %s", err)
		} else {
			log.Infof("Redis is running and accessible on port %d", redisPort)
		}
		if !adamRemoteRedis {
			adamRemoteRedisURL = ""
		}
		if err := utils.StartAdam(adamPort, adamPath, adamForce, adamTag, adamRemoteRedisURL); err != nil {
			log.Errorf("cannot start adam: %s", err)
		} else {
			log.Infof("Adam is running and accesible on port %d", adamPort)
		}
		if err := utils.StartEServer(eserverPort, eserverImageDist, eserverForce, eserverTag); err != nil {
			log.Errorf("cannot start eserver: %s", err)
		} else {
			log.Infof("Eserver is running and accesible on port %d", eserverPort)
		}
		if eveRemote {
			return
		}
		if err := utils.StartEVEQemu(command, qemuARCH, qemuOS, eveImageFile, qemuSMBIOSSerial, qemuAccel, qemuConfigFile, eveLogFile, evePidFile); err != nil {
			log.Errorf("cannot start eve: %s", err)
		} else {
			log.Infof("EVE is starting")
		}
	},
}

func startInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	startCmd.Flags().StringVarP(&adamTag, "adam-tag", "", defaults.DefaultAdamTag, "tag on adam container to pull")
	startCmd.Flags().StringVarP(&adamDist, "adam-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultAdamDist), "adam dist to start (required)")
	startCmd.Flags().IntVarP(&adamPort, "adam-port", "", defaults.DefaultAdamPort, "adam dist to start")
	startCmd.Flags().BoolVarP(&adamForce, "adam-force", "", false, "adam force rebuild")
	startCmd.Flags().StringVarP(&adamRemoteRedisURL, "adam-redis-url", "", "", "adam remote redis url")
	startCmd.Flags().BoolVarP(&adamRemoteRedis, "adam-redis", "", true, "use adam remote redis")
	startCmd.Flags().StringVarP(&redisTag, "redis-tag", "", defaults.DefaultRedisTag, "tag on redis container to pull")
	startCmd.Flags().StringVarP(&redisDist, "redis-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultRedisDist), "redis dist to start (required)")
	startCmd.Flags().IntVarP(&redisPort, "redis-port", "", defaults.DefaultRedisPort, "redis dist to start")
	startCmd.Flags().BoolVarP(&redisForce, "redis-force", "", false, "redis force rebuild")
	startCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultImageDist), "image dist for eserver")
	startCmd.Flags().IntVarP(&eserverPort, "eserver-port", "", defaults.DefaultEserverPort, "eserver port")
	startCmd.Flags().StringVarP(&eserverTag, "eserver-tag", "", defaults.DefaultEServerTag, "tag of eserver container to pull")
	startCmd.Flags().BoolVarP(&eserverForce, "eserver-force", "", false, "eserver force rebuild")
	startCmd.Flags().StringVarP(&qemuARCH, "eve-arch", "", runtime.GOARCH, "arch of system")
	startCmd.Flags().StringVarP(&qemuOS, "eve-os", "", runtime.GOOS, "os to run on")
	startCmd.Flags().BoolVarP(&qemuAccel, "eve-accel", "", true, "use acceleration")
	startCmd.Flags().StringVarP(&qemuSMBIOSSerial, "eve-serial", "", defaults.DefaultEVESerial, "SMBIOS serial")
	startCmd.Flags().StringVarP(&qemuConfigFile, "qemu-config", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultQemuFileToSave), "config file to use")
	startCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	startCmd.Flags().StringVarP(&eveLogFile, "eve-log", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.log"), "file for save EVE log")
}
