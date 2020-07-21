package expect

import (
	"encoding/base64"
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

//createImageDocker creates Image for docker with tag and version from appExpectation and provided id and datastoreId
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

//createAppInstanceConfigDocker creates AppInstanceConfig for docker with provided img, netInstance, id and acls
//  it uses name of app and cpu/mem params from appExpectation
func (exp *appExpectation) createAppInstanceConfigDocker(img *config.Image, id uuid.UUID) *config.AppInstanceConfig {
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
	app.Drives = []*config.Drive{{
		Image:        img,
		Maxsizebytes: maxSizeBytes,
	}}
	return app
}
