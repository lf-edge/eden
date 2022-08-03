// Copyright (c) 2020 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var verbosity string
var configName string
var configFile string

var rootCmd = &cobra.Command{Use: "eden", PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
	configNameEnv := os.Getenv(defaults.DefaultConfigEnv)
	if configNameEnv != "" {
		configName = configNameEnv
	}
	configFile = utils.GetConfig(configName)
	if verbosity == "debug" {
		fmt.Println("configName: ", configName)
		fmt.Println("configFile: ", configFile)
	}
	return setUpLogs(os.Stdout, verbosity)
}}

func setUpLogs(out io.Writer, level string) error {
	log.SetOutput(out)
	lvl, err := log.ParseLevel(level)
	if err != nil {
		return err
	}
	log.SetLevel(lvl)
	return nil
}

func assignCobraToViper(cmd *cobra.Command) {
	for k, v := range defaults.DefaultCobraToViper {
		if flag := cmd.Flag(v); flag != nil {
			_ = viper.BindPFlag(k, flag)
		}
	}
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoInit()
	rootCmd.AddCommand(logCmd)
	logInit()
	rootCmd.AddCommand(netStatCmd)
	netStatInit()
	rootCmd.AddCommand(metricCmd)
	metricInit()
	rootCmd.AddCommand(startCmd)
	startInit()
	rootCmd.AddCommand(stopCmd)
	stopInit()
	rootCmd.AddCommand(statusCmd)
	statusInit()
	rootCmd.AddCommand(eveCmd)
	eveInit()
	rootCmd.AddCommand(adamCmd)
	adamInit()
	rootCmd.AddCommand(registryCmd)
	registryInit()
	rootCmd.AddCommand(redisCmd)
	redisInit()
	rootCmd.AddCommand(eserverCmd)
	eserverInit()
	rootCmd.AddCommand(configCmd)
	configInit()
	rootCmd.AddCommand(cleanCmd)
	cleanInit()
	rootCmd.AddCommand(setupCmd)
	setupInit()
	rootCmd.AddCommand(testCmd)
	testInit()
	rootCmd.AddCommand(utilsCmd)
	utilsInit()
	rootCmd.AddCommand(controllerCmd)
	controllerInit()
	rootCmd.AddCommand(podCmd)
	podInit()
	eciInit()
	rootCmd.AddCommand(networkCmd)
	networkInit()
	exportImportInit()
	rootCmd.AddCommand(volumeCmd)
	volumeInit()
	rootCmd.AddCommand(packetCmd)
	packetInit()
	rootCmd.AddCommand(rolCmd)
	rolInit()
	rootCmd.AddCommand(disksCmd)
	disksInit()
	rootCmd.AddCommand(sdnCmd)
	sdnInit()
}

// Execute primary function for cobra
func Execute() {
	rootCmd.PersistentFlags().StringVar(&configName, "config", defaults.DefaultContext, "Name of config")
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", log.InfoLevel.String(), "Log level (debug, info, warn, error, fatal, panic")
	_ = rootCmd.Execute()
}
