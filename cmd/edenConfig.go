package cmd

import (
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

var (
	qemuFileToSave string
	qemuCpus       int
	qemuMemory     int
	qemuFirmware   []string
	qemuConfigPath string
	eveImageFile   string
	qemuDTBPath    string
	qemuHostFwd    map[string]string
	qemuSocketPath string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "generate config eden",
	Long:  `Generate config eden.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if config == "" {
			config, err = utils.DefaultConfigPath()
			if err != nil {
				log.Fatalf("fail in DefaultConfigPath: %s", err)
			}
		}
		if _, err := os.Stat(config); !os.IsNotExist(err) {
			if force {
				if err := os.Remove(config); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Infof("config already exists: %s", config)
			}
		}
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			log.Fatalf("error reading config: %s", err)
		}
		if viperLoaded {
			qemuFirmware = viper.GetStringSlice("eve.firmware")
			qemuConfigPath = utils.ResolveAbsPath(viper.GetString("eve.config-part"))
			qemuDTBPath = utils.ResolveAbsPath(viper.GetString("eve.dtb-part"))
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			qemuHostFwd = viper.GetStringMapString("eve.hostfwd")
			qemuFileToSave = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(qemuFileToSave); os.IsNotExist(err) {
			f, err := os.Create(qemuFileToSave)
			if err != nil {
				log.Fatal(err)
			}
			qemuConfigPathAbsolute := ""
			if qemuConfigPath != "" {
				qemuConfigPathAbsolute, err = filepath.Abs(qemuConfigPath)
				if err != nil {
					log.Fatal(err)
				}
			}
			qemuDTBPathAbsolute := ""
			if qemuDTBPath != "" {
				qemuDTBPathAbsolute, err = filepath.Abs(qemuDTBPath)
				if err != nil {
					log.Fatal(err)
				}
			}
			var qemuFirmwareParam []string
			for _, el := range qemuFirmware {
				qemuFirmwarePathAbsolute := utils.ResolveAbsPath(el)
				if err != nil {
					log.Fatal(err)
				}
				qemuFirmwareParam = append(qemuFirmwareParam, qemuFirmwarePathAbsolute)
			}
			//generate netdevs with unused subnets
			nets, err := utils.GetSubnetsNotUsed(2)
			if err != nil {
				log.Fatal(err)
			}
			settings := utils.QemuSettings{
				ConfigDrive: qemuConfigPathAbsolute,
				DTBDrive:    qemuDTBPathAbsolute,
				Firmware:    qemuFirmwareParam,
				MemoryMB:    qemuMemory,
				CPUs:        qemuCpus,
				HostFWD:     qemuHostFwd,
				NetDevs:     nets,
			}
			conf, err := settings.GenerateQemuConfig()
			if err != nil {
				log.Fatal(err)
			}
			_, err = f.Write(conf)
			if err != nil {
				log.Fatal(err)
			}
			if err := f.Close(); err != nil {
				log.Fatal(err)
			}
			log.Infof("QEMU config file generated: %s", qemuFileToSave)
		} else {
			log.Infof("QEMU config already exists: %s", qemuFileToSave)
		}
	},
}

func configInit() {
	configPath, err := utils.DefaultConfigPath()
	if err != nil {
		log.Fatal(err)
	}
	configCmd.Flags().StringVar(&config, "config", configPath, "path to config file")
	configCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", defaultQemuFileToSave, "file to save config")
	configCmd.Flags().IntVarP(&qemuCpus, "cpus", "", defaultQemuCpus, "cpus")
	configCmd.Flags().IntVarP(&qemuMemory, "memory", "", defaultQemuMemory, "memory (MB)")
	configCmd.Flags().StringSliceVarP(&qemuFirmware, "eve-firmware", "", nil, "firmware path")
	configCmd.Flags().StringVarP(&qemuConfigPath, "config-part", "", "", "path for config drive")
	configCmd.Flags().StringVarP(&qemuDTBPath, "dtb-part", "", "", "path for device tree drive (for arm)")
	configCmd.Flags().StringToStringVarP(&qemuHostFwd, "eve-hostfwd", "", defaultQemuHostFwd, "port forward map")
	configCmd.Flags().StringVarP(&qemuSocketPath, "qmp", "", "", "use qmp socket with path")
}
