package cmd

import (
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newControllerCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var controllerMode string

	var controllerCmd = &cobra.Command{
		Use:               "controller",
		Short:             "interact with controller",
		Long:              `Interact with controller.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	var edgeNode = &cobra.Command{
		Use:   "edge-node",
		Short: "manage EVE instance",
		Long:  `Manage EVE instance.`,
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newEdgeNodeReboot(controllerMode),
				newEdgeNodeShutdown(controllerMode),
				newEdgeNodeEVEImageUpdate(controllerMode, cfg),
				newEdgeNodeEVEImageRemove(controllerMode, cfg),
				newEdgeNodeEVEImageUpdateRetry(controllerMode),
				newEdgeNodeUpdate(controllerMode),
				newEdgeNodeGetConfig(controllerMode),
				newEdgeNodeSetConfig(),
				newEdgeNodeGetOptions(controllerMode),
				newEdgeNodeSetOptions(controllerMode),
			},
		},
	}

	groups.AddTo(edgeNode)

	controllerCmd.AddCommand(edgeNode)

	controllerCmd.AddCommand(newControllerGetOptions())
	controllerCmd.AddCommand(newControllerSetOptions())

	controllerCmd.PersistentFlags().StringVarP(&controllerMode, "mode", "m", "", "mode to use [file|proto|adam|zedcloud]://<URL> (default is adam)")

	return controllerCmd
}

func newEdgeNodeReboot(controllerMode string) *cobra.Command {
	var edgeNodeReboot = &cobra.Command{
		Use:   "reboot",
		Short: "reboot EVE instance",
		Long:  `reboot EVE instance.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.EdgeNodeReboot(controllerMode); err != nil {
				log.Fatal(err)
			}
		},
	}
	return edgeNodeReboot
}

func newEdgeNodeEVEImageUpdateRetry(controllerMode string) *cobra.Command {
	var edgeNodeEVEImageUpdateRetry = &cobra.Command{
		Use:   "eveimage-update-retry",
		Short: "retry update of EVE image",
		Long:  `Update EVE image retry.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.EdgeNodeEVEImageUpdateRetry(controllerMode); err != nil {
				log.Fatal(err)
			}
		},
	}
	return edgeNodeEVEImageUpdateRetry
}

func newEdgeNodeShutdown(controllerMode string) *cobra.Command {
	var edgeNodeShutdown = &cobra.Command{
		Use:   "shutdown",
		Short: "shutdown EVE app instances",
		Long:  `shutdown EVE app instances.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.EdgeNodeShutdown(controllerMode); err != nil {
				log.Fatal(err)
			}
		},
	}

	return edgeNodeShutdown
}

