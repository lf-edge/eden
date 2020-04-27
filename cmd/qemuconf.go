package cmd

import (
	"fmt"
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

var qemuConfCmd = &cobra.Command{
	Use:   "qemuconf",
	Short: "generate qemu config file",
	Long:  `Generate qemu config file.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			qemuConfigFile = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			qemuHostFwd = viper.GetStringMapString("eve.hostfwd")
			qemuFirmware = viper.GetStringSlice("eve.firmware")
			qemuConfigPath = utils.ResolveAbsPath(viper.GetString("eve.config-part"))
			qemuDTBPath = utils.ResolveAbsPath(viper.GetString("eve.dtb-part"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
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
	},
}

func qemuConfInit() {
	qemuConfCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", defaultQemuFileToSave, "file to save config")
	qemuConfCmd.Flags().IntVarP(&qemuCpus, "cpus", "", defaultQemuCpus, "cpus")
	qemuConfCmd.Flags().IntVarP(&qemuMemory, "memory", "", defaultQemuMemory, "memory (MB)")
	qemuConfCmd.Flags().StringSliceVarP(&qemuFirmware, "eve-firmware", "", nil, "firmware path")
	qemuConfCmd.Flags().StringVarP(&qemuConfigPath, "config-part", "", "", "path for config drive")
	qemuConfCmd.Flags().StringVarP(&qemuDTBPath, "dtb-part", "", "", "path for device tree drive (for arm)")
	qemuConfCmd.Flags().StringToStringVarP(&qemuHostFwd, "eve-hostfwd", "", defaultQemuHostFwd, "port forward map")
	qemuConfCmd.Flags().StringVarP(&qemuSocketPath, "qmp", "", "", "use qmp socket with path")
}
