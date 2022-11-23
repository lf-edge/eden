package cmd

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newStartCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var startCmd = &cobra.Command{
		Use:               "start",
		Short:             "start harness",
		Long:              `Start harness.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.StartEden(cfg, cfg.Registry.Dist, cfg.Runtime.VmName, cfg.Eve.TapInterface); err != nil {
				log.Fatalf("Start eden failed: %s", err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	startCmd.Flags().StringVarP(&cfg.Adam.Tag, "adam-tag", "", defaults.DefaultAdamTag, "tag on adam container to pull")
	startCmd.Flags().StringVarP(&cfg.Adam.Dist, "adam-dist", "", cfg.Adam.Dist, "adam dist to start (required)")
	startCmd.Flags().IntVarP(&cfg.Adam.Port, "adam-port", "", defaults.DefaultAdamPort, "adam dist to start")
	startCmd.Flags().BoolVarP(&cfg.Adam.Force, "adam-force", "", cfg.Adam.Force, "adam force rebuild")
	startCmd.Flags().StringVarP(&cfg.Adam.Redis.RemoteURL, "adam-redis-url", "", cfg.Adam.Redis.RemoteURL, "adam remote redis url")
	startCmd.Flags().BoolVarP(&cfg.Adam.Remote.Redis, "adam-redis", "", cfg.Adam.Remote.Redis, "use adam remote redis")

	startCmd.Flags().StringVarP(&cfg.Registry.Tag, "registry-tag", "", defaults.DefaultRegistryTag, "tag on registry container to pull")
	startCmd.Flags().IntVarP(&cfg.Registry.Port, "registry-port", "", defaults.DefaultRegistryPort, "registry port to start")
	startCmd.Flags().StringVarP(&cfg.Registry.Dist, "registry-dist", "", cfg.Registry.Dist, "registry dist path to store (required)")

	startCmd.Flags().StringVarP(&cfg.Adam.Redis.Tag, "redis-tag", "", defaults.DefaultRedisTag, "tag on redis container to pull")
	startCmd.Flags().StringVarP(&cfg.Adam.Redis.Dist, "redis-dist", "", cfg.Adam.Redis.Dist, "redis dist to start (required)")
	startCmd.Flags().IntVarP(&cfg.Adam.Redis.Port, "redis-port", "", defaults.DefaultRedisPort, "redis dist to start")
	startCmd.Flags().BoolVarP(&cfg.Adam.Redis.Force, "redis-force", "", cfg.Adam.Redis.Force, "redis force rebuild")

	startCmd.Flags().StringVarP(&cfg.Eden.Eserver.Images.EserverImageDist, "image-dist", "", cfg.Eden.Eserver.Images.EserverImageDist, "image dist for eserver")
	startCmd.Flags().IntVarP(&cfg.Eden.Eserver.Port, "eserver-port", "", defaults.DefaultEserverPort, "eserver port")
	startCmd.Flags().StringVarP(&cfg.Eden.Eserver.Tag, "eserver-tag", "", defaults.DefaultEServerTag, "tag of eserver container to pull")
	startCmd.Flags().BoolVarP(&cfg.Eden.Eserver.Force, "eserver-force", "", cfg.Eden.Eserver.Force, "eserver force rebuild")

	startCmd.Flags().IntVarP(&cfg.Eve.QemuCpus, "cpus", "", defaults.DefaultCpus, "cpus count")
	startCmd.Flags().IntVarP(&cfg.Eve.QemuMemory, "memory", "", defaults.DefaultMemory, "memory size (MB)")
	startCmd.Flags().StringVarP(&cfg.Eve.Arch, "eve-arch", "", runtime.GOARCH, "arch of system")
	startCmd.Flags().StringVarP(&cfg.Eve.QemuOS, "eve-os", "", runtime.GOOS, "os to run on")
	startCmd.Flags().BoolVarP(&cfg.Eve.Accel, "eve-accel", "", cfg.Eve.Accel, "use acceleration")
	startCmd.Flags().StringVarP(&cfg.Eve.Serial, "eve-serial", "", defaults.DefaultEVESerial, "SMBIOS serial")
	startCmd.Flags().StringVarP(&cfg.Eve.QemuConfigPath, "qemu-config", "", filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultQemuFileToSave), "config file to use")
	startCmd.Flags().IntVarP(&cfg.Eve.QemuConfig.MonitorPort, "qemu-monitor-port", "", defaults.DefaultQemuMonitorPort, "Port for access to QEMU monitor")
	startCmd.Flags().StringVarP(&cfg.Eve.Pid, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	startCmd.Flags().StringVarP(&cfg.Eve.Log, "eve-log", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.log"), "file for save EVE log")
	startCmd.Flags().StringVarP(&cfg.Eve.ImageFile, "image-file", "", cfg.Eve.ImageFile, "path to image drive, overrides default setting")
	startCmd.Flags().StringVarP(&cfg.Eve.TapInterface, "with-tap", "", "", "use tap interface in QEMU as the third")
	startCmd.Flags().IntVarP(&cfg.Eve.EthLoops, "with-eth-loops", "", 0, "add one or more ethernet loops (requires custom device model)")
	startCmd.Flags().StringVarP(&cfg.Runtime.VmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")

	startCmd.Flags().StringVarP(&cfg.Eve.UsbNetConfFile, "eve-usbnetconf-file", "", "", "path to device network config (aka usb.json) applied in runtime using a USB stick")

	addSdnStartOpts(startCmd, cfg)

	return startCmd
}

func addSdnStartOpts(parentCmd *cobra.Command, cfg *openevec.EdenSetupArgs) {
	addSdnPidOpt(parentCmd, cfg)
	addSdnConfigDirOpt(parentCmd, cfg)
	addSdnVmOpts(parentCmd, cfg)
	addSdnNetModelOpt(parentCmd, cfg)
	addSdnPortOpts(parentCmd, cfg)
	addSdnLogOpt(parentCmd, cfg)
	addSdnImageOpt(parentCmd, cfg)
	addSdnDisableOpt(parentCmd, cfg)
}
