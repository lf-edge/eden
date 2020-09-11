package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/eden"
	"os"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	registryTag  string
	registryPort int
	registryDist string
	registryRm   bool
)

var registryCmd = &cobra.Command{
	Use: "registry",
}

var startRegistryCmd = &cobra.Command{
	Use:   "start",
	Short: "start registry",
	Long:  `Start OCI/docker registry.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			registryTag = viper.GetString("registry.tag")
			registryPort = viper.GetInt("registry.port")
			registryDist = utils.ResolveAbsPath(viper.GetString("registry.dist"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		log.Infof("Executable path: %s", command)
		if err := eden.StartRegistry(registryPort, registryTag, registryDist); err != nil {
			log.Errorf("cannot start registry: %s", err)
		} else {
			log.Infof("registry is running and accessible on port %d", adamPort)
		}
	},
}

var stopRegistryCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop registry",
	Long:  `Stop OCI/docker registry.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := eden.StopRegistry(registryRm); err != nil {
			log.Errorf("cannot stop registry: %s", err)
		}
	},
}

var statusRegistryCmd = &cobra.Command{
	Use:   "status",
	Short: "status of registry",
	Long:  `Status of OCI/docker registry.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		statusRegistry, err := eden.StatusRegistry()
		if err != nil {
			log.Errorf("cannot obtain status of registry: %s", err)
		} else {
			fmt.Printf("Registry status: %s\n", statusRegistry)
		}
	},
}

var loadRegistryCmd = &cobra.Command{
	Use:   "load <image>",
	Short: "load image into registry",
	Long: `load image into registry. First attempts local docker image cache.
	If it fails, pull from remote to local, and then load.`,
	Args: cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ref := args[0]
		registry := fmt.Sprintf("%s:%d", viper.GetString("registry.ip"), viper.GetInt("registry.port"))
		hash, err := utils.LoadRegistry(ref, registry)
		if err != nil {
			log.Errorf("failed to load image %s: %v", ref, err)
			os.Exit(1)
		}
		fmt.Printf("image %s loaded with manifest hash %s", ref, hash)
	},
}

func registryInit() {
	registryCmd.AddCommand(startRegistryCmd)
	registryCmd.AddCommand(stopRegistryCmd)
	registryCmd.AddCommand(statusRegistryCmd)
	registryCmd.AddCommand(loadRegistryCmd)
	startRegistryCmd.Flags().StringVarP(&registryTag, "registry-tag", "", defaults.DefaultRegistryTag, "tag on registry container to pull")
	startRegistryCmd.Flags().IntVarP(&registryPort, "registry-port", "", defaults.DefaultRegistryPort, "registry port to start")
	startRegistryCmd.Flags().StringVarP(&registryDist, "registry-dist", "", "", "registry dist path to store (required)")
	stopRegistryCmd.Flags().BoolVarP(&registryRm, "registry-rm", "", false, "registry rm on stop")
}
