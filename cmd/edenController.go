package cmd

import (
	"fmt"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
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
	eserverIP           string
	baseOSVersion       string
	getFromFileName     bool
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
		assingCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		modeType, modeURL, err := projects.GetControllerMode(controllerMode)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("Mode type: %s", modeType)
		log.Debugf("Mode url: %s", modeURL)
		var changer configChanger
		switch modeType {
		case "file":
			changer = &fileChanger{fileConfig: modeURL}
		case "adam":
			changer = &adamChanger{adamUrl: modeURL}

		default:
			log.Fatalf("Not implemented type: %s", modeType)
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

func checkIsFileOrUrl(pathToCheck string) (isFile bool, pathToRet string, err error) {
	res, err := url.Parse(pathToCheck)
	if err != nil {
		return false, "", err
	}
	switch res.Scheme {
	case "":
		return true, pathToCheck, nil
	case "file":
		return true, strings.TrimLeft(pathToCheck, "file://"), nil
	case "http":
		return false, pathToCheck, nil
	case "https":
		return false, pathToCheck, nil
	default:
		return false, "", fmt.Errorf("%s scheme not supported now", res.Scheme)
	}
}

var edgeNodeEVEImageUpdate = &cobra.Command{
	Use:   "eveimage-update <image file or url (file:// or http(s)://)>",
	Short: "update EVE image",
	Long:  `Update EVE image.`,
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
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
			eserverIP = viper.GetString("eden.eserver.ip")
			eserverPort = viper.GetInt("eden.eserver.port")
			eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		baseOSImage := args[0]
		isFile, baseOSImage, err := checkIsFileOrUrl(baseOSImage)
		if err != nil {
			log.Fatalf("checkIsFileOrUrl: %s", err)
		}
		var rootFsPath string
		if isFile {
			rootFsPath, err = utils.GetFileFollowLinks(baseOSImage)
			if err != nil {
				log.Fatalf("GetFileFollowLinks: %s", err)
			}
		} else {
			if err = os.MkdirAll(filepath.Join(eserverImageDist, "baseos", "tmp"), 0755); err != nil {
				log.Fatalf("cannot create dir for download image %s", err)
			}
			r, _ := url.Parse(baseOSImage)
			rootFsPath = filepath.Join(eserverImageDist, "baseos", "tmp", path.Base(r.Path))
			defer os.Remove(rootFsPath)
			if err := utils.DownloadFile(rootFsPath, baseOSImage); err != nil {
				log.Fatalf("DownloadFile error: %s", err)
			}
		}
		modeType, modeURL, err := projects.GetControllerMode(controllerMode)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("Mode type: %s", modeType)
		log.Debugf("Mode url: %s", modeURL)
		var changer configChanger
		switch modeType {
		case "file":
			changer = &fileChanger{fileConfig: modeURL}
		case "adam":
			changer = &adamChanger{adamUrl: modeURL}

		default:
			log.Fatalf("Not implemented type: %s", modeType)
		}

		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev error: %s", err)
		}
		var dataStore *config.DatastoreConfig
		if isFile {
			dataStore = &config.DatastoreConfig{
				Id:       defaults.DefaultDataStoreID,
				DType:    config.DsType_DsHttp,
				Fqdn:     fmt.Sprintf("http://%s:%d", eserverIP, eserverPort),
				ApiKey:   "",
				Password: "",
				Dpath:    "",
				Region:   "",
			}
		} else {
			dataStore = &config.DatastoreConfig{
				Id:       defaults.DefaultDataStoreID,
				DType:    config.DsType_DsHttp,
				Fqdn:     strings.TrimRight(baseOSImage, filepath.Base(rootFsPath)),
				ApiKey:   "",
				Password: "",
				Dpath:    "",
				Region:   "",
			}
			if strings.Contains(baseOSImage, "https://") {
				dataStore.DType = config.DsType_DsHttps
			}
		}
		if _, err := os.Lstat(rootFsPath); os.IsNotExist(err) {
			log.Fatalf("image file problem (%s): %s", args[0], err)
		}

		if getFromFileName {
			rootFSName := strings.TrimSuffix(filepath.Base(rootFsPath), filepath.Ext(rootFsPath))
			rootFSName = strings.TrimPrefix(rootFSName, "rootfs-")
			re := regexp.MustCompile(defaults.DefaultRootFSVersionPattern)
			if !re.MatchString(rootFSName) {
				log.Fatalf("Filename of rootfs %s does not match pattern %s", rootFSName, defaults.DefaultRootFSVersionPattern)
			}
			baseOSVersion = rootFSName
		}

		log.Infof("Will use rootfs version %s", baseOSVersion)
		imageFullPath := filepath.Join(eserverImageDist, filepath.Base(rootFsPath))
		os.MkdirAll(filepath.Join(eserverImageDist), os.ModePerm)
		if _, err := fileutils.CopyFile(rootFsPath, imageFullPath); err != nil {
			log.Fatalf("CopyFile problem: %s", err)
		}
		imageDSPath := fmt.Sprintf("eserver/%s", filepath.Base(rootFsPath))
		if !isFile {
			imageDSPath = filepath.Base(rootFsPath)
		}
		fi, err := os.Stat(imageFullPath)
		if err != nil {
			log.Fatalf("ImageFile (%s): %s", imageFullPath, err)
		}
		size := fi.Size()

		sha256sum := ""
		sha256sum, err = utils.SHA256SUM(imageFullPath)
		if err != nil {
			log.Fatalf("SHA256SUM (%s): %s", imageFullPath, err)
		}
		img := &config.Image{
			Uuidandversion: &config.UUIDandVersion{
				Uuid:    defaults.DefaultImageID,
				Version: "4",
			},
			Name:      imageDSPath,
			Sha256:    sha256sum,
			Iformat:   config.Format_QCOW2,
			DsId:      defaults.DefaultDataStoreID,
			SizeBytes: size,
			Siginfo: &config.SignatureInfo{
				Intercertsurl: "",
				Signercerturl: "",
				Signature:     nil,
			},
		}
		if _, err := ctrl.GetDataStore(defaults.DefaultDataStoreID); err == nil {
			if err = ctrl.RemoveDataStore(defaults.DefaultDataStoreID); err != nil {
				log.Fatalf("RemoveDataStore: %s", err)
			}
		}
		if err = ctrl.AddDataStore(dataStore); err != nil {
			log.Fatalf("AddDataStore: %s", err)
		}
		if _, err := ctrl.GetImage(defaults.DefaultImageID); err == nil {
			if err = ctrl.RemoveImage(defaults.DefaultImageID); err != nil {
				log.Fatalf("RemoveImage: %s", err)
			}
		}
		if err = ctrl.AddImage(img); err != nil {
			log.Fatalf("AddImage: %s", err)
		}

		baseOSConfig := &config.BaseOSConfig{
			Uuidandversion: &config.UUIDandVersion{
				Uuid:    defaults.DefaultBaseID,
				Version: "4",
			},
			Drives: []*config.Drive{{
				Image:        img,
				Readonly:     false,
				Drvtype:      config.DriveType_Unclassified,
				Target:       config.Target_TgtUnknown,
				Maxsizebytes: img.SizeBytes,
			}},
			Activate:      baseOSImageActivate,
			BaseOSVersion: baseOSVersion,
			BaseOSDetails: nil,
		}
		if _, err := ctrl.GetBaseOSConfig(defaults.DefaultBaseID); err == nil {
			if err = ctrl.RemoveBaseOsConfig(defaults.DefaultBaseID); err != nil {
				log.Fatalf("RemoveBaseOsConfig: %s", err)
			}
		}

		if err = ctrl.AddBaseOsConfig(baseOSConfig); err != nil {
			log.Fatalf("AddBaseOsConfig: %s", err)
		}
		dev.SetBaseOSConfig([]string{defaults.DefaultBaseID})
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev error: %s", err)
		}
	},
}

