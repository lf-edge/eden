// Copyright (c) 2020 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"log"
	"os"
)

var verbosity string
var configFile string
var rootCmd = &cobra.Command{Use: "eden", PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
	if err := setUpLogs(os.Stdout, verbosity); err != nil {
		return err
	}
	return nil
}}

func setUpLogs(out io.Writer, level string) error {
	logrus.SetOutput(out)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)
	return nil
}

func assingCobraToViper(cmd *cobra.Command) {
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
	rootCmd.AddCommand(metricCmd)
	metricInit()
	rootCmd.AddCommand(certsCmd)
	certsInit()
	rootCmd.AddCommand(ociImageCmd)
	ociImageInit()
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
	rootCmd.AddCommand(reconfCmd)
	reconfInit()
	rootCmd.AddCommand(testCmd)
	testInit()
	rootCmd.AddCommand(utilsCmd)
	utilsInit()
	rootCmd.AddCommand(controllerCmd)
	controllerInit()
	rootCmd.AddCommand(podCmd)
	podInit()
}

// Execute primary function for cobra
func Execute() {
	configPath, err := utils.DefaultConfigPath()
	if err != nil {
		log.Fatal(err)
	}
	rootCmd.PersistentFlags().StringVar(&configFile, "config-file", configPath, "path to config file")
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", logrus.InfoLevel.String(), "Log level (debug, info, warn, error, fatal, panic")
	_ = rootCmd.Execute()
}
