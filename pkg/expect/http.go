package expect

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"path"
	"time"
)

//createImageHttp downloads image into EServer directory from http/https endpoint and calculates size and sha256 of image
func (exp *appExpectation) createImageHttp(id uuid.UUID, dsId string) *config.Image {
	log.Infof("Starting download of image from %s", exp.appLink)
	server := &utils.EServer{
		EServerIP:   exp.ctrl.GetVars().EServerIp,
		EserverPort: exp.ctrl.GetVars().EServerPort,
	}
	var fileSize int64
	sha256 := ""
	filePath := ""
	name := server.EServerAddFileUrl(exp.appLink)
	log.Infof("Start download into eserver of %s", name)

	delayTime := defaults.DefaultRepeatTimeout

	for {
		status := server.EServerCheckStatus(name)
		if status.ISReady == false {
			log.Infof("Downloading... Ready %s", humanize.Bytes(uint64(status.Size)))
		} else {
			sha256 = status.Sha256
			fileSize = status.Size
			filePath = status.FileName
			log.Infof("Image downloaded with size %s and sha256 %s", humanize.Bytes(uint64(status.Size)), sha256)
			break
		}
		time.Sleep(delayTime)
	}
	if filePath == "" {
		log.Fatal("Not downloaded")
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

//checkImageHttp checks if provided img match expectation
func (exp *appExpectation) checkImageHttp(img *config.Image, dsId string) bool {
	if img.DsId == dsId && img.Name == path.Join("eserver", path.Base(exp.appUrl)) && img.Iformat == config.Format_QCOW2 {
		return true
	}
	return false
}

//checkDataStoreHttp checks if provided ds match expectation
func (exp *appExpectation) checkDataStoreHttp(ds *config.DatastoreConfig) bool {
	if ds.DType == config.DsType_DsHttp && ds.Fqdn == fmt.Sprintf("http://%s:%s", exp.ctrl.GetVars().AdamDomain, exp.ctrl.GetVars().EServerPort) {
		return true
	}
	return false
}

//createDataStoreHttp creates datastore, pointed onto EServer http endpoint
func (exp *appExpectation) createDataStoreHttp(id uuid.UUID) *config.DatastoreConfig {
	return &config.DatastoreConfig{
		Id:         id.String(),
		DType:      config.DsType_DsHttp,
		Fqdn:       fmt.Sprintf("http://%s:%s", exp.ctrl.GetVars().AdamDomain, exp.ctrl.GetVars().EServerPort),
		ApiKey:     "",
		Password:   "",
		Dpath:      "",
		Region:     "",
		CipherData: nil,
	}
}
