// Copyright (c) 2020 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var verbosity string
var config string
var rootCmd = &cobra.Command{Use: "eden", PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
	if err := setUpLogs(os.Stdout, verbosity); err != nil {
		return err
	}
	return nil
}}

func loadViperConfig() (loaded bool, err error) {
	if config != "" {
		abs, err := filepath.Abs(config)
		if err != nil {
			return false, fmt.Errorf("fail in reading filepath: %s", err.Error())
		}
		base := filepath.Base(abs)
		path := filepath.Dir(abs)
		viper.SetConfigName(strings.Split(base, ".")[0])
		viper.AddConfigPath(path)
		if err := viper.ReadInConfig(); err != nil {
			return false, fmt.Errorf("failed to read config file: %s", err.Error())
		}
		return true, nil
	}
	return false, nil
}

func setUpLogs(out io.Writer, level string) error {
	logrus.SetOutput(out)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)
	return nil
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
	rootCmd.AddCommand(qemuConfCmd)
	qemuConfInit()
	rootCmd.AddCommand(qemuRunCmd)
	qemuRunInit()
	rootCmd.AddCommand(startCmd)
	startInit()
	rootCmd.AddCommand(stopCmd)
	stopInit()
	rootCmd.AddCommand(statusCmd)
	statusInit()
}

// Execute primary function for cobra
func Execute() {
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", logrus.WarnLevel.String(), "Log level (debug, info, warn, error, fatal, panic")
	_ = rootCmd.Execute()
}
