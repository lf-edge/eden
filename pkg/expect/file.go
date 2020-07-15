package expect

import (
	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//createImageFile uploads image into EServer from file and calculates size and sha256 of image
func (exp *appExpectation) createImageFile(id uuid.UUID, dsId string) *config.Image {
	server := &utils.EServer{
		EServerIP:   exp.ctrl.GetVars().EServerIp,
		EserverPort: exp.ctrl.GetVars().EServerPort,
	}
	var fileSize int64
	sha256 := ""
	filePath := ""
	log.Infof("Start uploading into eserver of %s", exp.appLink)
	status := server.EServerAddFile(exp.appUrl)
	if status.Error != "" {
		log.Error(status.Error)
	}
	sha256 = status.Sha256
	fileSize = status.Size
	filePath = status.FileName
	log.Infof("Image uploaded with size %s and sha256 %s", humanize.Bytes(uint64(status.Size)), sha256)
	if filePath == "" {
		log.Fatal("Not uploaded")
	}
	return &config.Image{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		Name:      filePath,
		Iformat:   config.Format_QCOW2,
		DsId:      dsId,
		Siginfo:   &config.SignatureInfo{},
		SizeBytes: fileSize,
		Sha256:    sha256,
	}
}
