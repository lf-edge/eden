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
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			qemuARCH = viper.GetString("eve-arch")
			qemuOS = viper.GetString("eve-os")
			qemuAccel = viper.GetBool("eve-accel")
			qemuSMBIOSSerial = viper.GetString("eve-serial")
			qemuConfigFile = viper.GetString("eve-config")
			evePidFile = viper.GetString("eve-pid")
			eveLogFile = viper.GetString("eve-log")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		if err := utils.StartEVEQemu(command, qemuARCH, qemuOS, qemuSMBIOSSerial, qemuAccel, qemuConfigFile, eveLogFile, evePidFile); err != nil {
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
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = viper.GetString("eve-pid")
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
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = viper.GetString("eve-pid")
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
	startEveCmd.Flags().StringVarP(&qemuARCH, "eve-arch", "", runtime.GOARCH, "arch of system")
	startEveCmd.Flags().StringVarP(&qemuOS, "eve-os", "", runtime.GOOS, "os to run on")
	startEveCmd.Flags().BoolVarP(&qemuAccel, "eve-accel", "", true, "use acceleration")
	startEveCmd.Flags().StringVarP(&qemuSMBIOSSerial, "eve-serial", "", "", "SMBIOS serial")
	startEveCmd.Flags().StringVarP(&qemuConfigFile, "eve-config", "", filepath.Join(currentPath, "dist", "qemu.conf"), "config file to use")
	startEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, "dist", "eve.pid"), "file for save EVE pid")
	startEveCmd.Flags().StringVarP(&eveLogFile, "eve-log", "", filepath.Join(currentPath, "dist", "eve.log"), "file for save EVE log")
	if err := viper.BindPFlags(startEveCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	stopEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, "dist", "eve.pid"), "file for save EVE pid")
	if err := viper.BindPFlags(stopEveCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	statusEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, "dist", "eve.pid"), "file for save EVE pid")
	if err := viper.BindPFlags(statusEveCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	eveCmd.PersistentFlags().StringVar(&config, "config", "", "path to config file")
}
