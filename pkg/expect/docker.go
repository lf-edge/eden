package expect

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

func (exp *appExpectation) createImageDocker(id uuid.UUID, dsId string) *config.Image {
	return &config.Image{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		Name:    fmt.Sprintf("%s:%s", exp.appUrl, exp.appVersion),
		Iformat: config.Format_CONTAINER,
		DsId:    dsId,
		Siginfo: &config.SignatureInfo{},
	}
}

func (exp *appExpectation) checkImageDocker(img *config.Image, dsId string) bool {
	if img.DsId == dsId && img.Name == fmt.Sprintf("%s:%s", exp.appUrl, exp.appVersion) && img.Iformat == config.Format_CONTAINER {
		return true
	}
	return false
}

func (exp *appExpectation) checkDataStoreDocker(ds *config.DatastoreConfig) bool {
	if ds.DType == config.DsType_DsContainerRegistry && ds.Fqdn == "docker://docker.io" {
		return true
	}
	return false
}

func (exp *appExpectation) createDataStoreDocker(id uuid.UUID) *config.DatastoreConfig {
	return &config.DatastoreConfig{
		Id:         id.String(),
		DType:      config.DsType_DsContainerRegistry,
		Fqdn:       "docker://docker.io",
		ApiKey:     "",
		Password:   "",
		Dpath:      "",
		Region:     "",
		CipherData: nil,
	}
}

func (exp *appExpectation) createAppInstanceConfigDocker(img *config.Image, netInstId string, id uuid.UUID, acls []*config.ACE) *config.AppInstanceConfig {
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
		Drives: []*config.Drive{{
			Image: img,
		}},
		Activate:    true,
		Displayname: exp.appName,
		Interfaces: []*config.NetworkAdapter{{
			Name:      "default",
			NetworkId: netInstId,
			Acls:      acls,
		}},
	}
	app.Drives = []*config.Drive{{
		Image: img,
	}}
	return app
}
