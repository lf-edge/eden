package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	controllerMode      string
	baseOSImageActivate bool
	configItems         map[string]string
	baseOSVersion       string
	getFromFileName     bool
	edenDist            string
)

var controllerCmd = &cobra.Command{
	Use:   "controller",
	Short: "interact with controller",
	Long:  `Interact with controller.`,
}

var edgeNode = &cobra.Command{
	Use:   "edge-node",
	Short: "manage EVE instance",
	Long:  `Manage EVE instance.`,
}

var edgeNodeReboot = &cobra.Command{
	Use:   "reboot",
	Short: "reboot EVE instance",
	Long:  `reboot EVE instance.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		changer, err := changerByControllerMode(controllerMode)
		if err != nil {
			log.Fatal(err)
		}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev error: %s", err)
		}
		rebootCounter, _ := dev.GetRebootCounter()
		dev.SetRebootCounter(rebootCounter+1, true)
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev error: %s", err)
		}
		log.Info("Reboot request has been sent")
	},
}

func checkIsFileOrURL(pathToCheck string) (isFile bool, pathToRet string, err error) {
	res, err := url.Parse(pathToCheck)
	if err != nil {
		return false, "", err
	}
	switch res.Scheme {
	case "":
		return true, pathToCheck, nil
	case "file":
		return true, strings.TrimPrefix(pathToCheck, "file://"), nil
	case "http":
		return false, pathToCheck, nil
	case "https":
		return false, pathToCheck, nil
	case "oci":
		return false, pathToCheck, nil
	case "docker":
		return false, pathToCheck, nil
	default:
		return false, "", fmt.Errorf("%s scheme not supported now", res.Scheme)
	}
}

var edgeNodeEVEImageUpdate = &cobra.Command{
	Use:   "eveimage-update <image file or url (oci:// or file:// or http(s)://)>",
	Short: "update EVE image",
	Long:  `Update EVE image.`,
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		if baseOSVersionFlag := cmd.Flags().Lookup("os-version"); baseOSVersionFlag != nil {
			if err := viper.BindPFlag("eve.base-tag", baseOSVersionFlag); err != nil {
				log.Fatal(err)
			}
		}
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		if viperLoaded {
			eserverPort = viper.GetInt("eden.eserver.port")
			edenDist = viper.GetString("eden.dist")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		baseOSImage := args[0]
		changer, err := changerByControllerMode(controllerMode)
		if err != nil {
			log.Fatal(err)
		}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev error: %s", err)
		}
		expectation := expect.AppExpectationFromURL(ctrl, dev, baseOSImage, "")
		if len(qemuPorts) == 0 {
			qemuPorts = nil
		}
		baseOSImageConfig := expectation.BaseOSImage()
		dev.SetBaseOSConfig(append(dev.GetBaseOSConfigs(), baseOSImageConfig.Uuidandversion.Uuid))
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev: %s", err)
		}
	},
}

var edgeNodeEVEImageRemove = &cobra.Command{
	Use:   "eveimage-remove <image file or url (oci:// or file:// or http(s)://)>",
	Short: "remove EVE image",
	Long:  `Remove EVE image.`,
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		if viperLoaded {
			eserverPort = viper.GetInt("eden.eserver.port")
			edenDist = viper.GetString("eden.dist")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		baseOSImage := args[0]
		isFile, baseOSImage, err := checkIsFileOrURL(baseOSImage)
		if err != nil {
			log.Fatalf("checkIsFileOrURL: %s", err)
		}
		var rootFsPath string
		if isFile {
			rootFsPath, err = utils.GetFileFollowLinks(baseOSImage)
			if err != nil {
				log.Fatalf("GetFileFollowLinks: %s", err)
			}
		} else {
			r, _ := url.Parse(baseOSImage)
			switch r.Scheme {
			case "http", "https":
				if err = os.MkdirAll(filepath.Join(edenDist, "tmp"), 0755); err != nil {
					log.Fatalf("cannot create dir for download image %s", err)
				}
				rootFsPath = filepath.Join(edenDist, "tmp", path.Base(r.Path))
				defer os.Remove(rootFsPath)
				if err := utils.DownloadFile(rootFsPath, baseOSImage); err != nil {
					log.Fatalf("DownloadFile error: %s", err)
				}
			case "oci", "docker":
				bits := strings.Split(r.Path, ":")
				if len(bits) == 2 {
					rootFsPath = "rootfs-" + bits[1] + ".dummy"
				} else {
					rootFsPath = "latest.dummy"
				}
			default:
				log.Fatalf("unknown URI scheme: %s", r.Scheme)
			}
		}
		changer, err := changerByControllerMode(controllerMode)
		if err != nil {
			log.Fatal(err)
		}

		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev error: %s", err)
		}

		if getFromFileName {
			rootFSName := utils.FileNameWithoutExtension(rootFsPath)
			rootFSName = strings.TrimPrefix(rootFSName, "rootfs-")
			re := regexp.MustCompile(defaults.DefaultRootFSVersionPattern)
			if !re.MatchString(rootFSName) {
				log.Fatalf("Filename of rootfs %s does not match pattern %s", rootFSName, defaults.DefaultRootFSVersionPattern)
			}
			baseOSVersion = rootFSName
		}

		log.Infof("Will use rootfs version %s", baseOSVersion)

		toActivate := true
		for _, baseOSConfig := range ctrl.ListBaseOSConfig() {
			if baseOSConfig.BaseOSVersion == baseOSVersion {
				if ind, found := utils.FindEleInSlice(dev.GetBaseOSConfigs(), baseOSConfig.Uuidandversion.GetUuid()); found {
					configs := dev.GetBaseOSConfigs()
					utils.DelEleInSlice(&configs, ind)
					dev.SetBaseOSConfig(configs)
					log.Infof("EVE base OS image removed with id %s", baseOSConfig.Uuidandversion.GetUuid())
				}
			} else {
				if toActivate {
					toActivate = false
					baseOSConfig.Activate = true //activate another one if exists
				}
			}
		}
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev error: %s", err)
		}
	},
}

var edgeNodeUpdate = &cobra.Command{
	Use:   "update --config key=value",
	Short: "update EVE config",
	Long:  `Update EVE config.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		if viperLoaded {
			eserverPort = viper.GetInt("eden.eserver.port")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		changer, err := changerByControllerMode(controllerMode)
		if err != nil {
			log.Fatal(err)
		}

		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev error: %s", err)
		}
		for key, val := range configItems {
			dev.SetConfigItem(key, val)
		}

		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev error: %s", err)
		}
	},
}

