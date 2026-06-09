package openevec

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func (openEVEC *OpenEVEC) EdgeNodeReboot(controllerMode string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}
	dev.Reboot()
	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev error: %w", err)
	}
	log.Info("Reboot request has been sent")

	return nil
}

func (openEVEC *OpenEVEC) EdgeNodeShutdown(controllerMode string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}
	dev.Shutdown()
	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev error: %w", err)
	}
	log.Info("Shutdown request has been sent")

	return nil
}

func (openEVEC *OpenEVEC) EdgeNodeEVEImageUpdate(baseOSImage, baseOSVersion, registry, controllerMode string,
	baseOSImageActivate, baseOSVDrive bool) error {

	var opts []expect.ExpectationOption
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}
	registryToUse := registry
	switch registry {
	case "local":
		registryToUse = fmt.Sprintf("%s:%d", openEVEC.cfg.Registry.IP, openEVEC.cfg.Registry.Port)
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
		return fmt.Errorf("setControllerAndDev: %w", err)
	}
	return nil
}

func (openEVEC *OpenEVEC) EdgeNodeEVEImageUpdateRetry(controllerMode string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}
	dev.SetBaseOSRetryCounter(dev.GetBaseOSRetryCounter() + 1)

	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %w", err)
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

func (openEVEC *OpenEVEC) EdgeNodeEVEImageRemove(controllerMode, baseOSVersion, baseOSImage string) error {
	cfg := *openEVEC.cfg
	isFile, baseOSImage, err := checkIsFileOrURL(baseOSImage)
	if err != nil {
		return fmt.Errorf("checkIsFileOrURL: %w", err)
	}
	var rootFsPath string
	if isFile {
		rootFsPath, err = utils.GetFileFollowLinks(baseOSImage)
		if err != nil {
			return fmt.Errorf("GetFileFollowLinks: %w", err)
		}
	} else {
		r, _ := url.Parse(baseOSImage)
		switch r.Scheme {
		case "http", "https":
			if err = os.MkdirAll(filepath.Join(cfg.Eden.Dist, "tmp"), 0755); err != nil {
				return fmt.Errorf("cannot create dir for download image %w", err)
			}
			rootFsPath = filepath.Join(cfg.Eden.Dist, "tmp", path.Base(r.Path))
			defer os.Remove(rootFsPath)
			if err := utils.DownloadFile(rootFsPath, baseOSImage); err != nil {
				return fmt.Errorf("DownloadFile error: %w", err)
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

	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}

	if baseOSVersion == "" {
		correctionFileName := fmt.Sprintf("%s.ver", rootFsPath)
		if rootFSFromCorrectionFile, err := os.ReadFile(correctionFileName); err == nil {
			baseOSVersion = string(rootFSFromCorrectionFile)
		} else {
			rootFSName := utils.FileNameWithoutExtension(rootFsPath)
			rootFSName = strings.TrimPrefix(rootFSName, "rootfs-")
			re := regexp.MustCompile(defaults.DefaultRootFSVersionPattern)
			if !re.MatchString(rootFSName) {
				return fmt.Errorf("filename of rootfs %s does not match pattern %s", rootFSName, defaults.DefaultRootFSVersionPattern)
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
				baseOSConfig.Activate = true // activate another one if exists
			}
		}
	}

	// Symmetric counterpart to EdgeNodeEVEImageUpdate: when the removed
	// version matches the modern single-block baseos fields, clear them
	// and drop the corresponding ContentTree reference. Without this,
	// EdgeDevConfig still ships baseos:{activate:true, version:<X>,
	// content_tree_uuid:<Y>} and contentInfo:[<Y>] to the device.
	if dev.GetBaseOSVersion() == baseOSVersion {
		ctUUID := dev.GetBaseOSContentTree()
		dev.SetBaseOSActivate(false)
		dev.SetBaseOSContentTree("")
		dev.SetBaseOSVersion("")
		dev.SetBaseOSRetryCounter(0)
		if ctUUID != "" {
			if err := ctrl.RemoveContentTree(ctUUID); err != nil {
				log.Debugf("RemoveContentTree(%s): %v (orphan tolerated)", ctUUID, err)
			}
		}
		log.Infof("EVE base OS image removed (modern baseos block cleared for %s)", baseOSVersion)
	}

	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev error: %w", err)
	}
	return nil
}

func (openEVEC *OpenEVEC) EdgeNodeUpdate(controllerMode string, deviceItems, configItems map[string]string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}

	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}
	for key, val := range configItems {
		dev.SetConfigItem(key, val)
	}
	for key, val := range deviceItems {
		if err := dev.SetDeviceItem(key, val); err != nil {
			return fmt.Errorf("SetDeviceItem: %w", err)
		}
	}

	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev error: %w", err)
	}

	return nil
}

