package cmd

import (
	"fmt"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var controllerMode string
var baseOSImage string
var baseOSImageActivate bool

func getParams(line, regEx string) (paramsMap map[string]string) {

	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(strings.TrimSpace(line))

	paramsMap = make(map[string]string)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return
}

func getControllerMode() (modeType, modeURL string, err error) {
	params := getParams(controllerMode, controllerModePattern)
	if len(params) == 0 {
		return "", "", fmt.Errorf("cannot parse mode (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	ok := false
	if modeType, ok = params["Type"]; !ok {
		return "", "", fmt.Errorf("cannot parse modeType (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	if modeURL, ok = params["URL"]; !ok {
		return "", "", fmt.Errorf("cannot parse modeURL (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	return
}

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
		if baseOSVersionFlag := cmd.Flags().Lookup("os-version"); baseOSVersionFlag != nil {
			if err := viper.BindPFlag("eve.base-tag", baseOSVersionFlag); err != nil {
				log.Fatal(err)
			}
		}
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading configFile: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		modeType, modeURL, err := getControllerMode()
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Mode type: %s", modeType)
		log.Infof("Mode url: %s", modeURL)
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

var edgeNodeEVEImageUpdate = &cobra.Command{
	Use:   "eveimage-update",
	Short: "update EVE image",
	Long:  `Update EVE image.`,
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
			eserverPort = viper.GetString("eden.eserver.port")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		rootFsPath, err := utils.GetFileFollowLinks(baseOSImage)
		if err != nil {
			log.Fatalf("GetFileFollowLinks: %s", err)
		}
		modeType, modeURL, err := getControllerMode()
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Mode type: %s", modeType)
		log.Infof("Mode url: %s", modeURL)
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

		dataStore := &config.DatastoreConfig{
			Id:       dataStoreID,
			DType:    config.DsType_DsHttp,
			Fqdn:     fmt.Sprintf("http://%s:%s", eserverIP, eserverPort),
			ApiKey:   "",
			Password: "",
			Dpath:    "",
			Region:   "",
		}
		if _, err := os.Lstat(rootFsPath); os.IsNotExist(err) {
			log.Fatalf("image file problem (%s): %s", args[0], err)
		}

		if getFromFileName {
			rootFSName := strings.TrimSuffix(filepath.Base(rootFsPath), filepath.Ext(rootFsPath))
			rootFSName = strings.TrimPrefix(rootFSName, "rootfs-")
			re := regexp.MustCompile(rootFSVersionPattern)
			if !re.MatchString(rootFSName) {
				log.Fatalf("Filename of rootfs %s does not match pattern %s", rootFSName, rootFSVersionPattern)
			}
			baseOSVersion = rootFSName
		}

		log.Infof("Will use rootfs version %s", baseOSVersion)

		imageFullPath := filepath.Join(eserverImageDist, "baseos", defaultFilename)
		if _, err := fileutils.CopyFile(rootFsPath, imageFullPath); err != nil {
			log.Fatalf("CopyFile problem: %s", err)
		}
		imageDSPath := fmt.Sprintf("baseos/%s", defaultFilename)
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
				Uuid:    imageID,
				Version: "4",
			},
			Name:      imageDSPath,
			Sha256:    sha256sum,
			Iformat:   config.Format_QCOW2,
			DsId:      dataStoreID,
			SizeBytes: size,
			Siginfo: &config.SignatureInfo{
				Intercertsurl: "",
				Signercerturl: "",
				Signature:     nil,
			},
		}
		if _, err := ctrl.GetDataStore(dataStoreID); err == nil {
			if err = ctrl.RemoveDataStore(dataStoreID); err != nil {
				log.Fatalf("RemoveDataStore: %s", err)
			}
		}
		if err = ctrl.AddDataStore(dataStore); err != nil {
			log.Fatalf("AddDataStore: %s", err)
		}
		if _, err := ctrl.GetImage(imageID); err == nil {
			if err = ctrl.RemoveImage(imageID); err != nil {
				log.Fatalf("RemoveImage: %s", err)
			}
		}
		if err = ctrl.AddImage(img); err != nil {
			log.Fatalf("AddImage: %s", err)
		}

		baseOSConfig := &config.BaseOSConfig{
			Uuidandversion: &config.UUIDandVersion{
				Uuid:    baseID,
				Version: "4",
			},
			Drives: []*config.Drive{{
				Image:        img,
				Readonly:     false,
				Preserve:     false,
				Drvtype:      config.DriveType_Unclassified,
				Target:       config.Target_TgtUnknown,
				Maxsizebytes: img.SizeBytes,
			}},
			Activate:      baseOSImageActivate,
			BaseOSVersion: baseOSVersion,
			BaseOSDetails: nil,
		}
		if _, err := ctrl.GetBaseOSConfig(baseID); err == nil {
			if err = ctrl.RemoveBaseOsConfig(baseID); err != nil {
				log.Fatalf("RemoveBaseOsConfig: %s", err)
			}
		}

		if err = ctrl.AddBaseOsConfig(baseOSConfig); err != nil {
			log.Fatalf("AddBaseOsConfig: %s", err)
		}
		dev.SetBaseOSConfig([]string{baseID})
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev error: %s", err)
		}
		log.Info("EVE update request has been sent")
	},
}

var edgeNodeEVEImageRemove = &cobra.Command{
	Use:   "eveimage-remove",
	Short: "remove EVE image",
	Long:  `Remove EVE image.`,
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
			eserverPort = viper.GetString("eden.eserver.port")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		rootFsPath, err := utils.GetFileFollowLinks(baseOSImage)
		if err != nil {
			log.Fatalf("GetFileFollowLinks: %s", err)
		}
		modeType, modeURL, err := getControllerMode()
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Mode type: %s", modeType)
		log.Infof("Mode url: %s", modeURL)
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
			re := regexp.MustCompile(rootFSVersionPattern)
			if !re.MatchString(rootFSName) {
				log.Fatalf("Filename of rootfs %s does not match pattern %s", rootFSName, rootFSVersionPattern)
			}
			baseOSVersion = rootFSName
		}

		log.Infof("Will use rootfs version %s", baseOSVersion)

		imageFullPath := filepath.Join(eserverImageDist, "baseos", defaultFilename)
		if _, err := fileutils.CopyFile(rootFsPath, imageFullPath); err != nil {
			log.Fatalf("CopyFile problem: %s", err)
		}

		sha256sum := ""
		sha256sum, err = utils.SHA256SUM(imageFullPath)
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
		log.Info("EVE update request has been sent")
	},
}

