package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
)

var (
	qemuFileToSave  string
	qemuCpus        int
	qemuMemory      int
	qemuFirmware    []string
	qemuConfigPath  string
	eveImageFile    string
	qemuDTBPath     string
	qemuHostFwd     map[string]string
	qemuSocketPath  string
	qemuUsbSerials  int
	qemuUsbTablets  int
	contextFile     string
	contextKeySet   string
	contextValueSet string
	contextKeyGet   string
	contextAllGet   bool
	ssid            string
	password        string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "work with config",
	Long:  `Work with config.`,
}

func reloadConfigDetails() {
	viperLoaded, err := utils.LoadConfigFile(configFile)
	if err != nil {
		log.Fatalf("error reading config: %s", err)
	}
	if viperLoaded {
		qemuFirmware = viper.GetStringSlice("eve.firmware")
		qemuConfigPath = utils.ResolveAbsPath(viper.GetString("eve.config-part"))
		qemuDTBPath = utils.ResolveAbsPath(viper.GetString("eve.dtb-part"))
		eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
		qemuHostFwd = viper.GetStringMapString("eve.hostfwd")
		qemuFileToSave = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
		devModel = viper.GetString("eve.devmodel")
		eveRemote = viper.GetBool("eve.remote")
	}
}

var configAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "generate config context for eden with defined name",
	Long:  `Generate config context for eden.`,
	Args:  cobra.ExactValidArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if devModel != defaults.DefaultRPIModel && devModel != defaults.DefaultEVEModel && devModel != defaults.DefaultGCPModel {
			log.Fatal("unsupported model")
		}
		var err error
		if configFile == "" {
			configFile, err = utils.DefaultConfigPath()
			if err != nil {
				log.Fatalf("fail in DefaultConfigPath: %s", err)
			}
		}
		if _, err := os.Stat(configFile); !os.IsNotExist(err) {
			if force {
				if err := os.Remove(configFile); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Debugf("current config already exists: %s", configFile)
			}
		}
		assignCobraToViper(cmd)
		if _, err = os.Stat(configFile); os.IsNotExist(err) {
			if err = utils.GenerateConfigFile(configFile); err != nil {
				log.Fatalf("fail in generate yaml: %s", err.Error())
			}
			log.Infof("Config file generated: %s", configFile)
		}
		reloadConfigDetails()
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		context, err := utils.ContextLoad()
		if err != nil {
			log.Fatalf("Load context error: %s", err)
		}
		currentContextName := context.Current
		context.Current = args[0]
		configFile = context.GetCurrentConfig()
		if contextFile != "" {
			if err := utils.CopyFile(contextFile, configFile); err != nil {
				log.Fatalf("Cannot copy file: %s", err)
			}
			log.Infof("Context file generated: %s", contextFile)
		} else {
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				if err = utils.GenerateConfigFileDiff(configFile, context); err != nil {
					log.Fatalf("error generate config: %s", err)
				}
				log.Infof("Context file generated: %s", configFile)
			} else {
				log.Debugf("Config file already exists %s", configFile)
			}
		}
		reloadConfigDetails()
		if devModel == defaults.DefaultRPIModel { //modify default settings according to RPI4 params
			eveRemote = true
			viper.Set("eve.hostfwd", map[string]string{})
			viper.Set("eve.devmodel", defaults.DefaultRPIModel)
			viper.Set("eve.arch", "arm64")
			viper.Set("eve.serial", "*")
			viper.Set("eve.remote", eveRemote)
			viper.Set("eve.remote-addr", "")
			if ssid != "" {
				viper.Set("eve.ssid", ssid)
			}
			if err = utils.GenerateConfigFileFromViper(); err != nil {
				log.Fatalf("error writing config: %s", err)
			}
		}
		if devModel == defaults.DefaultGCPModel { //modify default settings according to GCP params
			eveRemote = true
			viper.Set("eve.hostfwd", map[string]string{})
			viper.Set("eve.devmodel", defaults.DefaultGCPModel)
			viper.Set("eve.serial", "*")
			viper.Set("eve.remote", eveRemote)
			viper.Set("eve.remote-addr", "")
			if err = utils.GenerateConfigFileFromViper(); err != nil {
				log.Fatalf("error writing config: %s", err)
			}
		}
		context.SetContext(currentContextName)
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List config contexts",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		context, err := utils.ContextLoad()
		if err != nil {
			log.Fatalf("Load context error: %s", err)
		}
		currentContext := context.Current
		contexts := context.ListContexts()
		for _, el := range contexts {
			if el == currentContext {
				fmt.Println("* " + el)
			} else {
				fmt.Println(el)
			}
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "Set current context to name",
	Long:  "Set current context to name \n\t will only modify key for name context if --key not empty",
	Args:  cobra.ExactValidArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		context, err := utils.ContextLoad()
		if err != nil {
			log.Fatalf("Load context error: %s", err)
		}
		oldContext := context.Current
		if contextKeySet != "" {
			defer context.SetContext(oldContext) //restore context after modifications
		}
		contexts := context.ListContexts()
		for _, el := range contexts {
			if el == args[0] {
				context.SetContext(el)
				if contextKeySet != "" {
					_, err := utils.LoadConfigFileContext(context.GetCurrentConfig())
					if err != nil {
						log.Fatalf("error reading config: %s", err.Error())
					}
					viper.Set(contextKeySet, contextValueSet)
					if err = utils.GenerateConfigFileFromViper(); err != nil {
						log.Fatalf("error writing config: %s", err)
					}
				}
				log.Infof("Current context is: %s", el)
				return
			}
		}
		log.Fatalf("context not found %s", args[0])
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit current or context with defined name with $EDITOR",
	Args:  cobra.RangeArgs(0, 1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			log.Fatal("$EDITOR environment not set")
		}
		context, err := utils.ContextLoad()
		if err != nil {
			log.Fatalf("Load context error: %s", err)
		}

		contextNameEdit := context.Current
		if len(args) == 1 {
			contextNameEdit = args[0]
		}
		contexts := context.ListContexts()
		for _, el := range contexts {
			if el == contextNameEdit {
				context.Current = contextNameEdit
				if err = utils.RunCommandForeground(editor, context.GetCurrentConfig()); err != nil {
					log.Fatal(err)
				}
				return
			}
		}
		log.Fatalf("context not found %s", contextNameEdit)
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset [name]",
	Short: "Reset current or context with defined name to defaults",
	Args:  cobra.RangeArgs(0, 1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		context, err := utils.ContextLoad()
		if err != nil {
			log.Fatalf("Load context error: %s", err)
		}
		oldContext := context.Current
		defer context.SetContext(oldContext) //restore context after modifications

		contextNameReset := context.Current
		if len(args) == 1 {
			contextNameReset = args[0]
		}
		contexts := context.ListContexts()
		for _, el := range contexts {
			if el == contextNameReset {
				context.SetContext(el)
				if err = os.Remove(context.GetCurrentConfig()); err != nil {
					log.Fatalf("cannot delete old config file: %s", err)
				}
				_, err := utils.LoadConfigFile(context.GetCurrentConfig())
				if err != nil {
					log.Fatalf("error reading config: %s", err.Error())
				}
				return
			}
		}
		log.Fatalf("context not found %s", contextNameReset)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "get config context for current or defined name",
	Long:  "Get config context for current or defined name. \n\tif --key set will show selected key only\n\tif --all set will return complete config",
	Args:  cobra.RangeArgs(0, 1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		contextNameGet := ""
		if len(args) == 1 {
			contextNameGet = args[0]
		}
		context, err := utils.ContextLoad()
		if err != nil {
			log.Fatalf("Load context error: %s", err)
		}
		oldContext := context.Current
		defer context.SetContext(oldContext) //restore context after modifications
		if contextNameGet != "" {
			found := false
			contexts := context.ListContexts()
			for _, el := range contexts {
				if el == contextNameGet {
					context.SetContext(el)
					found = true
					break
				}
			}
			if !found {
				log.Fatalf("context not found %s", contextNameGet)
			}
			_, err := utils.LoadConfigFile(context.GetCurrentConfig())
			if err != nil {
				log.Fatalf("error reading config: %s", err.Error())
			}
		}
		if contextKeyGet == "" && !contextAllGet {
			fmt.Println(context.Current)
		} else if contextKeyGet != "" {
			fmt.Println(viper.Get(contextKeyGet))
		} else if contextAllGet {
			if err = viper.WriteConfigAs(defaults.DefaultConfigHidden); err != nil {
				log.Fatal(err)
			}
			data, err := ioutil.ReadFile(defaults.DefaultConfigHidden)
			if err != nil {
				log.Fatalf("cannot read context config file %s: %s", configFile, err)
				return
			}
			fmt.Print(string(data))
		}
	},
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "delete config context",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		contextNameDel := args[0]
		context, err := utils.ContextLoad()
		if err != nil {
			log.Fatalf("Load context error: %s", err)
		}
		currentContext := context.Current
		log.Infof("currentContext %s", currentContext)
		log.Infof("contextName %s", contextNameDel)
		if (contextNameDel == "" || contextNameDel == defaults.DefaultContext) && defaults.DefaultContext == currentContext {
			log.Fatal("Cannot delete default context. Use 'eden clean' instead.")
		}
		if contextNameDel == currentContext {
			contextNameDel = context.Current
			context.SetContext(defaults.DefaultContext)
			log.Infof("Move to %s context", defaults.DefaultContext)
		}
		context.Current = contextNameDel
		configFile = context.GetCurrentConfig()
		reloadConfigDetails()
		if _, err := os.Stat(qemuFileToSave); !os.IsNotExist(err) {
			if err := os.Remove(qemuFileToSave); err == nil {
				log.Infof("deleted qemu config %s", qemuFileToSave)
			}
		}
		log.Infof("currentContextFile %s", configFile)
		if err := os.Remove(configFile); err != nil {
			log.Fatalf("Cannot delete context %s: %s", contextNameDel, err)
		}
	},
}

