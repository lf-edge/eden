package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"runtime"
)

var eveCmd = &cobra.Command{
	Use: "eve",
}

var startEveCmd = &cobra.Command{
	Use:   "start",
	Short: "start eve",
	Long:  `Start eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			qemuARCH = viper.GetString("eve.arch")
			qemuOS = viper.GetString("eve.os")
			qemuAccel = viper.GetBool("eve.accel")
			qemuSMBIOSSerial = viper.GetString("eve.serial")
			qemuConfigFile = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveLogFile = utils.ResolveAbsPath(viper.GetString("eve.log"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		if err := utils.StartEVEQemu(command, qemuARCH, qemuOS, eveImageFile, qemuSMBIOSSerial, qemuAccel, qemuConfigFile, eveLogFile, evePidFile); err != nil {
			log.Errorf("cannot start eve: %s", err)
		} else {
			fmt.Println("EVE is running")
		}
	},
}

var stopEveCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop eve",
	Long:  `Stop eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := utils.StopEVEQemu(evePidFile); err != nil {
			log.Errorf("cannot stop EVE: %s", err)
		}
	},
}

var statusEveCmd = &cobra.Command{
	Use:   "status",
	Short: "status of eve",
	Long:  `Status of eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		statusEVE, err := utils.StatusEVEQemu(evePidFile)
		if err != nil {
			log.Errorf("cannot obtain status of EVE: %s", err)
		} else {
			fmt.Printf("EVE status: %s\n", statusEVE)
		}
	},
}

func eveInit() {
	eveCmd.AddCommand(qemuConfCmd)
	qemuConfInit()
	eveCmd.AddCommand(qemuRunCmd)
	qemuRunInit()
	eveCmd.AddCommand(confChangerCmd)
	confChangerInit()
	eveCmd.AddCommand(downloaderCmd)
	downloaderInit()
	eveCmd.AddCommand(startEveCmd)
	eveCmd.AddCommand(stopEveCmd)
	eveCmd.AddCommand(statusEveCmd)
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	startEveCmd.Flags().StringVarP(&eveImageFile, "image-file", "", "", "path for image drive (required)")
	startEveCmd.Flags().StringVarP(&qemuARCH, "eve-arch", "", runtime.GOARCH, "arch of system")
	startEveCmd.Flags().StringVarP(&qemuOS, "eve-os", "", runtime.GOOS, "os to run on")
	startEveCmd.Flags().BoolVarP(&qemuAccel, "eve-accel", "", true, "use acceleration")
	startEveCmd.Flags().StringVarP(&qemuSMBIOSSerial, "eve-serial", "", "", "SMBIOS serial")
	startEveCmd.Flags().StringVarP(&qemuConfigFile, "qemu-config", "", filepath.Join(currentPath, "dist", "qemu.conf"), "config file to use")
	startEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, "dist", "eve.pid"), "file for save EVE pid")
	startEveCmd.Flags().StringVarP(&eveLogFile, "eve-log", "", filepath.Join(currentPath, "dist", "eve.log"), "file for save EVE log")
	stopEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, "dist", "eve.pid"), "file for save EVE pid")
	statusEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, "dist", "eve.pid"), "file for save EVE pid")
	eveCmd.PersistentFlags().StringVar(&config, "config", "", "path to config file")
}