func controllerInit() {
	configPath, err := utils.DefaultConfigPath()
	if err != nil {
		log.Fatal(err)
	}
	controllerCmd.AddCommand(edgeNode)
	edgeNode.AddCommand(edgeNodeReboot)
	edgeNode.AddCommand(edgeNodeEVEImageUpdate)
	edgeNode.AddCommand(edgeNodeEVEImageRemove)
	pf := controllerCmd.PersistentFlags()
	pf.StringVarP(&controllerMode, "mode", "m", "", "mode to use [file|proto|adam|zedcloud]://<URL>")
	pf.StringVar(&configFile, "config", configPath, "path to config file")
	if err = cobra.MarkFlagRequired(pf, "mode"); err != nil {
		log.Fatal(err)
	}
	edgeNodeEVEImageUpdateFlags := edgeNodeEVEImageUpdate.Flags()
	edgeNodeEVEImageUpdateFlags.StringVarP(&baseOSVersion, "os-version", "", fmt.Sprintf("%s-%s-%s", utils.DefaultBaseOSVersion, eveHV, eveArch), "version of ROOTFS")
	edgeNodeEVEImageUpdateFlags.BoolVarP(&getFromFileName, "from-filename", "", true, "get version from filename")
	edgeNodeEVEImageUpdateFlags.BoolVarP(&baseOSImageActivate, "activate", "", true, "activate image")
	edgeNodeEVEImageUpdateFlags.StringVarP(&baseOSImage, "image", "", "", "image file")
	if err = cobra.MarkFlagRequired(edgeNodeEVEImageUpdateFlags, "image"); err != nil {
		log.Fatal(err)
	}
	edgeNodeEVEImageRemoveFlags := edgeNodeEVEImageRemove.Flags()
	edgeNodeEVEImageRemoveFlags.StringVarP(&baseOSImage, "image", "", "", "image file for compare hash")
	if err = cobra.MarkFlagRequired(edgeNodeEVEImageRemoveFlags, "image"); err != nil {
		log.Fatal(err)
	}
}
