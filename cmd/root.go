// Copyright (c) 2020 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "eden"}

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
	_ = rootCmd.Execute()
}
