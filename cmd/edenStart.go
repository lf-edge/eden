package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	adamDist           string
	adamPort           int
	adamForce          bool
	eserverForce       bool
	eserverImageDist   string
	eserverPort        int
	eserverTag         string
	evePidFile         string
	eveLogFile         string
	eveRemote         bool
	eveUsbNetConfFile string
	vmName            string
	cpus              int
	mem               int
	sdnDisable        bool
	sdnSourceDir      string
	sdnConfigDir      string
	sdnImageFile      string
	sdnLinuxkitBin    string
	sdnCPU            int
	sdnRAM            int
	sdnPidFile        string
	sdnConsoleLogFile string
	sdnTelnetPort     int
	sdnSSHPort        int
	sdnMgmtPort       int
	sdnNetModelFile   string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start harness",
	Long:  `Start harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
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
			registryTag = viper.GetString("registry.tag")
			registryPort = viper.GetInt("registry.port")
			redisTag = viper.GetString("redis.tag")
			redisPort = viper.GetInt("redis.port")
			redisDist = utils.ResolveAbsPath(viper.GetString("redis.dist"))
			redisForce = viper.GetBool("redis.force")
			eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))
			eserverPort = viper.GetInt("eden.eserver.port")
			eserverForce = viper.GetBool("eden.eserver.force")
			eserverTag = viper.GetString("eden.eserver.tag")
			adamForce = viper.GetBool("adam.force")
			devModel = viper.GetString("eve.devmodel")
			qemuARCH = viper.GetString("eve.arch")
			qemuOS = viper.GetString("eve.os")
			qemuAccel = viper.GetBool("eve.accel")
			qemuSMBIOSSerial = viper.GetString("eve.serial")
			qemuConfigFile = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			qemuMonitorPort = viper.GetInt("eve.qemu.monitor-port")
			qemuNetdevSocketPort = viper.GetInt("eve.qemu.netdev-socket-port")
			qemuFirmware = viper.GetStringSlice("eve.firmware")
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveLogFile = utils.ResolveAbsPath(viper.GetString("eve.log"))
			eveRemote = viper.GetBool("eve.remote")
			eveUsbNetConfFile = viper.GetString("eve.usbnetconf-file")
			hostFwd = viper.GetStringMapString("eve.hostfwd")
			eveTelnetPort = viper.GetInt("eve.telnet-port")
			apiV1 = viper.GetBool("adam.v1")
			cpus = viper.GetInt("eve.cpu")
			mem = viper.GetInt("eve.ram")
			gcpvTPM = viper.GetBool("eve.tpm")
			customInstallerPath = utils.ResolveAbsPath(viper.GetString("eve.custom-installer.path"))
			customInstallerFormat = viper.GetString("eve.custom-installer.format")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := eden.StartRedis(redisPort, redisDist, redisForce, redisTag); err != nil {
			log.Errorf("cannot start redis: %s", err)
		} else {
			log.Infof("Redis is running and accessible on port %d", redisPort)
		}
		if !adamRemoteRedis {
			adamRemoteRedisURL = ""
		}
		if err := eden.StartAdam(adamPort, adamDist, adamForce, adamTag, adamRemoteRedisURL, apiV1); err != nil {
			log.Errorf("cannot start adam: %s", err)
		} else {
			log.Infof("Adam is running and accesible on port %d", adamPort)
		}
		if err := eden.StartRegistry(registryPort, registryTag, registryDist); err != nil {
			log.Errorf("cannot start registry: %s", err)
		} else {
			log.Infof("registry is running and accesible on port %d", registryPort)
		}
		if err := eden.StartEServer(eserverPort, eserverImageDist, eserverForce, eserverTag); err != nil {
			log.Errorf("cannot start eserver: %s", err)
		} else {
			log.Infof("Eserver is running and accesible on port %d", eserverPort)
		}
		if eveRemote {
			return
		}
		if devModel == defaults.DefaultParallelsModel {
			if err := eden.StartEVEParallels(vmName, eveImageFile, cpus, mem, hostFwd); err != nil {
				log.Errorf("cannot start eve: %s", err)
			} else {
				log.Infof("EVE is starting in Parallels")
			}
		} else if devModel == defaults.DefaultVBoxModel {
			if err := eden.StartEVEVBox(vmName, eveImageFile, cpus, mem, hostFwd); err != nil {
				log.Errorf("cannot start eve: %s", err)
			} else {
				log.Infof("EVE is starting in Virtual Box")
			}
		} else {
			startEveQemu()
		}
	},
}

func startInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	startCmd.Flags().StringVarP(&adamTag, "adam-tag", "", defaults.DefaultAdamTag, "tag on adam container to pull")
	startCmd.Flags().StringVarP(&adamDist, "adam-dist", "", "", "adam dist to start (required)")
	startCmd.Flags().IntVarP(&adamPort, "adam-port", "", defaults.DefaultAdamPort, "adam dist to start")
	startCmd.Flags().BoolVarP(&adamForce, "adam-force", "", false, "adam force rebuild")
	startCmd.Flags().StringVarP(&adamRemoteRedisURL, "adam-redis-url", "", "", "adam remote redis url")
	startCmd.Flags().BoolVarP(&adamRemoteRedis, "adam-redis", "", true, "use adam remote redis")
	startCmd.Flags().StringVarP(&registryTag, "registry-tag", "", defaults.DefaultRegistryTag, "tag on registry container to pull")
	startCmd.Flags().IntVarP(&registryPort, "registry-port", "", defaults.DefaultRegistryPort, "registry port to start")
	startCmd.Flags().StringVarP(&registryDist, "registry-dist", "", "", "registry dist path to store (required)")
	startCmd.Flags().StringVarP(&redisTag, "redis-tag", "", defaults.DefaultRedisTag, "tag on redis container to pull")
	startCmd.Flags().StringVarP(&redisDist, "redis-dist", "", "", "redis dist to start (required)")
	startCmd.Flags().IntVarP(&redisPort, "redis-port", "", defaults.DefaultRedisPort, "redis dist to start")
	startCmd.Flags().BoolVarP(&redisForce, "redis-force", "", false, "redis force rebuild")
	startCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", "", "image dist for eserver")
	startCmd.Flags().IntVarP(&eserverPort, "eserver-port", "", defaults.DefaultEserverPort, "eserver port")
	startCmd.Flags().StringVarP(&eserverTag, "eserver-tag", "", defaults.DefaultEServerTag, "tag of eserver container to pull")
	startCmd.Flags().BoolVarP(&eserverForce, "eserver-force", "", false, "eserver force rebuild")
	startCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")
	startCmd.Flags().IntVarP(&cpus, "cpus", "", defaults.DefaultCpus, "cpus count")
	startCmd.Flags().IntVarP(&mem, "memory", "", defaults.DefaultMemory, "memory size (MB)")
	startCmd.Flags().StringVarP(&qemuARCH, "eve-arch", "", runtime.GOARCH, "arch of system")
	startCmd.Flags().StringVarP(&qemuOS, "eve-os", "", runtime.GOOS, "os to run on")
	startCmd.Flags().BoolVarP(&qemuAccel, "eve-accel", "", true, "use acceleration")
	startCmd.Flags().StringVarP(&qemuSMBIOSSerial, "eve-serial", "", defaults.DefaultEVESerial, "SMBIOS serial")
	startCmd.Flags().StringVarP(&qemuConfigFile, "qemu-config", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultQemuFileToSave), "config file to use")
	startCmd.Flags().IntVarP(&qemuMonitorPort, "qemu-monitor-port", "", defaults.DefaultQemuMonitorPort, "Port for access to QEMU monitor")
	startCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	startCmd.Flags().StringVarP(&eveLogFile, "eve-log", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.log"), "file for save EVE log")
	startCmd.Flags().StringVarP(&eveImageFile, "image-file", "", "", "path to image drive, overrides default setting")
	startCmd.Flags().StringVarP(&tapInterface, "with-tap", "", "", "use tap interface in QEMU as the third")
	startCmd.Flags().StringVarP(&eveUsbNetConfFile, "eve-usbnetconf-file", "", "", "path to device network config (aka usb.json) applied in runtime using a USB stick")
	addSdnStartOpts(startCmd)
}
