package expect

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//createImageHTTP downloads image into EServer directory from http/https endpoint and calculates size and sha256 of image
func (exp *AppExpectation) createImageHTTP(id uuid.UUID, dsID string) *config.Image {
	log.Infof("Starting download of image from %s", exp.appLink)
	server := &eden.EServer{
		EServerIP:   exp.ctrl.GetVars().EServerIP,
		EServerPort: exp.ctrl.GetVars().EServerPort,
	}
	var fileSize int64
	sha256 := ""
	filePath := ""
	if el, stored := defaults.ImageStore[exp.appLink]; exp.httpDirectLoad && stored {
		sha256 = el.Sha256
		fileSize = el.Size
	} else {
		name := server.EServerAddFileURL(exp.appLink)
		log.Infof("Start download into eserver of %s", name)

		delayTime := defaults.DefaultRepeatTimeout

		for {
			status := server.EServerCheckStatus(name)
			if !status.ISReady {
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
	}
	if exp.httpDirectLoad {
		u, err := url.Parse(exp.appLink)
		if err != nil {
			log.Fatal(err)
		}
		filePath = strings.TrimLeft(u.RequestURI(), "/")
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

//checkImageHTTP checks if provided img match expectation
func (exp *AppExpectation) checkImageHTTP(img *config.Image, dsID string) bool {
	if img.DsId == dsID && img.Name == path.Join("eserver", path.Base(exp.appURL)) && img.Iformat == config.Format_QCOW2 {
		return true
	}
	return false
}

//checkDataStoreHTTP checks if provided ds match expectation
func (exp *AppExpectation) checkDataStoreHTTP(ds *config.DatastoreConfig) bool {
	if ds.DType == config.DsType_DsHttp || ds.DType == config.DsType_DsHttps {
		if exp.httpDirectLoad && ds.Fqdn == fmt.Sprintf("http://%s:%s", exp.ctrl.GetVars().AdamDomain, exp.ctrl.GetVars().EServerPort) {
			return true
		}
		u, err := url.Parse(exp.appLink)
		if err != nil {
			log.Fatal(err)
		}
		if !exp.httpDirectLoad && ds.Fqdn == fmt.Sprintf("%s://%s", u.Scheme, u.Hostname()) {
			return true
		}
	}
	return false
}

//createDataStoreHTTP creates datastore, pointed onto EServer http endpoint
func (exp *AppExpectation) createDataStoreHTTP(id uuid.UUID) *config.DatastoreConfig {
	ds := &config.DatastoreConfig{
		Id:         id.String(),
		DType:      config.DsType_DsHttp,
		ApiKey:     "",
		Password:   "",
		Dpath:      "",
		Region:     "",
		CipherData: nil,
	}
	if exp.httpDirectLoad && exp.appType != fileApp {
		u, err := url.Parse(exp.appLink)
		if err != nil {
			log.Fatal(err)
		}
		if u.Scheme == "https" {
			ds.DType = config.DsType_DsHttps
		}
		ds.Fqdn = fmt.Sprintf("%s://%s", u.Scheme, u.Hostname())
	} else {
		ds.Fqdn = fmt.Sprintf("http://%s:%s", exp.ctrl.GetVars().AdamDomain, exp.ctrl.GetVars().EServerPort)
	}
	return ds
}
