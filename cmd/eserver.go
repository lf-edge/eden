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

var eserverCmd = &cobra.Command{
	Use: "eserver",
}

var startEserverCmd = &cobra.Command{
	Use:   "start",
	Short: "start eserver",
	Long:  `Start eserver.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eserverImageDist = viper.GetString("image-dist")
			eserverPort = viper.GetString("eserver-port")
			eserverPidFile = viper.GetString("eserver-pid")
			eserverLogFile = viper.GetString("eserver-log")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		log.Infof("Executable path: %s", command)
		if err := utils.StartEServer(command, eserverPort, eserverImageDist, eserverLogFile, eserverPidFile); err != nil {
			log.Errorf("cannot start eserver: %s", err)
		} else {
			log.Infof("Eserver is running and accesible on port %s", eserverPort)
		}
	},
}

var stopEserverCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop eserver",
	Long:  `Stop eserver.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eserverPidFile = viper.GetString("eserver-pid")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := utils.StopEServer(eserverPidFile); err != nil {
			log.Errorf("cannot stop eserver: %s", err)
		}
	},
}

var statusEserverCmd = &cobra.Command{
	Use:   "status",
	Short: "status of eserver",
	Long:  `Status of eserver.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		viperLoaded, err := utils.LoadConfigFile(config)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eserverPidFile = viper.GetString("eserver-pid")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		statusEServer, err := utils.StatusEServer(eserverPidFile)
		if err != nil {
			log.Errorf("cannot obtain status of eserver: %s", err)
		} else {
			fmt.Printf("EServer status: %s\n", statusEServer)
		}
	},
}

func eserverInit() {
	eserverCmd.AddCommand(startEserverCmd)
	eserverCmd.AddCommand(stopEserverCmd)
	eserverCmd.AddCommand(statusEserverCmd)
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	startEserverCmd.Flags().StringVarP(&eserverImageDist, "image-dist", "", filepath.Join(currentPath, "dist", "images"), "image dist for eserver")
	startEserverCmd.Flags().StringVarP(&eserverPort, "eserver-port", "", "8888", "eserver port")
	startEserverCmd.Flags().StringVarP(&eserverPidFile, "eserver-pid", "", filepath.Join(currentPath, "dist", "eserver.pid"), "file for save eserver pid")
	startEserverCmd.Flags().StringVarP(&eserverLogFile, "eserver-log", "", filepath.Join(currentPath, "dist", "eserver.log"), "file for save eserver log")
	if err := viper.BindPFlags(startEserverCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	stopEserverCmd.Flags().StringVarP(&eserverPidFile, "eserver-pid", "", filepath.Join(currentPath, "dist", "eserver.pid"), "file for save eserver pid")
	if err := viper.BindPFlags(stopEserverCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	statusEserverCmd.Flags().StringVarP(&eserverPidFile, "eserver-pid", "", filepath.Join(currentPath, "dist", "eserver.pid"), "file for save eserver pid")
	if err := viper.BindPFlags(statusEserverCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	eserverCmd.PersistentFlags().StringVar(&config, "config", "", "path to config file")
}
