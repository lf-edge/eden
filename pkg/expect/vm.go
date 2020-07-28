package expect

import (
	"encoding/base64"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

//createAppInstanceConfigVM creates AppInstanceConfig for VM with provided img, netInstance, id and acls
//  it uses name of app and cpu/mem params from appExpectation
//  it use ZArch param to choose VirtualizationMode
func (exp *appExpectation) createAppInstanceConfigVM(img *config.Image, id uuid.UUID) *config.AppInstanceConfig {
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
	if exp.virtualizationMode == config.VmMode_PV {
		app.Fixedresources.Rootdev = "/dev/xvda1"
		app.Fixedresources.Bootloader = "/usr/bin/pygrub"
	}
	app.Fixedresources.VirtualizationMode = exp.virtualizationMode
	maxSizeBytes := img.SizeBytes
	if exp.diskSize > 0 {
		maxSizeBytes = exp.diskSize
	}
	app.Drives = []*config.Drive{{
		Image:        img,
		Readonly:     false,
		Drvtype:      config.DriveType_HDD,
		Target:       config.Target_Disk,
		Maxsizebytes: maxSizeBytes,
	}}
	return app
}
