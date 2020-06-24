package expect

import (
	"encoding/base64"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

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
	app.Fixedresources.VirtualizationMode = config.VmMode_HVM
	app.Drives = []*config.Drive{{
		Image:    img,
		Readonly: false,
		Drvtype:  config.DriveType_HDD,
		Target:   config.Target_Disk,
	}}
	return app
}
