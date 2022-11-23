package openevec

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/protojson"
)

func EdgeNodeReboot(controllerMode string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev error: %s", err)
	}
	dev.Reboot()
	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		log.Fatalf("setControllerAndDev error: %s", err)
	}
	log.Info("Reboot request has been sent")

	return nil
}

func EdgeNodeShutdown(controllerMode string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev error: %s", err)
	}
	dev.Shutdown()
	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev error: %s", err)
	}
	log.Info("Shutdown request has been sent")

	return nil
}

func EdgeNodeEVEImageUpdate(baseOSImage, baseOSVersion, registry, controllerMode string,
	baseOSImageActivate, baseOSVDrive bool) error {

	var opts []expect.ExpectationOption
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		log.Fatal(err)
	}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		log.Fatalf("getControllerAndDev error: %s", err)
	}
	registryToUse := registry
	switch registry {
	case "local":
		registryToUse = fmt.Sprintf("%s:%d", viper.GetString("registry.ip"), viper.GetInt("registry.port"))
	case "remote":
		registryToUse = ""
	}
	opts = append(opts, expect.WithRegistry(registryToUse))
	expectation := expect.AppExpectationFromURL(ctrl, dev, baseOSImage, "", opts...)
	if baseOSVDrive {
		baseOSImageConfig := expectation.BaseOSConfig(baseOSVersion)
		dev.SetBaseOSConfig(append(dev.GetBaseOSConfigs(), baseOSImageConfig.Uuidandversion.Uuid))
	}

	baseOS := expectation.BaseOS(baseOSVersion)
	dev.SetBaseOSActivate(baseOSImageActivate)
	dev.SetBaseOSContentTree(baseOS.ContentTreeUuid)
	dev.SetBaseOSRetryCounter(0)
	dev.SetBaseOSVersion(baseOS.BaseOsVersion)

	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %s", err)
	}
	return nil
}

func EdgeNodeEVEImageUpdateRetry(controllerMode string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev error: %s", err)
	}
	dev.SetBaseOSRetryCounter(dev.GetBaseOSRetryCounter() + 1)

	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %s", err)
	}

	return nil
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

func EdgeNodeEVEImageRemove(controllerMode, baseOSVersion, baseOSImage, edenDist string) error {
	isFile, baseOSImage, err := checkIsFileOrURL(baseOSImage)
	if err != nil {
		return fmt.Errorf("checkIsFileOrURL: %s", err)
	}
	var rootFsPath string
	if isFile {
		rootFsPath, err = utils.GetFileFollowLinks(baseOSImage)
		if err != nil {
			return fmt.Errorf("GetFileFollowLinks: %s", err)
		}
	} else {
		r, _ := url.Parse(baseOSImage)
		switch r.Scheme {
		case "http", "https":
			if err = os.MkdirAll(filepath.Join(edenDist, "tmp"), 0755); err != nil {
				return fmt.Errorf("cannot create dir for download image %s", err)
			}
			rootFsPath = filepath.Join(edenDist, "tmp", path.Base(r.Path))
			defer os.Remove(rootFsPath)
			if err := utils.DownloadFile(rootFsPath, baseOSImage); err != nil {
				return fmt.Errorf("DownloadFile error: %s", err)
			}
		case "oci", "docker":
			bits := strings.Split(r.Path, ":")
			if len(bits) == 2 {
				rootFsPath = "rootfs-" + bits[1] + ".dummy"
			} else {
				rootFsPath = "latest.dummy"
			}
		default:
			return fmt.Errorf("unknown URI scheme: %s", r.Scheme)
		}
	}
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}

	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev error: %s", err)
	}

	if baseOSVersion == "" {
		correctionFileName := fmt.Sprintf("%s.ver", rootFsPath)
		if rootFSFromCorrectionFile, err := ioutil.ReadFile(correctionFileName); err == nil {
			baseOSVersion = string(rootFSFromCorrectionFile)
		} else {
			rootFSName := utils.FileNameWithoutExtension(rootFsPath)
			rootFSName = strings.TrimPrefix(rootFSName, "rootfs-")
			re := regexp.MustCompile(defaults.DefaultRootFSVersionPattern)
			if !re.MatchString(rootFSName) {
				return fmt.Errorf("Filename of rootfs %s does not match pattern %s", rootFSName, defaults.DefaultRootFSVersionPattern)
			}
			baseOSVersion = rootFSName
		}
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
		return fmt.Errorf("setControllerAndDev error: %s", err)
	}
	return nil
}

func EdgeNodeUpdate(controllerMode string, deviceItems, configItems map[string]string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}

	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev error: %s", err)
	}
	for key, val := range configItems {
		dev.SetConfigItem(key, val)
	}
	for key, val := range deviceItems {
		if err := dev.SetDeviceItem(key, val); err != nil {
			return fmt.Errorf("SetDeviceItem: %s", err)
		}
	}

	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev error: %s", err)
	}

	return nil
}

