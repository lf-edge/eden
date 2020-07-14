package expect

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

//parse file or url name and returns Base OS Version
func (exp *appExpectation) getBaseOSVersion() string {
	rootFSName := path.Base(exp.appUrl)
	rootFSName = strings.TrimSuffix(rootFSName, filepath.Ext(rootFSName))
	rootFSName = strings.TrimPrefix(rootFSName, "rootfs-")
	if re := regexp.MustCompile(defaults.DefaultRootFSVersionPattern); !re.MatchString(rootFSName) {
		log.Fatalf("Filename of rootfs %s does not match pattern %s", rootFSName, defaults.DefaultRootFSVersionPattern)
	}
	return rootFSName
}

//checkBaseOSConfig checks if provided BaseOSConfig match expectation
func (exp *appExpectation) checkBaseOSConfig(baseOS *config.BaseOSConfig) bool {
	if baseOS == nil {
		return false
	}
	if baseOS.BaseOSVersion == exp.getBaseOSVersion() {
		return true
	}
	return false
}

//createBaseOSConfig creates BaseOSConfig with provided img
func (exp *appExpectation) createBaseOSConfig(img *config.Image) (*config.BaseOSConfig, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	baseOSConfig := &config.BaseOSConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
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
		BaseOSDetails: nil,
	}
	switch exp.appType {
	case dockerApp:
		return nil, fmt.Errorf("cannot create base os image from docker")
	case httpApp, httpsApp, fileApp:
		return baseOSConfig, nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

//BaseOSImage expectation gets or creates Image definition,
//gets BaseOSConfig and returns it or creates BaseOSConfig, adds it into internal controller and returns it
func (exp *appExpectation) BaseOSImage() (baseOSConfig *config.BaseOSConfig) {
	var err error
	if exp.appType == fileApp {
		usr, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		newPath := exp.appUrl
		dir := usr.HomeDir
		if newPath == "~" {
			newPath = dir
		} else if strings.HasPrefix(newPath, "~/") {
			newPath = filepath.Join(dir, newPath[2:])
		}
		if exp.appUrl, err = utils.GetFileFollowLinks(newPath); err != nil {
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
