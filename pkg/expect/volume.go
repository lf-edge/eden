package expect

import (
	"fmt"

	"github.com/lf-edge/eve-api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// driveToVolume converts information about drive, its number and content tree into volume representation
func (exp *AppExpectation) driveToVolume(dr *config.Drive, numberOfDrive int, contentTree *config.ContentTree) *config.Volume {
	for _, volID := range exp.device.GetVolumes() {
		el, err := exp.ctrl.GetVolume(volID)
		if err != nil {
			log.Fatalf("no volume %s found in controller: %s", volID, err)
		}
		if el.DisplayName == fmt.Sprintf("%s_%d_m_0", contentTree.DisplayName, numberOfDrive) {
			// we already have this one in controller
			return el
		}
	}
	id, err := uuid.NewV4()
	if err != nil {
		log.Fatal(err)
	}
	volume := &config.Volume{
		Uuid: id.String(),
		Origin: &config.VolumeContentOrigin{
			Type:                  config.VolumeContentOriginType_VCOT_DOWNLOAD,
			DownloadContentTreeID: contentTree.Uuid,
		},
		Protocols:    nil,
		Maxsizebytes: dr.Maxsizebytes,
		DisplayName:  fmt.Sprintf("%s_%d_m_0", exp.appName, numberOfDrive),
	}
	_ = exp.ctrl.AddVolume(volume)
	return volume
}

// Volume generates volume for provided expectation
func (exp *AppExpectation) Volume() *config.Volume {
	img := exp.Image()

	maxSizeBytes := int64(0)
	if exp.diskSize > 0 {
		maxSizeBytes = exp.diskSize
	}
	drive := &config.Drive{
		Image:        img,
		Maxsizebytes: maxSizeBytes,
	}
	contentTree := exp.imageToContentTree(img, img.Name)
	_ = exp.ctrl.AddContentTree(contentTree)
	exp.device.SetContentTreeConfig(append(exp.device.GetContentTrees(), contentTree.Uuid))
	volume := exp.driveToVolume(drive, 0, contentTree)
	volume.DisplayName = exp.appName
	_ = exp.ctrl.AddVolume(volume)
	exp.device.SetVolumeConfigs(append(exp.device.GetVolumes(), volume.Uuid))
	return volume
}