func EdgeNodeGetConfig(controllerMode, fileWithConfig string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}

	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev error: %s", err)
	}

	res, err := ctrl.GetConfigBytes(dev, true)
	if err != nil {
		return fmt.Errorf("GetConfigBytes error: %s", err)
	}
	if fileWithConfig != "" {
		if err = ioutil.WriteFile(fileWithConfig, res, 0755); err != nil {
			log.Fatalf("WriteFile: %s", err)
		}
	} else {
		fmt.Println(string(res))
	}
	return nil
}

func EdgeNodeSetConfig(fileWithConfig string) error {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return fmt.Errorf("CloudPrepare: %s", err)
	}
	devFirst, err := ctrl.GetDeviceCurrent()
	if err != nil {
		return fmt.Errorf("GetDeviceCurrent error: %s", err)
	}
	devUUID := devFirst.GetID()
	var newConfig []byte
	if fileWithConfig != "" {
		newConfig, err = ioutil.ReadFile(fileWithConfig)
		if err != nil {
			return fmt.Errorf("File reading error: %s", err)
		}
	} else if utils.IsInputFromPipe() {
		newConfig, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("Stdin reading error: %s", err)
		}
	} else {
		return fmt.Errorf("Please run command with --file or use it with pipe")
	}
	// we should validate config with unmarshal
	var dConfig config.EdgeDevConfig
	if err := protojson.Unmarshal(newConfig, &dConfig); err != nil {
		return fmt.Errorf("Cannot unmarshal config: %s", err)
	}
	// Adam expects json type
	cfg, err := json.Marshal(&dConfig)
	if err != nil {
		return fmt.Errorf("Cannot marshal config: %s", err)
	}
	if err = ctrl.ConfigSet(devUUID, cfg); err != nil {
		return fmt.Errorf("ConfigSet: %s", err)
	}
	log.Info("Config loaded")
	return nil
}

func EdgeNodeGetOptions(controllerMode, fileWithConfig string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev error: %s", err)
	}
	res, err := ctrl.GetDeviceOptions(dev.GetID())
	if err != nil {
		return fmt.Errorf("GetDeviceOptions error: %s", err)
	}
	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		return fmt.Errorf("Cannot marshal: %s", err)
	}
	if fileWithConfig != "" {
		if err = ioutil.WriteFile(fileWithConfig, data, 0755); err != nil {
			return fmt.Errorf("WriteFile: %s", err)
		}
	} else {
		fmt.Println(string(data))
	}

	return nil
}

func EdgeNodeSetOptions(controllerMode, fileWithConfig string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev error: %s", err)
	}
	var newOptionsBytes []byte
	if fileWithConfig != "" {
		newOptionsBytes, err = ioutil.ReadFile(fileWithConfig)
		if err != nil {
			return fmt.Errorf("File reading error: %s", err)
		}
	} else if utils.IsInputFromPipe() {
		newOptionsBytes, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("Stdin reading error: %s", err)
		}
	} else {
		return fmt.Errorf("Please run command with --file or use it with pipe")
	}
	var devOptions types.DeviceOptions
	if err := json.Unmarshal(newOptionsBytes, &devOptions); err != nil {
		return fmt.Errorf("Cannot unmarshal: %s", err)
	}
	if err := ctrl.SetDeviceOptions(dev.GetID(), &devOptions); err != nil {
		return fmt.Errorf("Cannot set device options: %s", err)
	}
	log.Info("Options loaded")

	return nil
}

func ControllerGetOptions(fileWithConfig string) error {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return fmt.Errorf("CloudPrepare error: %s", err)
	}
	res, err := ctrl.GetGlobalOptions()
	if err != nil {
		return fmt.Errorf("GetGlobalOptions error: %s", err)
	}
	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		return fmt.Errorf("Cannot marshal: %s", err)
	}
	if fileWithConfig != "" {
		if err = ioutil.WriteFile(fileWithConfig, data, 0755); err != nil {
			return fmt.Errorf("WriteFile: %s", err)
		}
	} else {
		fmt.Println(string(data))
	}
	return nil
}

func ControllerSetOptions(fileWithConfig string) error {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return fmt.Errorf("CloudPrepare error: %s", err)
	}
	var newOptionsBytes []byte
	if fileWithConfig != "" {
		newOptionsBytes, err = ioutil.ReadFile(fileWithConfig)
		if err != nil {
			return fmt.Errorf("File reading error: %s", err)
		}
	} else if utils.IsInputFromPipe() {
		newOptionsBytes, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("Stdin reading error: %s", err)
		}
	} else {
		log.Fatal("Please run command with --file or use it with pipe")
	}
	var globalOptions types.GlobalOptions
	if err := json.Unmarshal(newOptionsBytes, &globalOptions); err != nil {
		return fmt.Errorf("Cannot unmarshal: %s", err)
	}
	if err := ctrl.SetGlobalOptions(&globalOptions); err != nil {
		return fmt.Errorf("Cannot set global options: %s", err)
	}
	log.Info("Options loaded")

	return nil
}
