package expect

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
)

func (exp *appExpectation) createImageHttp(id uuid.UUID, dsId string) *config.Image {
	log.Infof("Starting download of image from %s", exp.appLink)
	filePath := filepath.Join(exp.ctrl.GetVars().EServerImageDist, path.Base(exp.appUrl))
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		log.Info("file already exists %s", filePath)
	} else {
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			log.Fatal(err)
		}
		if err := utils.DownloadFile(filePath, exp.appLink); err != nil {
			log.Fatal(err)
		}
	}
	fileSize := utils.GetFileSize(filePath)
	sha256, err := utils.SHA256SUM(filePath)
	if err != nil {
		log.Fatal(err)
	}
	return &config.Image{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		Name:      path.Base(exp.appUrl),
		Iformat:   config.Format_QCOW2,
		DsId:      dsId,
		Siginfo:   &config.SignatureInfo{},
		SizeBytes: fileSize,
		Sha256:    sha256,
	}
}

func (exp *appExpectation) checkImageHttp(img *config.Image, dsId string) bool {
	if img.DsId == dsId && img.Name == path.Join(exp.appName, path.Base(exp.appUrl)) && img.Iformat == config.Format_QCOW2 {
		return true
	}
	return false
}

func (exp *appExpectation) checkDataStoreHttp(ds *config.DatastoreConfig) bool {
	if ds.DType == config.DsType_DsHttp && ds.Fqdn == fmt.Sprintf("http://%s:%s", exp.ctrl.GetVars().AdamDomain, exp.ctrl.GetVars().EServerPort) {
		return true
	}
	return false
}

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
