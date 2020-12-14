package expect

import (
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

//parse file or url name and returns Base OS Version
func (exp *AppExpectation) getBaseOSVersion() string {
	if exp.appType == dockerApp {
		return exp.appVersion
	}
	rootFSName := path.Base(exp.appURL)
	rootFSName = strings.TrimSuffix(rootFSName, filepath.Ext(rootFSName))
	rootFSName = strings.TrimPrefix(rootFSName, "rootfs-")
	if re := regexp.MustCompile(defaults.DefaultRootFSVersionPattern); !re.MatchString(rootFSName) {
		log.Fatalf("Filename of rootfs %s does not match pattern %s", rootFSName, defaults.DefaultRootFSVersionPattern)
	}
	return rootFSName
}

//checkBaseOSConfig checks if provided BaseOSConfig match expectation
func (exp *AppExpectation) checkBaseOSConfig(baseOS *config.BaseOSConfig) bool {
	if baseOS == nil {
		return false
	}
	if baseOS.BaseOSVersion == exp.getBaseOSVersion() {
		return true
	}
	return false
}

//createBaseOSConfig creates BaseOSConfig with provided img
func (exp *AppExpectation) createBaseOSConfig(img *config.Image) (*config.BaseOSConfig, error) {
	baseOSConfig := &config.BaseOSConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    img.Uuidandversion.Uuid,
			Version: "4",
		},
		Drives: []*config.Drive{{
			Image:        img,
			Readonly:     false,
			Drvtype:      config.DriveType_Unclassified,
			Target:       config.Target_TgtUnknown,
			Maxsizebytes: img.SizeBytes,
		}},
		Activate:      true,
		BaseOSVersion: exp.getBaseOSVersion(),
	}
	return baseOSConfig, nil
}

//BaseOSImage expectation gets or creates Image definition,
//gets BaseOSConfig and returns it or creates BaseOSConfig, adds it into internal controller and returns it
func (exp *AppExpectation) BaseOSImage() (baseOSConfig *config.BaseOSConfig) {
	var err error
	if exp.appType == fileApp {
		if exp.appURL, err = utils.GetFileFollowLinks(exp.appURL); err != nil {
			log.Fatalf("GetFileFollowLinks: %s", err)
		}
	}
	image := exp.Image()
	for _, baseOS := range exp.ctrl.ListBaseOSConfig() {
		if exp.checkBaseOSConfig(baseOS) {
			baseOSConfig = baseOS
			break
		}
	}
	if baseOSConfig == nil { //if baseOSConfig not exists, create it
		for _, baseOS := range exp.ctrl.ListBaseOSConfig() {
			baseOS.Activate = false
		}
		if baseOSConfig, err = exp.createBaseOSConfig(image); err != nil {
			log.Fatalf("cannot create baseOS: %s", err)
		}
		if err = exp.ctrl.AddBaseOsConfig(baseOSConfig); err != nil {
			log.Fatalf("AddBaseOsConfig: %s", err)
		}
		log.Infof("new base os created %s", baseOSConfig.Uuidandversion.Uuid)
	}

	// provision content tree and volume in addition to Drive record in config
	if len(baseOSConfig.Drives) == 1 {
		contentTree := exp.imageToContentTree(image, image.Name)
		_ = exp.ctrl.AddContentTree(contentTree)
		exp.device.SetContentTreeConfig(append(exp.device.GetContentTrees(), contentTree.Uuid))

		volume := exp.driveToVolume(baseOSConfig.Drives[0], 0, contentTree)
		_ = exp.ctrl.AddVolume(volume)
		exp.device.SetVolumeConfigs(append(exp.device.GetVolumes(), volume.Uuid))
	}

	return
}
