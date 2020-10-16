package expect

import (
	"encoding/base64"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
)

//createAppInstanceConfigVM creates appBundle for VM with provided img, netInstance, id and acls
//  it uses name of app and cpu/mem params from AppExpectation
//  it use ZArch param to choose VirtualizationMode
func (exp *AppExpectation) createAppInstanceConfigVM(img *config.Image, id uuid.UUID) *appBundle {
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
	drive := &config.Drive{
		Image:        img,
		Readonly:     false,
		Drvtype:      config.DriveType_HDD,
		Target:       config.Target_Disk,
		Maxsizebytes: maxSizeBytes,
	}
	app.Drives = []*config.Drive{drive}
	contentTree := exp.imageToContentTree(img, exp.appName)
	contentTrees := []*config.ContentTree{contentTree}
	volume := exp.driveToVolume(drive, 0, contentTree)
	volumes := []*config.Volume{volume}
	app.VolumeRefList = []*config.VolumeRef{{MountDir: "/", Uuid: volume.Uuid}}

	return &appBundle{
		appInstanceConfig: app,
		contentTrees:      contentTrees,
		volumes:           volumes,
	}
}
