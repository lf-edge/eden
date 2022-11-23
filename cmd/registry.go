package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newRegistryCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var registryCmd = &cobra.Command{
		Use:               "registry",
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newStartRegistryCmd(cfg),
				newStopRegistryCmd(cfg),
				newStatusRegistryCmd(),
				newLoadRegistryCmd(cfg),
			},
		},
	}

	groups.AddTo(registryCmd)

	return registryCmd
}

func newStartRegistryCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var startRegistryCmd = &cobra.Command{
		Use:   "start",
		Short: "start registry",
		Long:  `Start OCI/docker registry.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.RegistryStart(&cfg.Registry); err != nil {
				log.Fatal("Registry start failed %s", err)
			}
		},
	}

	startRegistryCmd.Flags().StringVarP(&cfg.Registry.Tag, "registry-tag", "", defaults.DefaultRegistryTag, "tag on registry container to pull")
	startRegistryCmd.Flags().IntVarP(&cfg.Registry.Port, "registry-port", "", defaults.DefaultRegistryPort, "registry port to start")
	startRegistryCmd.Flags().StringVarP(&cfg.Registry.Dist, "registry-dist", "", cfg.Registry.Dist, "registry dist path to store (required)")

	return startRegistryCmd
}

func newStopRegistryCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var registryRm bool

	var stopRegistryCmd = &cobra.Command{
		Use:   "stop",
		Short: "stop registry",
		Long:  `Stop OCI/docker registry.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := eden.StopRegistry(registryRm); err != nil {
				log.Errorf("cannot stop registry: %s", err)
			}
		},
	}

	stopRegistryCmd.Flags().BoolVarP(&registryRm, "registry-rm", "", false, "registry rm on stop")

	return stopRegistryCmd
}

func newStatusRegistryCmd() *cobra.Command {
	var statusRegistryCmd = &cobra.Command{
		Use:   "status",
		Short: "status of registry",
		Long:  `Status of OCI/docker registry.`,
		Run: func(cmd *cobra.Command, args []string) {
			statusRegistry, err := eden.StatusRegistry()
			if err != nil {
				log.Errorf("cannot obtain status of registry: %s", err)
			} else {
				fmt.Printf("Registry status: %s\n", statusRegistry)
			}
		},
	}
	return statusRegistryCmd
}

func newLoadRegistryCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var loadRegistryCmd = &cobra.Command{
		Use:   "load <image>",
		Short: "load image into registry",
		Long: `load image into registry. First attempts local docker image cache.
	If it fails, pull from remote to local, and then load.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ref := args[0]
			if err := openevec.RegistryLoad(ref, &cfg.Registry); err != nil {
				log.Fatalf("Load registry failed %s", err)
			}
		},
	}

	//TODO: Why are we not linking  registry IP and PORT here?

	return loadRegistryCmd
}