func newEdgeNodeEVEImageUpdate(controllerMode string, cfg *openevec.EdenSetupArgs) *cobra.Command {
	var baseOSVersion, registry string
	var baseOSImageActivate, baseOSVDrive bool

	var edgeNodeEVEImageUpdate = &cobra.Command{
		Use:   "eveimage-update <image file or url (oci:// or file:// or http(s)://)>",
		Short: "update EVE image",
		Long:  `Update EVE image.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			baseOSImage := args[0]
			if err := openevec.EdgeNodeEVEImageUpdate(baseOSImage, baseOSVersion, registry, controllerMode, baseOSImageActivate, baseOSVDrive); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeEVEImageUpdate.Flags().StringVarP(&baseOSVersion, "os-version", "", "", "version of ROOTFS")
	edgeNodeEVEImageUpdate.Flags().StringVar(&registry, "registry", "remote", "Select registry to use for containers (remote/local)")
	edgeNodeEVEImageUpdate.Flags().BoolVarP(&baseOSImageActivate, "activate", "", true, "activate image")
	edgeNodeEVEImageUpdate.Flags().BoolVar(&baseOSVDrive, "drive", true, "provide drive to baseOS")

	return edgeNodeEVEImageUpdate
}

func newEdgeNodeEVEImageRemove(controllerMode string, cfg *openevec.EdenSetupArgs) *cobra.Command {
	var baseOSVersion string

	var edgeNodeEVEImageRemove = &cobra.Command{
		Use:   "eveimage-remove <image file or url (oci:// or file:// or http(s)://)>",
		Short: "remove EVE image",
		Long:  `Remove EVE image.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			baseOSImage := args[0]
			if err := openevec.EdgeNodeEVEImageRemove(controllerMode, baseOSVersion, baseOSImage, cfg.Eden.Dist); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeEVEImageRemove.Flags().StringVarP(&baseOSVersion, "os-version", "", "", "version of ROOTFS")
	// TODO: NOT USED
	//edgeNodeEVEImageRemove.Flags().StringVar(&registry, "registry", "remote", "Select registry to use for containers (remote/local)")

	return edgeNodeEVEImageRemove
}

func newEdgeNodeUpdate(controllerMode string) *cobra.Command {
	var deviceItems, configItems map[string]string

	var edgeNodeUpdate = &cobra.Command{
		Use:   "update --config key=value --device key=value",
		Short: "update EVE config",
		Long:  `Update EVE config.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.EdgeNodeUpdate(controllerMode, deviceItems, configItems); err != nil {
				log.Fatal(err)
			}
		},
	}

	configUsage := `set of key=value items.
Supported keys are defined in https://github.com/lf-edge/eve/blob/master/docs/CONFIG-PROPERTIES.md`
	deviceUsage := `set of key=value items.
Supported keys: global_profile,local_profile_server,profile_server_token`
	edgeNodeUpdate.Flags().StringToStringVar(&configItems, "config", make(map[string]string), configUsage)
	edgeNodeUpdate.Flags().StringToStringVar(&deviceItems, "device", make(map[string]string), deviceUsage)

	return edgeNodeUpdate
}

func newEdgeNodeGetOptions(controllerMode string) *cobra.Command {
	var fileWithConfig string

	var edgeNodeGetOptions = &cobra.Command{
		Use:   "get-options",
		Short: "fetch EVE options",
		Long:  `Fetch EVE options.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.EdgeNodeGetOptions(controllerMode, fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeGetOptions.Flags().StringVar(&fileWithConfig, "file", "", "save options to file")

	return edgeNodeGetOptions
}

func newEdgeNodeSetOptions(controllerMode string) *cobra.Command {
	var fileWithConfig string

	var edgeNodeSetOptions = &cobra.Command{
		Use:   "set-options",
		Short: "set EVE options",
		Long:  `Set EVE options.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.EdgeNodeSetOptions(controllerMode, fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeSetOptions.Flags().StringVar(&fileWithConfig, "file", "", "set options from file")

	return edgeNodeSetOptions
}

func newControllerGetOptions() *cobra.Command {
	var fileWithConfig string

	var controllerGetOptions = &cobra.Command{
		Use:   "get-options",
		Short: "fetch controller options",
		Long:  `Fetch controller options.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.ControllerGetOptions(fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	controllerGetOptions.Flags().StringVar(&fileWithConfig, "file", "", "save options to file")

	return controllerGetOptions
}

func newControllerSetOptions() *cobra.Command {
	var fileWithConfig string

	var controllerSetOptions = &cobra.Command{
		Use:   "set-options",
		Short: "set controller options",
		Long:  `Set controller options.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.ControllerSetOptions(fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	controllerSetOptions.Flags().StringVar(&fileWithConfig, "file", "", "set options from file")

	return controllerSetOptions
}

func newEdgeNodeGetConfig(controllerMode string) *cobra.Command {
	var fileWithConfig string

	var edgeNodeGetConfig = &cobra.Command{
		Use:   "get-config",
		Short: "fetch EVE config",
		Long:  `Fetch EVE config.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.EdgeNodeGetConfig(controllerMode, fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeGetConfig.Flags().StringVar(&fileWithConfig, "file", "", "save config to file")

	return edgeNodeGetConfig
}

func newEdgeNodeSetConfig() *cobra.Command {
	var fileWithConfig string

	var edgeNodeSetConfig = &cobra.Command{
		Use:   "set-config",
		Short: "set EVE config",
		Long:  `Set EVE config.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.EdgeNodeSetConfig(fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeSetConfig.Flags().StringVar(&fileWithConfig, "file", "", "set config from file")

	return edgeNodeSetConfig
}