func (openEVEC *OpenEVEC) EdgeNodeGetConfig(controllerMode, fileWithConfig string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}

	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}

	res, err := ctrl.GetConfigBytes(dev, true)
	if err != nil {
		return fmt.Errorf("GetConfigBytes error: %w", err)
	}
	if fileWithConfig != "" {
		if err = os.WriteFile(fileWithConfig, res, 0755); err != nil {
			return fmt.Errorf("writeFile: %w", err)
		}
	} else {
		fmt.Println(string(res))
	}
	return nil
}

func (openEVEC *OpenEVEC) EdgeNodeSetConfig(fileWithConfig string) error {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return fmt.Errorf("CloudPrepare: %w", err)
	}
	vars, err := InitVarsFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("InitVarsFromConfig error: %w", err)
	}
	ctrl.SetVars(vars)
	devFirst, err := ctrl.GetDeviceCurrent()
	if err != nil {
		return fmt.Errorf("GetDeviceCurrent error: %w", err)
	}
	devUUID := devFirst.GetID()
	var newConfig []byte
	if fileWithConfig != "" {
		newConfig, err = os.ReadFile(fileWithConfig)
		if err != nil {
			return fmt.Errorf("file reading error: %w", err)
		}
	} else if utils.IsInputFromPipe() {
		newConfig, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("stdin reading error: %w", err)
		}
	} else {
		return fmt.Errorf("please run command with --file or use it with pipe")
	}
	// we should validate config with unmarshal
	var dConfig config.EdgeDevConfig
	if err := protojson.Unmarshal(newConfig, &dConfig); err != nil {
		return fmt.Errorf("cannot unmarshal config: %w", err)
	}
	// Adam expects json type
	cfg, err := proto.Marshal(&dConfig)
	if err != nil {
		return fmt.Errorf("cannot marshal config: %w", err)
	}
	if err = ctrl.ConfigSet(devUUID, cfg); err != nil {
		return fmt.Errorf("ConfigSet: %w", err)
	}
	log.Info("Config loaded")
	return nil
}

func (openEVEC *OpenEVEC) EdgeNodeGetOptions(controllerMode, fileWithConfig string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}
	res, err := ctrl.GetDeviceOptions(dev.GetID())
	if err != nil {
		return fmt.Errorf("GetDeviceOptions error: %w", err)
	}
	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		return fmt.Errorf("cannot marshal: %w", err)
	}
	if fileWithConfig != "" {
		if err = os.WriteFile(fileWithConfig, data, 0755); err != nil {
			return fmt.Errorf("WriteFile: %w", err)
		}
	} else {
		fmt.Println(string(data))
	}

	return nil
}

func (openEVEC *OpenEVEC) EdgeNodeSetOptions(controllerMode, fileWithConfig string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}
	var newOptionsBytes []byte
	if fileWithConfig != "" {
		newOptionsBytes, err = os.ReadFile(fileWithConfig)
		if err != nil {
			return fmt.Errorf("file reading error: %w", err)
		}
	} else if utils.IsInputFromPipe() {
		newOptionsBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("stdin reading error: %w", err)
		}
	} else {
		return fmt.Errorf("please run command with --file or use it with pipe")
	}
	var devOptions types.DeviceOptions
	if err := json.Unmarshal(newOptionsBytes, &devOptions); err != nil {
		return fmt.Errorf("cannot unmarshal: %w", err)
	}
	if err := ctrl.SetDeviceOptions(dev.GetID(), &devOptions); err != nil {
		return fmt.Errorf("cannot set device options: %w", err)
	}
	log.Info("Options loaded")

	return nil
}

func (openEVEC *OpenEVEC) ControllerGetOptions(fileWithConfig string) error {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return fmt.Errorf("CloudPrepare error: %w", err)
	}
	vars, err := InitVarsFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("InitVarsFromConfig error: %w", err)
	}
	ctrl.SetVars(vars)
	res, err := ctrl.GetGlobalOptions()
	if err != nil {
		return fmt.Errorf("GetGlobalOptions error: %w", err)
	}
	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		return fmt.Errorf("cannot marshal: %w", err)
	}
	if fileWithConfig != "" {
		if err = os.WriteFile(fileWithConfig, data, 0755); err != nil {
			return fmt.Errorf("WriteFile: %w", err)
		}
	} else {
		fmt.Println(string(data))
	}
	return nil
}

