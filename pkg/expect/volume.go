package expect

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//driveToVolume converts information about drive, its number and content tree into volume representation
func (exp *AppExpectation) driveToVolume(dr *config.Drive, numberOfDrive int, contentTree *config.ContentTree) *config.Volume {
	for _, el := range exp.ctrl.ListVolume() {
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
