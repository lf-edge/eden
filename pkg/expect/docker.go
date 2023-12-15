package expect

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/edge-containers/pkg/registry"
	"github.com/lf-edge/eve-api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// createImageDocker creates Image for docker with tag and version from AppExpectation and provided id and datastoreId
func (exp *AppExpectation) createImageDocker(id uuid.UUID, dsID string) *config.Image {
	ref, err := name.ParseReference(exp.appURL)
	if err != nil {
		return nil
	}
	return &config.Image{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		Name:    fmt.Sprintf("%s:%s", ref.Context().RepositoryStr(), exp.appVersion),
		Iformat: exp.imageFormatEnum(),
		DsId:    dsID,
	}
}

// checkImageDocker checks if provided img match expectation
func (exp *AppExpectation) checkImageDocker(img *config.Image, dsID string) bool {
	if img.DsId == dsID && img.Name == fmt.Sprintf("%s:%s", exp.appURL, exp.appVersion) && img.Iformat == config.Format_CONTAINER {
		return true
	}
	return false
}

// getDataStoreFQDN return fqdn info for datastore based on provided ref of image and registry
func (exp *AppExpectation) getDataStoreFQDN(withProto bool) string {
	if exp.datastoreOverride != "" {
		return exp.datastoreOverride
	}
	var fqdn string
	if exp.registry != "" {
		fqdn = exp.registry
	} else {
		ref, err := name.ParseReference(exp.appURL)
		if err != nil {
			return ""
		}
		fqdn = ref.Context().Registry.Name()
	}
	if withProto {
		fqdn = fmt.Sprintf("docker://%s", fqdn)
	}
	return fqdn
}

// checkDataStoreDocker checks if provided ds match expectation
func (exp *AppExpectation) checkDataStoreDocker(ds *config.DatastoreConfig) bool {
	if ds.DType == config.DsType_DsContainerRegistry && ds.Fqdn == exp.getDataStoreFQDN(true) {
		return true
	}
	return false
}

// createDataStoreDocker creates DatastoreConfig for docker.io with provided id
func (exp *AppExpectation) createDataStoreDocker(id uuid.UUID) *config.DatastoreConfig {
	return &config.DatastoreConfig{
		Id:         id.String(),
		DType:      config.DsType_DsContainerRegistry,
		Fqdn:       exp.getDataStoreFQDN(true),
		ApiKey:     "",
		Password:   "",
		Dpath:      "",
		Region:     "",
		CipherData: nil,
	}
}

// applyRootFSType try to parse manifest to get Annotations provided in https://github.com/lf-edge/edge-containers/blob/master/docs/annotations.md
func (exp *AppExpectation) applyRootFSType(image *config.Image) error {
	if exp.appLink == defaults.DefaultDummyExpect {
		log.Debug("skip applyRootFSType")
		return nil
	}
	ref := fmt.Sprintf("%s/%s", exp.getDataStoreFQDN(false), image.Name)
	manifest, err := crane.Manifest(ref)
	if err != nil {
		return err
	}
	manifestFile, err := v1.ParseManifest(bytes.NewReader(manifest))
	if err != nil {
		return err
	}
	for _, el := range manifestFile.Layers {
		if val, ok := el.Annotations[registry.AnnotationRole]; !ok || val != registry.RoleRootDisk {
			continue
		}
		if mediaType, ok := el.Annotations[registry.AnnotationMediaType]; ok {
			switch mediaType {
			case registry.MimeTypeECIDiskRaw:
				image.Iformat = config.Format_RAW
			case registry.MimeTypeECIDiskQcow:
				image.Iformat = config.Format_QCOW
			case registry.MimeTypeECIDiskQcow2:
				image.Iformat = config.Format_QCOW2
			case registry.MimeTypeECIDiskVhd:
				image.Iformat = config.Format_VHD
			case registry.MimeTypeECIDiskVmdk:
				image.Iformat = config.Format_VMDK
			case registry.MimeTypeECIDiskOva:
				image.Iformat = config.Format_OVA
			case registry.MimeTypeECIDiskVhdx:
				image.Iformat = config.Format_VHDX
			}
		}
	}
	return nil
}

// obtainVolumeInfo try to parse docker manifest of defined image and return array of mount points
func (exp *AppExpectation) obtainVolumeInfo(image *config.Image) ([]string, error) {
	if exp.appLink == defaults.DefaultDummyExpect {
		log.Debug("skip obtainVolumeInfo")
		return nil, nil
	}
	ref := fmt.Sprintf("%s/%s", exp.getDataStoreFQDN(false), image.Name)
	cfg, err := crane.Config(ref)
	if err != nil {
		return nil, fmt.Errorf("error getting config %s: %v", image.Name, err)
	}
	// parse the config file
	configFile, err := v1.ParseConfigFile(bytes.NewReader(cfg))
	if err != nil {
		return nil, fmt.Errorf("unable to parse config file: %v", err)
	}

	var mountPoints []string

	//read docker image config
	for key := range configFile.Config.Volumes {
		log.Infof("volumes MountDir: %s", key)
		mountPoints = append(mountPoints, key)
	}
	return mountPoints, nil
}