func configInit() {
	configCmd.AddCommand(configDeleteCmd)
	configCmd.AddCommand(configGetCmd)
	configGetCmd.Flags().StringVar(&contextKeyGet, "key", "", "will return value of key from current config context")
	configGetCmd.Flags().BoolVar(&contextAllGet, "all", false, "will return config context")
	configCmd.AddCommand(configSetCmd)
	configSetCmd.Flags().StringVar(&contextKeySet, "key", "", "will set value of key from current config context")
	configSetCmd.Flags().StringVar(&contextValueSet, "value", "", "will set value of key from current config context")
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configAddCmd)
	configAddCmd.Flags().StringVar(&devModel, "devmodel", defaults.DefaultEVEModel, fmt.Sprintf("device model (%s or %s)", defaults.DefaultRPIModel, defaults.DefaultEVEModel))
	configAddCmd.Flags().StringVar(&contextFile, "file", "", "file with config to add")
	configAddCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", defaults.DefaultQemuFileToSave, "file to save config")
	configAddCmd.Flags().IntVarP(&qemuCpus, "cpus", "", defaults.DefaultQemuCpus, "cpus")
	configAddCmd.Flags().IntVarP(&qemuMemory, "memory", "", defaults.DefaultQemuMemory, "memory (MB)")
	configAddCmd.Flags().IntVarP(&qemuUsbSerials, "usbserials", "", 0, "number of USB serial adapters")
	configAddCmd.Flags().IntVarP(&qemuUsbTablets, "usbtablets", "", 0, "number of USB tablet controllers")
	configAddCmd.Flags().StringSliceVarP(&qemuFirmware, "eve-firmware", "", nil, "firmware path")
	configAddCmd.Flags().StringVarP(&qemuConfigPath, "config-part", "", "", "path for config drive")
	configAddCmd.Flags().StringVarP(&qemuDTBPath, "dtb-part", "", "", "path for device tree drive (for arm)")
	configAddCmd.Flags().StringToStringVarP(&qemuHostFwd, "eve-hostfwd", "", defaults.DefaultQemuHostFwd, "port forward map")
	configAddCmd.Flags().StringVarP(&qemuSocketPath, "qmp", "", "", "use qmp socket with path")
	configAddCmd.Flags().StringVar(&ssid, "ssid", "", "set ssid of wifi for rpi")
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configEditCmd)
}
