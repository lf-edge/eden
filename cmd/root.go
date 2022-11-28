// Copyright (c) 2020 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"reflect"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewEdenCommand() *cobra.Command {
	var configName, verbosity string

	rootCmd := &cobra.Command{
		Use: "eden",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return openevec.SetUpLogs(verbosity)
		},
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newSetupCmd(&configName, &verbosity),
				newStartCmd(&configName, &verbosity),
				newEveCmd(&configName, &verbosity),
				newPodCmd(&configName, &verbosity),
				newStatusCmd(&configName, &verbosity),
				newStopCmd(&configName, &verbosity),
				newCleanCmd(&configName, &verbosity),
				newConfigCmd(&configName, &verbosity),
				newSdnCmd(&configName, &verbosity),
			},
		},
		{
			Message: "Advanced Commands",
			Commands: []*cobra.Command{
				newInfoCmd(&configName, &verbosity),
				newLogCmd(&configName, &verbosity),
				newNetStatCmd(&configName, &verbosity),
				newMetricCmd(&configName, &verbosity),
				newAdamCmd(&configName, &verbosity),
				newRegistryCmd(&configName, &verbosity),
				newRedisCmd(&configName, &verbosity),
				newEserverCmd(&configName, &verbosity),
				newTestCmd(&configName, &verbosity),
				newUtilsCmd(&configName, &verbosity),
				newControllerCmd(&configName, &verbosity),
				newNetworkCmd(),
				newVolumeCmd(),
				newDisksCmd(),
				newPacketCmd(&configName, &verbosity),
				newRolCmd(&configName, &verbosity),
			},
		},
	}

	groups.AddTo(rootCmd)

	rootCmd.PersistentFlags().StringVar(&configName, "config", defaults.DefaultContext, "Name of config")
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", log.InfoLevel.String(), "Log level (debug, info, warn, error, fatal, panic")

	return rootCmd
}

func preRunViperLoadFunction(cfg *openevec.EdenSetupArgs, configName, verbosity *string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		viperCfg, err := openevec.FromViper(*configName, *verbosity)
		if err != nil {
			return err
		}
		openevec.Merge(reflect.ValueOf(viperCfg).Elem(), reflect.ValueOf(*cfg), cmd.Flags())
		*cfg = *viperCfg
		return nil
	}
}

// Execute primary function for cobra
func Execute() {
	rootCmd := NewEdenCommand()
	_ = rootCmd.Execute()
}
