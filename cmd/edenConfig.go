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
	"path/filepath"
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
	contextFile     string
	contextKeySet   string
	contextValueSet string
	contextKeyGet   string
	contextAllGet   bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "work with config",
	Long:  `Work with config.`,
}

var configAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "generate config context for eden with defined name",
	Long:  `Generate config context for eden.`,
	Args:  cobra.ExactValidArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
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
		assingCobraToViper(cmd)
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
		}
		testSript := utils.ResolveAbsPath(viper.GetString("eden.test-script"))
		if _, err := os.Stat(testSript); !os.IsNotExist(err) {
			if force {
				if err := os.Remove(testSript); err != nil {
					log.Fatal(err)
				}
				err = utils.GenerateTestSript(testSript)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Debugf("Test script already exists: %s", testSript)
			}
		} else {
			err = utils.GenerateTestSript(testSript)
			if err != nil {
				log.Fatal(err)
			}
		}
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
			} else {
				log.Infof("Context file generated: %s", contextFile)
			}
		}
		_, err = utils.LoadConfigFile(configFile)
		if err != nil {
			log.Fatalf("error reading config: %s", err)
		}
		context.SetContext(currentContextName)
		if _, err := os.Stat(qemuFileToSave); os.IsNotExist(err) {
			f, err := os.Create(qemuFileToSave)
			if err != nil {
				log.Fatal(err)
			}
			qemuConfigPathAbsolute := ""
			if qemuConfigPath != "" {
				qemuConfigPathAbsolute, err = filepath.Abs(qemuConfigPath)
				if err != nil {
					log.Fatal(err)
				}
			}
			qemuDTBPathAbsolute := ""
			if qemuDTBPath != "" {
				qemuDTBPathAbsolute, err = filepath.Abs(qemuDTBPath)
				if err != nil {
					log.Fatal(err)
				}
			}
			var qemuFirmwareParam []string
			for _, el := range qemuFirmware {
				qemuFirmwarePathAbsolute := utils.ResolveAbsPath(el)
				if err != nil {
					log.Fatal(err)
				}
				qemuFirmwareParam = append(qemuFirmwareParam, qemuFirmwarePathAbsolute)
			}
			//generate netdevs with unused subnets
			nets, err := utils.GetSubnetsNotUsed(2)
			if err != nil {
				log.Fatal(err)
			}
			settings := utils.QemuSettings{
				ConfigDrive: qemuConfigPathAbsolute,
				DTBDrive:    qemuDTBPathAbsolute,
				Firmware:    qemuFirmwareParam,
				MemoryMB:    qemuMemory,
				CPUs:        qemuCpus,
				HostFWD:     qemuHostFwd,
				NetDevs:     nets,
			}
			conf, err := settings.GenerateQemuConfig()
			if err != nil {
				log.Fatal(err)
			}
			_, err = f.Write(conf)
			if err != nil {
				log.Fatal(err)
			}
			if err := f.Close(); err != nil {
				log.Fatal(err)
			}
			log.Infof("QEMU config file generated: %s", qemuFileToSave)
		} else {
			log.Debugf("QEMU config already exists: %s", qemuFileToSave)
		}
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List config contexts",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
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
		assingCobraToViper(cmd)
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
					_, err := utils.LoadConfigFile(context.GetCurrentConfig())
					if err != nil {
						log.Fatalf("error reading config: %s", err.Error())
					}
					viper.Set(contextKeySet, contextValueSet)
					if err = viper.WriteConfig(); err != nil {
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
		assingCobraToViper(cmd)
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
		assingCobraToViper(cmd)
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
		assingCobraToViper(cmd)
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
	Use:   "delete [name]",
	Short: "delete config context",
	Args:  cobra.RangeArgs(0, 1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		contextNameDel := ""
		if len(args) == 1 {
			contextNameDel = args[0]
		}
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
		if contextNameDel == "" {
			contextNameDel = context.Current
			context.SetContext(defaults.DefaultContext)
			log.Infof("Move to %s context", defaults.DefaultContext)
		}
		context.Current = contextNameDel
		currentContextFile := context.GetCurrentConfig()
		log.Infof("currentContextFile %s", currentContextFile)
		if err := os.Remove(currentContextFile); err != nil {
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
	configAddCmd.Flags().StringVar(&contextFile, "file", "", "file with config to add")
	configAddCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", defaults.DefaultQemuFileToSave, "file to save config")
	configAddCmd.Flags().IntVarP(&qemuCpus, "cpus", "", defaults.DefaultQemuCpus, "cpus")
	configAddCmd.Flags().IntVarP(&qemuMemory, "memory", "", defaults.DefaultQemuMemory, "memory (MB)")
	configAddCmd.Flags().StringSliceVarP(&qemuFirmware, "eve-firmware", "", nil, "firmware path")
	configAddCmd.Flags().StringVarP(&qemuConfigPath, "config-part", "", "", "path for config drive")
	configAddCmd.Flags().StringVarP(&qemuDTBPath, "dtb-part", "", "", "path for device tree drive (for arm)")
	configAddCmd.Flags().StringToStringVarP(&qemuHostFwd, "eve-hostfwd", "", defaults.DefaultQemuHostFwd, "port forward map")
	configAddCmd.Flags().StringVarP(&qemuSocketPath, "qmp", "", "", "use qmp socket with path")
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configEditCmd)
}
