package expect

import (
	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"path/filepath"
)

//createImageFile uploads image into EServer from file and calculates size and sha256 of image
func (exp *AppExpectation) createImageFile(id uuid.UUID, dsID string) *config.Image {
	server := &eden.EServer{
		EServerIP:   exp.ctrl.GetVars().EServerIP,
		EserverPort: exp.ctrl.GetVars().EServerPort,
	}
	var fileSize int64
	sha256 := ""
	filePath := ""
	status := server.EServerCheckStatus(filepath.Base(exp.appURL))
	if !status.ISReady || status.Size != utils.GetFileSize(exp.appURL) {
		log.Infof("Start uploading into eserver of %s", exp.appLink)
		status = server.EServerAddFile(exp.appURL)
		if status.Error != "" {
			log.Error(status.Error)
		}
	}
	sha256 = status.Sha256
	fileSize = status.Size
	filePath = status.FileName
	log.Infof("Image uploaded with size %s and sha256 %s", humanize.Bytes(uint64(status.Size)), status.Sha256)
	if filePath == "" {
		log.Fatal("Not uploaded")
	}
	return &config.Image{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		Name:      filePath,
		Iformat:   exp.imageFormatEnum(),
		DsId:      dsID,
		SizeBytes: fileSize,
		Sha256:    sha256,
	}
}
