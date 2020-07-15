package expect

import (
	"encoding/base64"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	"log"
)

//createAppInstanceConfigVM creates AppInstanceConfig for VM with provided img, netInstance, id and acls
//  it uses name of app and cpu/mem params from appExpectation
//  it use ZArch param to choose VirtualizationMode
func (exp *appExpectation) createAppInstanceConfigVM(img *config.Image, netInstId string, id uuid.UUID, acls []*config.ACE) *config.AppInstanceConfig {
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
		UserData:    base64.StdEncoding.EncodeToString([]byte(exp.metadata)),
		Activate:    true,
		Displayname: exp.appName,
		Interfaces: []*config.NetworkAdapter{{
			Name:      "default",
			NetworkId: netInstId,
			Acls:      acls,
		}},
	}
	switch exp.ctrl.GetVars().ZArch {
	case "amd64":
		app.Fixedresources.VirtualizationMode = config.VmMode_HVM
	case "arm64":
		app.Fixedresources.VirtualizationMode = config.VmMode_PV
		app.Fixedresources.Rootdev = "/dev/xvda1"
		app.Fixedresources.Bootloader = "/usr/bin/pygrub"
	default:
		log.Fatalf("Unexpected arch %s", exp.ctrl.GetVars().ZArch)
	}
	app.Drives = []*config.Drive{{
		Image:    img,
		Readonly: false,
		Drvtype:  config.DriveType_HDD,
		Target:   config.Target_Disk,
	}}
	return app
}
