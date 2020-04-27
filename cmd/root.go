// Copyright (c) 2020 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"os"
)

var verbosity string
var config string
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
	for k, v := range cobraToViper {
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
	rootCmd.AddCommand(certsCmd)
	certsInit()
	rootCmd.AddCommand(serverCmd)
	serverInit()
	rootCmd.AddCommand(logwatchCmd)
	logwatchInit()
	rootCmd.AddCommand(infowatchCmd)
	infowatchInit()
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
	rootCmd.AddCommand(eserverCmd)
	eserverInit()
	rootCmd.AddCommand(configCmd)
	configInit()
	rootCmd.AddCommand(cleanCmd)
	cleanInit()
}

// Execute primary function for cobra
func Execute() {
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", logrus.WarnLevel.String(), "Log level (debug, info, warn, error, fatal, panic")
	_ = rootCmd.Execute()
}
