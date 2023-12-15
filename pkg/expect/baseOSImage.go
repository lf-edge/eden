package expect

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/config"
	log "github.com/sirupsen/logrus"
)

// parse file or url name and returns Base OS Version
func (exp *AppExpectation) getBaseOSVersion() string {
	if exp.baseOSVersion != "" {
		return exp.baseOSVersion
	}
	if exp.appType == dockerApp {
		return exp.appVersion
	}

	correctionFileName := fmt.Sprintf("%s.ver", exp.appURL)
	if rootFSFromCorrectionFile, err := os.ReadFile(correctionFileName); err == nil {
		return string(rootFSFromCorrectionFile)
	}
	rootFSName := path.Base(exp.appURL)
	rootFSName = strings.TrimSuffix(rootFSName, filepath.Ext(rootFSName))
	rootFSName = strings.TrimPrefix(rootFSName, "rootfs-")
	if re := regexp.MustCompile(defaults.DefaultRootFSVersionPattern); !re.MatchString(rootFSName) {
		log.Warnf("Filename of rootfs %s does not match pattern %s", rootFSName, defaults.DefaultRootFSVersionPattern)
		// check for eve_version file
		if v, err := os.ReadFile(filepath.Join(filepath.Dir(exp.appURL), "eve_version")); err == nil {
			baseOSVersion := strings.TrimSpace(string(v))
			log.Warnf("Will use version from eve_version file: %s", baseOSVersion)
			return baseOSVersion
		}
		log.Fatalf("Cannot use provided file: version unknown, please provide it with --os-version flag")
	}
	return rootFSName
}

// checkBaseOSConfig checks if provided BaseOSConfig match expectation
func (exp *AppExpectation) checkBaseOS(baseOS *config.BaseOS) bool {
	if baseOS == nil {
		return false
	}
	return baseOS.BaseOsVersion == exp.getBaseOSVersion()
}

// checkBaseOSConfig checks if provided BaseOSConfig match expectation
func (exp *AppExpectation) checkBaseOSConfig(baseOSConfig *config.BaseOSConfig) bool {
	if baseOSConfig == nil {
		return false
	}
	if baseOSConfig.BaseOSVersion == exp.getBaseOSVersion() {
		return true
	}
	return false
}

// createBaseOSConfig creates BaseOSConfig with provided img
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

// BaseOSConfig expectation gets or creates BaseOSConfig definition,
// adds it into internal controller and returns it
// if version is not empty will use it as BaseOSVersion
func (exp *AppExpectation) BaseOSConfig(baseOSVersion string) (baseOSConfig *config.BaseOSConfig) {
	exp.baseOSVersion = baseOSVersion
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

	return
}

// BaseOS expectation gets or creates BaseOS definition,
// adds contentTree into internal controller and returns BaseOS
// if version is not empty will use it as BaseOSVersion
func (exp *AppExpectation) BaseOS(baseOSVersion string) (baseOS *config.BaseOS) {
	exp.baseOSVersion = baseOSVersion
	var err error
	if exp.appType == fileApp {
		if exp.appURL, err = utils.GetFileFollowLinks(exp.appURL); err != nil {
			log.Fatalf("GetFileFollowLinks: %s", err)
		}
	}
	image := exp.Image()
	contentTree := exp.imageToContentTree(image, image.Name)
	_ = exp.ctrl.AddContentTree(contentTree)
	exp.device.SetContentTreeConfig(append(exp.device.GetContentTrees(), contentTree.Uuid))
	baseOS = &config.BaseOS{
		ContentTreeUuid: contentTree.GetUuid(),
		BaseOsVersion:   exp.getBaseOSVersion(),
	}

	return
}