var edgeNodeEVEImageRemove = &cobra.Command{
	Use:   "eveimage-remove <image file or url (file:// or http(s)://)>",
	Short: "remove EVE image",
	Long:  `Remove EVE image.`,
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		if viperLoaded {
			eserverIP = viper.GetString("eden.eserver.ip")
			eserverPort = viper.GetInt("eden.eserver.port")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		baseOSImage := args[0]
		isFile, baseOSImage, err := checkIsFileOrUrl(baseOSImage)
		if err != nil {
			log.Fatalf("checkIsFileOrUrl: %s", err)
		}
		var rootFsPath string
		if isFile {
			rootFsPath, err = utils.GetFileFollowLinks(baseOSImage)
			if err != nil {
				log.Fatalf("GetFileFollowLinks: %s", err)
			}
		} else {
			if err = os.MkdirAll(filepath.Join(eserverImageDist, "baseos", "tmp"), 0755); err != nil {
				log.Fatalf("cannot create dir for download image %s", err)
			}
			r, _ := url.Parse(baseOSImage)
			rootFsPath = filepath.Join(eserverImageDist, "baseos", "tmp", path.Base(r.Path))
			defer os.Remove(rootFsPath)
			if err := utils.DownloadFile(rootFsPath, baseOSImage); err != nil {
				log.Fatalf("DownloadFile error: %s", err)
			}
		}
		modeType, modeURL, err := projects.GetControllerMode(controllerMode)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("Mode type: %s", modeType)
		log.Debugf("Mode url: %s", modeURL)
		var changer configChanger
		switch modeType {
		case "file":
			changer = &fileChanger{fileConfig: modeURL}
		case "adam":
			changer = &adamChanger{adamUrl: modeURL}

		default:
			log.Fatalf("Not implemented type: %s", modeType)
		}

		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev error: %s", err)
		}

		if _, err := os.Lstat(rootFsPath); os.IsNotExist(err) {
			log.Fatalf("image file problem (%s): %s", args[0], err)
		}

		if getFromFileName {
			rootFSName := strings.TrimSuffix(filepath.Base(rootFsPath), filepath.Ext(rootFsPath))
			rootFSName = strings.TrimPrefix(rootFSName, "rootfs-")
			re := regexp.MustCompile(defaults.DefaultRootFSVersionPattern)
			if !re.MatchString(rootFSName) {
				log.Fatalf("Filename of rootfs %s does not match pattern %s", rootFSName, defaults.DefaultRootFSVersionPattern)
			}
			baseOSVersion = rootFSName
		}

		log.Infof("Will use rootfs version %s", baseOSVersion)

		imageFullPath := filepath.Join(eserverImageDist, "baseos", filepath.Base(rootFsPath))
		if _, err := fileutils.CopyFile(rootFsPath, imageFullPath); err != nil {
			log.Fatalf("CopyFile problem: %s", err)
		}

		sha256sum, err := utils.SHA256SUM(imageFullPath)
		if err != nil {
			log.Fatalf("SHA256SUM (%s): %s", imageFullPath, err)
		}

		for _, baseOSConfig := range ctrl.ListBaseOSConfig() {
			if len(baseOSConfig.Drives) == 1 {
				if baseOSConfig.Drives[0].Image.Sha256 == sha256sum {
					if err = ctrl.RemoveBaseOsConfig(baseOSConfig.Uuidandversion.GetUuid()); err != nil {
						log.Fatalf("RemoveBaseOsConfig (%s): %s", baseOSConfig.Uuidandversion.GetUuid(), err)
					}
					if ind, found := utils.FindEleInSlice(dev.GetBaseOSConfigs(), baseOSConfig.Uuidandversion.GetUuid()); found {
						configs := dev.GetBaseOSConfigs()
						utils.DelEleInSlice(&configs, ind)
						dev.SetBaseOSConfig(configs)
						log.Infof("EVE image removed with id %s", baseOSConfig.Uuidandversion.GetUuid())
					}
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
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		if viperLoaded {
			eserverIP = viper.GetString("eden.eserver.ip")
			eserverPort = viper.GetInt("eden.eserver.port")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		modeType, modeURL, err := projects.GetControllerMode(controllerMode)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("Mode type: %s", modeType)
		log.Debugf("Mode url: %s", modeURL)
		var changer configChanger
		switch modeType {
		case "file":
			changer = &fileChanger{fileConfig: modeURL}
		case "adam":
			changer = &adamChanger{adamUrl: modeURL}

		default:
			log.Fatalf("Not implemented type: %s", modeType)
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
		assingCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		modeType, modeURL, err := projects.GetControllerMode(controllerMode)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("Mode type: %s", modeType)
		log.Debugf("Mode url: %s", modeURL)
		var changer configChanger
		switch modeType {
		case "file":
			changer = &fileChanger{fileConfig: modeURL}
		case "adam":
			changer = &adamChanger{adamUrl: modeURL}

		default:
			log.Fatalf("Not implemented type: %s", modeType)
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
	pf.StringVarP(&controllerMode, "mode", "m", "", "mode to use [file|proto|adam|zedcloud]://<URL> (required)")
	if err := cobra.MarkFlagRequired(pf, "mode"); err != nil {
		log.Fatal(err)
	}
	edgeNodeEVEImageUpdateFlags := edgeNodeEVEImageUpdate.Flags()
	edgeNodeEVEImageUpdateFlags.StringVarP(&baseOSVersion, "os-version", "", "", "version of ROOTFS")
	edgeNodeEVEImageUpdateFlags.BoolVarP(&getFromFileName, "from-filename", "", true, "get version from filename")
	edgeNodeEVEImageUpdateFlags.BoolVarP(&baseOSImageActivate, "activate", "", true, "activate image")
	edgeNodeUpdateFlags := edgeNodeUpdate.Flags()
	configUsage := `set of key=value items. 
Supported keys are defined in https://github.com/lf-edge/eve/blob/master/docs/CONFIG-PROPERTIES.md`
	edgeNodeUpdateFlags.StringToStringVar(&configItems, "config", make(map[string]string), configUsage)
}
