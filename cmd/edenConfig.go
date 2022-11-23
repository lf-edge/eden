package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newConfigCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var configCmd = &cobra.Command{
		Use:               "config",
		Short:             "work with config",
		Long:              `Work with config.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newConfigAddCmd(cfg),
				newConfigDeleteCmd(cfg),
				newConfigGetCmd(),
				newConfigSetCmd(),
				newConfigListCmd(),
				newConfigResetCmd(),
				newConfigEditCmd(),
			},
		},
	}

	groups.AddTo(configCmd)

	return configCmd
}

func newConfigAddCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var force bool

	var configAddCmd = &cobra.Command{
		Use:   "add [name]",
		Long:  "Generate config context for eden with defined name ('default' by default)",
		Short: `Generate config context for eden with defined name ('default' by default).`,
		Args:  cobra.MaximumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},

		Run: func(cmd *cobra.Command, args []string) {
			configName := "default"
			if len(args) > 0 {
				configName = args[0]
			}
			if err := openevec.ConfigAdd(cfg, configName, force); err != nil {
				log.Fatal(err)
			}
		},
	}

	configAddCmd.Flags().StringVar(&cfg.Eve.DevModel, "devmodel", defaults.DefaultQemuModel,
		fmt.Sprintf("device model (%s/%s/%s/%s)",
			defaults.DefaultQemuModel, defaults.DefaultRPIModel, defaults.DefaultGCPModel, defaults.DefaultGeneralModel))
	configAddCmd.Flags().StringVar(&cfg.Runtime.ContextFile, "file", "", "file with config to add")
	//not used in function
	configAddCmd.Flags().StringVarP(&cfg.Eve.QemuFileToSave, "qemu-config", "", defaults.DefaultQemuFileToSave, "file to save config")
	configAddCmd.Flags().IntVarP(&cfg.Eve.QemuCpus, "cpus", "", defaults.DefaultCpus, "cpus")
	configAddCmd.Flags().IntVarP(&cfg.Eve.QemuMemory, "memory", "", defaults.DefaultMemory, "memory (MB)")
	configAddCmd.Flags().IntVarP(&cfg.Eve.QemuUsbSerials, "usbserials", "", 0, "number of USB serial adapters")    // !
	configAddCmd.Flags().IntVarP(&cfg.Eve.QemuUsbTablets, "usbtablets", "", 0, "number of USB tablet controllers") // !
	configAddCmd.Flags().StringSliceVarP(&cfg.Eve.QemuFirmware, "eve-firmware", "", nil, "firmware path")
	configAddCmd.Flags().StringVarP(&cfg.Eve.QemuConfigPath, "config-part", "", "", "path for config drive")
	configAddCmd.Flags().StringVarP(&cfg.Eve.QemuDTBPath, "dtb-part", "", "", "path for device tree drive (for arm)")
	configAddCmd.Flags().StringToStringVarP(&cfg.Eve.HostFwd, "eve-hostfwd", "", defaults.DefaultQemuHostFwd, "port forward map")
	configAddCmd.Flags().StringVarP(&cfg.Eve.QemuSocketPath, "qmp", "", "", "use qmp socket with path") //!!
	configAddCmd.Flags().StringVar(&cfg.Eve.Ssid, "ssid", "", "set ssid of wifi for rpi")
	configAddCmd.Flags().StringVar(&cfg.Eve.Arch, "arch", "", "arch of EVE (amd64 or arm64)")
	configAddCmd.Flags().StringVar(&cfg.Eve.ModelFile, "devmodel-file", "", "File to use for overwrite of model defaults")
	// TODO: I've added this flag, needed to be checked
	configAddCmd.Flags().BoolVarP(&force, "force", "", false, "force overwrite config file")

	return configAddCmd
}

func newConfigListCmd() *cobra.Command {
	var configListCmd = &cobra.Command{
		Use:   "list",
		Short: "List config contexts",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.ConfigList(); err != nil {
				log.Fatal(err)
			}
		},
	}

	return configListCmd
}

func newConfigSetCmd() *cobra.Command {
	var contextKeySet, contextValueSet string

	var configSetCmd = &cobra.Command{
		Use:   "set <name>",
		Short: "Set current context to name",
		Long:  "Set current context to name \n\t will only modify key for name context if --key not empty",
		Args:  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.ConfigSet(args[0], contextKeySet, contextValueSet); err != nil {
				log.Fatal(err)
			}
		},
	}

	configSetCmd.Flags().StringVar(&contextKeySet, "key", "", "will set value of key from current config context")
	configSetCmd.Flags().StringVar(&contextValueSet, "value", "", "will set value of key from current config context")

	return configSetCmd
}

func newConfigEditCmd() *cobra.Command {
	var configEditCmd = &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit current or context with defined name with $EDITOR",
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			if err := openevec.ConfigEdit(target); err != nil {
				log.Fatal(err)
			}
		},
	}
	return configEditCmd
}

func newConfigResetCmd() *cobra.Command {
	var configResetCmd = &cobra.Command{
		Use:   "reset [name]",
		Short: "Reset current or context with defined name to defaults",
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			if err := openevec.ConfigReset(target); err != nil {
				log.Fatal(err)
			}
		},
	}
	return configResetCmd
}

func newConfigGetCmd() *cobra.Command {
	var contextKeyGet string
	var contextAllGet bool

	var configGetCmd = &cobra.Command{
		Use:   "get [name]",
		Short: "get config context for current or defined name",
		Long:  "Get config context for current or defined name. \n\tif --key set will show selected key only\n\tif --all set will return complete config",
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			contextNameGet := ""
			if len(args) == 1 {
				contextNameGet = args[0]
			}
			if err := openevec.ConfigGet(contextNameGet, contextKeyGet, contextAllGet); err != nil {
				log.Fatal(err)
			}
		},
	}

	configGetCmd.Flags().StringVar(&contextKeyGet, "key", "", "will return value of key from current config context")
	configGetCmd.Flags().BoolVar(&contextAllGet, "all", false, "will return config context")

	return configGetCmd
}

func newConfigDeleteCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var configDeleteCmd = &cobra.Command{
		Use:   "delete <name>",
		Short: "delete config context",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			contextNameDel := args[0]
			if err := openevec.ConfigDelete(contextNameDel, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	return configDeleteCmd
}
