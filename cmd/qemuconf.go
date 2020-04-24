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

const (
	defaultQemuFileToSave = "qemu.conf"
	defaultQemuCpus       = 4
	defaultQemuMemory     = 4096
)

var (
	qemuFileToSave     string
	qemuCpus           int
	qemuMemory         int
	qemuFirmwarePath   string
	qemuConfigPath     string
	qemuImagePath      string
	qemuDTBPath        string
	defaultQemuHostFwd = map[string]string{"2222": "22"}
	qemuHostFwd        map[string]string
	qemuSocketPath     string
)

var qemuConfCmd = &cobra.Command{
	Use:   "qemuconf",
	Short: "generate qemu config file",
	Long:  `Generate qemu config file.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			qemuConfigFile = viper.GetString("eve-config")
			qemuHostFwd = viper.GetStringMapString("eve-hostfwd")
			qemuFirmwarePath = viper.GetString("eve-firmware")
			qemuConfigPath = viper.GetString("config-part")
			qemuDTBPath = viper.GetString("dtb-part")
			qemuImagePath = viper.GetString("image-part")
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
		qemuImagePathAbsolute := ""
		if qemuImagePath != "" {
			qemuImagePathAbsolute, err = filepath.Abs(qemuImagePath)
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
		qemuFirmwarePathAbsolute := ""
		if qemuFirmwarePath != "" {
			qemuFirmwarePathAbsolute, err = filepath.Abs(qemuFirmwarePath)
			if err != nil {
				log.Fatal(err)
			}
		}
		//generate netdevs with unused subnets
		nets, err := utils.GetSubnetsNotUsed(2)
		if err != nil {
			log.Fatal(err)
		}
		settings := utils.QemuSettings{
			ConfigDrive: qemuConfigPathAbsolute,
			DTBDrive:    qemuDTBPathAbsolute,
			SystemDrive: qemuImagePathAbsolute,
			Firmware:    qemuFirmwarePathAbsolute,
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
	qemuConfCmd.Flags().StringVarP(&qemuFileToSave, "eve-config", "", defaultQemuFileToSave, "file to save config")
	qemuConfCmd.Flags().IntVarP(&qemuCpus, "cpus", "", defaultQemuCpus, "cpus")
	qemuConfCmd.Flags().IntVarP(&qemuMemory, "memory", "", defaultQemuMemory, "memory (MB)")
	qemuConfCmd.Flags().StringVarP(&qemuFirmwarePath, "eve-firmware", "", "", "firmware path")
	qemuConfCmd.Flags().StringVarP(&qemuConfigPath, "config-part", "", "", "path for config drive")
	qemuConfCmd.Flags().StringVarP(&qemuDTBPath, "dtb-part", "", "", "path for device tree drive (for arm)")
	qemuConfCmd.Flags().StringToStringVarP(&qemuHostFwd, "eve-hostfwd", "", defaultQemuHostFwd, "port forward map")
	qemuConfCmd.Flags().StringVarP(&qemuSocketPath, "qmp", "", "", "use qmp socket with path")
	qemuConfCmd.Flags().StringVarP(&qemuImagePath, "image-part", "", "", "path for image drive (required)")
	if err := cobra.MarkFlagRequired(qemuConfCmd.Flags(), "image-part"); err != nil {
		log.Fatal(err)
	}
	if err := viper.BindPFlags(qemuConfCmd.Flags()); err != nil {
		log.Fatal(err)
	}
}