// prepareImage generates new image for mountable volume
func (exp *AppExpectation) prepareImage() *config.Image {
	appLink := defaults.DefaultEmptyVolumeLinkQcow2
	switch exp.volumesType {
	case VolumeQcow2:
		appLink = defaults.DefaultEmptyVolumeLinkQcow2
	case VolumeOCI:
		appLink = defaults.DefaultEmptyVolumeLinkDocker
	case VolumeRaw:
		appLink = defaults.DefaultEmptyVolumeLinkRaw
	case VolumeQcow:
		appLink = defaults.DefaultEmptyVolumeLinkQcow
	case VolumeVHDX:
		appLink = defaults.DefaultEmptyVolumeLinkVHDX
	case VolumeVMDK:
		appLink = defaults.DefaultEmptyVolumeLinkVMDK
	case VolumeNone:
		return nil
	}
	if !strings.Contains(appLink, "://") {
		//if we use file, we must resolve absolute path
		appLink = fmt.Sprintf("file://%s", utils.ResolveAbsPath(appLink))
	}
	tempExp := AppExpectationFromURL(exp.ctrl, exp.device, appLink, "")
	tempExp.imageFormat = string(exp.volumesType)
	return tempExp.Image()
}

// createAppInstanceConfigDocker creates appBundle for docker with provided img, netInstance, id and acls
//
//	it uses name of app and cpu/mem params from AppExpectation
func (exp *AppExpectation) createAppInstanceConfigDocker(img *config.Image, id uuid.UUID) *appBundle {
	log.Debugf("Try to obtain info about volumes, please wait")
	mountPointsList, err := exp.obtainVolumeInfo(img)
	if err != nil {
		//if something wrong with info about image, just print information
		log.Errorf("cannot obtain info about volumes: %v", err)
	}
	log.Debugf("Try to obtain info about disks, please wait")
	if err := exp.applyRootFSType(img); err != nil {
		//if something wrong with info about disks, just print information
		log.Errorf("cannot obtain info about disks: %v", err)
	}
	app := &config.AppInstanceConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		Fixedresources: &config.VmConfig{
			Memory: exp.mem,
			Maxmem: exp.mem,
			Vcpus:  exp.cpu,
		},
		Activate:    true,
		Displayname: exp.appName,
	}
	exp.applyUserData(app)
	maxSizeBytes := int64(0)
	if exp.diskSize > 0 {
		maxSizeBytes = exp.diskSize
	}
	drive := &config.Drive{
		Image:        img,
		Maxsizebytes: maxSizeBytes,
	}
	app.Drives = []*config.Drive{drive}
	contentTree := exp.imageToContentTree(img, img.Name)
	contentTrees := []*config.ContentTree{contentTree}
	volume := exp.driveToVolume(drive, 0, contentTree)
	volumes := []*config.Volume{volume}
	app.VolumeRefList = []*config.VolumeRef{{MountDir: "/", Uuid: volume.Uuid}}

	if len(mountPointsList) > 0 {
		// we need to add volumes for every mount point
		image := exp.prepareImage()
		for ind, el := range mountPointsList {
			if image != nil {
				drive := &config.Drive{
					Image:        image,
					Maxsizebytes: exp.volumeSize,
				}
				toAppend := true
				var contentTree *config.ContentTree
				for _, ct := range contentTrees {
					if ct.URL == image.Name && ct.Sha256 == image.Sha256 {
						//skip append of existent ContentTree
						toAppend = false
						contentTree = ct
					}
				}
				if toAppend {
					contentTree = exp.imageToContentTree(image, fmt.Sprintf("%s-%d", exp.appName, ind))
					contentTrees = append(contentTrees, contentTree)
				}
				volume := exp.driveToVolume(drive, ind+1, contentTree)
				volumes = append(volumes, volume)
				app.VolumeRefList = append(app.VolumeRefList, &config.VolumeRef{MountDir: el, Uuid: volume.Uuid})
			}
		}
	}
	app.Fixedresources.VirtualizationMode = exp.virtualizationMode
	return &appBundle{
		appInstanceConfig: app,
		contentTrees:      contentTrees,
		volumes:           volumes,
	}
}