var edgeNodeGetConfig = &cobra.Command{
	Use:   "get-config",
	Short: "fetch EVE config",
	Long:  `Fetch EVE config.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		changer, err := changerByControllerMode(controllerMode)
		if err != nil {
			log.Fatal(err)
		}

		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev error: %s", err)
		}

		res, err := ctrl.GetConfigBytes(dev, true)
		if err != nil {
			log.Fatalf("GetConfigBytes error: %s", err)
		}

		fmt.Printf("%s\n", string(res))
	},
}

func controllerInit() {
	controllerCmd.AddCommand(edgeNode)
	edgeNode.AddCommand(edgeNodeReboot)
	edgeNode.AddCommand(edgeNodeEVEImageUpdate)
	edgeNode.AddCommand(edgeNodeEVEImageRemove)
	edgeNode.AddCommand(edgeNodeUpdate)
	edgeNode.AddCommand(edgeNodeGetConfig)
	pf := controllerCmd.PersistentFlags()
	pf.StringVarP(&controllerMode, "mode", "m", "", "mode to use [file|proto|adam|zedcloud]://<URL> (default is adam)")
	edgeNodeEVEImageUpdateFlags := edgeNodeEVEImageUpdate.Flags()
	edgeNodeEVEImageUpdateFlags.StringVarP(&baseOSVersion, "os-version", "", "", "version of ROOTFS")
	edgeNodeEVEImageUpdateFlags.BoolVarP(&getFromFileName, "from-filename", "", true, "get version from filename")
	edgeNodeEVEImageUpdateFlags.BoolVarP(&baseOSImageActivate, "activate", "", true, "activate image")
	edgeNodeUpdateFlags := edgeNodeUpdate.Flags()
	configUsage := `set of key=value items.
Supported keys are defined in https://github.com/lf-edge/eve/blob/master/docs/CONFIG-PROPERTIES.md`
	edgeNodeUpdateFlags.StringToStringVar(&configItems, "config", make(map[string]string), configUsage)
}
