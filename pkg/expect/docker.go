package expect

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/edge-containers/pkg/registry"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//createImageDocker creates Image for docker with tag and version from appExpectation and provided id and datastoreId
func (exp *appExpectation) createImageDocker(id uuid.UUID, dsId string) *config.Image {
	ref, err := name.ParseReference(exp.appUrl)
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
		DsId:    dsId,
	}
}

//checkImageDocker checks if provided img match expectation
func (exp *appExpectation) checkImageDocker(img *config.Image, dsId string) bool {
	if img.DsId == dsId && img.Name == fmt.Sprintf("%s:%s", exp.appUrl, exp.appVersion) && img.Iformat == config.Format_CONTAINER {
		return true
	}
	return false
}

//checkDataStoreDocker checks if provided ds match expectation
func (exp *appExpectation) checkDataStoreDocker(ds *config.DatastoreConfig) bool {
	if ds.DType == config.DsType_DsContainerRegistry && ds.Fqdn == "docker://docker.io" {
		return true
	}
	return false
}

//createDataStoreDocker creates DatastoreConfig for docker.io with provided id
func (exp *appExpectation) createDataStoreDocker(id uuid.UUID) *config.DatastoreConfig {
	ref, err := name.ParseReference(exp.appUrl)
	if err != nil {
		return nil
	}
	return &config.DatastoreConfig{
		Id:         id.String(),
		DType:      config.DsType_DsContainerRegistry,
		Fqdn:       fmt.Sprintf("docker://%s", ref.Context().Registry.Name()),
		ApiKey:     "",
		Password:   "",
		Dpath:      "",
		Region:     "",
		CipherData: nil,
	}
}

//applyRootFSType try to parse manifest to get Annotations provided in https://github.com/lf-edge/edge-containers/blob/master/docs/annotations.md
func applyRootFSType(image *config.Image) error {
	manifest, err := crane.Manifest(image.Name)
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
				break
			case registry.MimeTypeECIDiskQcow:
				image.Iformat = config.Format_QCOW
				break
			case registry.MimeTypeECIDiskQcow2:
				image.Iformat = config.Format_QCOW2
				break
			case registry.MimeTypeECIDiskVhd:
				image.Iformat = config.Format_VHD
				break
			case registry.MimeTypeECIDiskVmdk:
				image.Iformat = config.Format_VMDK
				break
			case registry.MimeTypeECIDiskOva:
				image.Iformat = config.Format_OVA
				break
			case registry.MimeTypeECIDiskVhdx:
				image.Iformat = config.Format_VHDX
				break
			}
		}
	}
	return nil
}

//obtainVolumeInfo try to parse docker manifest of defined image and return array of mount points
func obtainVolumeInfo(image *config.Image) ([]string, error) {
	cfg, err := crane.Config(image.Name)
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

//prepareImage generates new image for mountable volume
func (exp *appExpectation) prepareImage() *config.Image {
	appLink := defaults.DefaultEmptyVolumeLinkQcow2
	if exp.volumesType == VolumeOCI {
		appLink = defaults.DefaultEmptyVolumeLinkDocker
	}
	if !strings.Contains(appLink, "://") {
		//if we use file, we must resolve absolute path
		appLink = fmt.Sprintf("file://%s", utils.ResolveAbsPath(appLink))
	}
	tempExp := AppExpectationFromUrl(exp.ctrl, exp.device, appLink, "")
	return tempExp.Image()
}

//createAppInstanceConfigDocker creates appBundle for docker with provided img, netInstance, id and acls
//  it uses name of app and cpu/mem params from appExpectation
func (exp *appExpectation) createAppInstanceConfigDocker(img *config.Image, id uuid.UUID) *appBundle {
	log.Infof("Try to obtain info about volumes, please wait")
	mountPointsList, err := obtainVolumeInfo(img)
	if err != nil {
		//if something wrong with info about image, just print information
		log.Errorf("cannot obtain info about volumes: %v", err)
	}
	log.Infof("Try to obtain info about disks, please wait")
	if err := applyRootFSType(img); err != nil {
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
		UserData:    base64.StdEncoding.EncodeToString([]byte(exp.metadata)),
		Activate:    true,
		Displayname: exp.appName,
	}
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

	// we need to add volumes for every mount point
	for ind, el := range mountPointsList {
		image := exp.prepareImage()
		drive := &config.Drive{
			Image:        image,
			Maxsizebytes: defaults.DefaultVolumeSize,
		}
		contentTree := exp.imageToContentTree(image, fmt.Sprintf("%s-%d", exp.appName, ind))
		contentTrees = append(contentTrees, contentTree)
		volume := exp.driveToVolume(drive, ind+1, contentTree)
		volumes = append(volumes, volume)
		app.VolumeRefList = append(app.VolumeRefList, &config.VolumeRef{MountDir: el, Uuid: volume.Uuid})
	}
	app.Fixedresources.VirtualizationMode = exp.virtualizationMode
	return &appBundle{
		appInstanceConfig: app,
		contentTrees:      contentTrees,
		volumes:           volumes,
	}
}
