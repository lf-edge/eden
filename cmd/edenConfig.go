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
	qemuFileToSave string
	qemuCpus       int
	qemuMemory     int
	qemuFirmware   []string
	qemuConfigPath string
	eveImageFile   string
	qemuDTBPath    string
	qemuHostFwd    map[string]string
	qemuSocketPath string
	contextName    string
	contextFile    string
	contextNameDel string
	contextNameGet string
	contextNameSet string
	contextKeyGet  string
	contextAllGet  bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "work with config",
	Long:  `Work with config.`,
}
var configAddCmd = &cobra.Command{
	Use:   "add",
	Short: "generate config eden",
	Long:  `Generate config eden.`,
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
				log.Infof("current config already exists: %s", configFile)
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
				log.Infof("Test script already exists: %s", testSript)
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
		context.Current = contextName
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
			log.Infof("QEMU config already exists: %s", qemuFileToSave)
		}
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configs",
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
	Use:   "set name",
	Short: "Set current contexts",
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
		contexts := context.ListContexts()
		for _, el := range contexts {
			if el == contextNameSet {
				context.SetContext(el)
				return
			}
		}
		log.Fatalf("context not found %s", contextNameSet)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "get config context",
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
			configFile := context.GetCurrentConfig()
			data, err := ioutil.ReadFile(configFile)
			if err != nil {
				log.Fatalf("cannot read context config file %s: %s", configFile, err)
				return
			}
			fmt.Print(string(data))
		}
	},
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete config context",
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
	configDeleteCmd.Flags().StringVar(&contextNameDel, "name", "", "context name to delete (empty - current)")
	configCmd.AddCommand(configGetCmd)
	configGetCmd.Flags().StringVar(&contextNameGet, "name", "", "context name to get from (empty - current)")
	configGetCmd.Flags().StringVar(&contextKeyGet, "key", "", "will return value of key from current config context")
	configGetCmd.Flags().BoolVar(&contextAllGet, "all", false, "will return config context")
	configCmd.AddCommand(configSetCmd)
	configSetCmd.Flags().StringVar(&contextNameSet, "name", defaults.DefaultContext, "context name to set as current")
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configAddCmd)
	configAddCmd.Flags().StringVar(&contextName, "name", defaults.DefaultContext, "context name to add")
	configAddCmd.Flags().StringVar(&contextFile, "file", "", "file with config to add")
	configAddCmd.Flags().StringVarP(&qemuFileToSave, "qemu-config", "", defaults.DefaultQemuFileToSave, "file to save config")
	configAddCmd.Flags().IntVarP(&qemuCpus, "cpus", "", defaults.DefaultQemuCpus, "cpus")
	configAddCmd.Flags().IntVarP(&qemuMemory, "memory", "", defaults.DefaultQemuMemory, "memory (MB)")
	configAddCmd.Flags().StringSliceVarP(&qemuFirmware, "eve-firmware", "", nil, "firmware path")
	configAddCmd.Flags().StringVarP(&qemuConfigPath, "config-part", "", "", "path for config drive")
	configAddCmd.Flags().StringVarP(&qemuDTBPath, "dtb-part", "", "", "path for device tree drive (for arm)")
	configAddCmd.Flags().StringToStringVarP(&qemuHostFwd, "eve-hostfwd", "", defaults.DefaultQemuHostFwd, "port forward map")
	configAddCmd.Flags().StringVarP(&qemuSocketPath, "qmp", "", "", "use qmp socket with path")
}