func (openEVEC *OpenEVEC) ControllerSetOptions(fileWithConfig string) error {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return fmt.Errorf("CloudPrepare error: %w", err)
	}
	vars, err := InitVarsFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("InitVarsFromConfig error: %w", err)
	}
	ctrl.SetVars(vars)
	var newOptionsBytes []byte
	if fileWithConfig != "" {
		newOptionsBytes, err = os.ReadFile(fileWithConfig)
		if err != nil {
			return fmt.Errorf("file reading error: %w", err)
		}
	} else if utils.IsInputFromPipe() {
		newOptionsBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("stdin reading error: %w", err)
		}
	} else {
		return fmt.Errorf("please run command with --file or use it with pipe")
	}
	var globalOptions types.GlobalOptions
	if err := json.Unmarshal(newOptionsBytes, &globalOptions); err != nil {
		return fmt.Errorf("cannot unmarshal: %w", err)
	}
	if err := ctrl.SetGlobalOptions(&globalOptions); err != nil {
		return fmt.Errorf("cannot set global options: %w", err)
	}
	log.Info("Options loaded")

	return nil
}

// EdgeNodeClusterSet pushes a stub EdgeNodeCluster config to the device
// with the specified cluster type (k3sbase | replicated-storage | ha).
// Workaround for lf-edge/eve#6018; see device.DefaultStubCluster for the
// rationale.
func (openEVEC *OpenEVEC) EdgeNodeClusterSet(controllerMode string, clusterType string) error {
	var ct config.ClusterType
	switch clusterType {
	case "k3sbase":
		ct = config.ClusterType_CLUSTER_TYPE_K3S_BASE
	case "replicated-storage":
		ct = config.ClusterType_CLUSTER_TYPE_REPLICATED_STORAGE
	case "ha":
		ct = config.ClusterType_CLUSTER_TYPE_HA
	case "none":
		ct = config.ClusterType_CLUSTER_TYPE_UNSPECIFIED
	default:
		return fmt.Errorf("unsupported cluster type %q (want one of: k3sbase, replicated-storage, ha, none)", clusterType)
	}

	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}

	cluster := dev.GetCluster()
	if cluster == nil {
		// No prior cluster set — start from the loopback stub.
		cluster = device.DefaultStubCluster()
	}
	cluster.ClusterType = ct
	dev.SetCluster(cluster)

	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev error: %w", err)
	}
	log.Infof("Pushed EdgeNodeCluster (type=%s) to the device", clusterType)
	return nil
}

// EdgeNodeContentTreeAdd registers a standalone ContentTree (no associated
// Volume or AppInstance) in the device config. The URL accepts the same
// forms as 'eden volume create' / 'eden pod deploy': docker://, file://,
// http(s)://. Pillar's volumemgr downloads ContentTrees eagerly, so the
// device will fetch the blobs and reach LOADED state. Because blob lookup
// is by SHA256, a subsequent app deploy against the same image (or any
// other image sharing the SHA) reuses the existing blobs without
// re-downloading. This lets tests pre-stage a content tree, exercise a
// migration (e.g. cross-HV upgrade), and then deploy an app from it
// without involving the PVC machinery.
func (openEVEC *OpenEVEC) EdgeNodeContentTreeAdd(appLink, registry, contentTreeName, datastoreOverride string, sftpLoad, directLoad bool) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig: %w", err)
	}
	var opts []expect.ExpectationOption
	opts = append(opts, expect.WithSFTPLoad(sftpLoad))
	if !sftpLoad {
		opts = append(opts, expect.WithHTTPDirectLoad(directLoad))
	}
	opts = append(opts, expect.WithDatastoreOverride(datastoreOverride))
	registryToUse := registry
	switch registry {
	case "local":
		registryToUse = fmt.Sprintf("%s:%d", openEVEC.cfg.Registry.IP, openEVEC.cfg.Registry.Port)
	case "remote":
		registryToUse = ""
	}
	opts = append(opts, expect.WithRegistry(registryToUse))
	expectation := expect.AppExpectationFromURL(ctrl, dev, appLink, contentTreeName, opts...)
	contentTree := expectation.ContentTree(contentTreeName)
	log.Infof("create content tree %s with %s request sent", contentTree.DisplayName, appLink)
	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %w", err)
	}
	return nil
}

// EdgeNodeClusterClear removes any EdgeNodeCluster config from the device.
// After clearing, pillar's parseEdgeNodeClusterConfig will publish an
// empty (Initialized=true, Valid=false) ENCC — which means volumemgr will
// fall back to waiting for longhorn (the behavior workaround'd by the
// default stub). Useful for tests that explicitly want pillar's "no
// cluster config" branch.
func (openEVEC *OpenEVEC) EdgeNodeClusterClear(controllerMode string) error {
	changer, err := changerByControllerMode(controllerMode)
	if err != nil {
		return err
	}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		return fmt.Errorf("getControllerAndDevFromConfig error: %w", err)
	}
	dev.SetCluster(nil)
	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev error: %w", err)
	}
	log.Info("Cleared EdgeNodeCluster on the device")
	return nil
}
