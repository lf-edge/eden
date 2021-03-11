package expect

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/evecommon"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//appBundle type for aggregate objects, needed for application
type appBundle struct {
	appInstanceConfig *config.AppInstanceConfig
	contentTrees      []*config.ContentTree
	volumes           []*config.Volume
}

//checkAppInstanceConfig checks if provided app match expectation
func (exp *AppExpectation) checkAppInstanceConfig(app *config.AppInstanceConfig) bool {
	if app == nil {
		return false
	}
	if app.Displayname == exp.appName && app.Displayname != exp.oldAppName {
		return true
	}
	return false
}

//createAppInstanceConfig creates AppInstanceConfig with provided img and netInstances
//  it uses published ports info from AppExpectation to create ACE
func (exp *AppExpectation) createAppInstanceConfig(img *config.Image, netInstances map[*NetInstanceExpectation]*config.NetworkInstanceConfig) (*appBundle, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	var bundle *appBundle
	switch exp.appType {
	case dockerApp:
		bundle = exp.createAppInstanceConfigDocker(img, id)
	case httpApp, httpsApp, fileApp:
		bundle = exp.createAppInstanceConfigVM(img, id)
	default:
		return nil, fmt.Errorf("not supported appType")
	}
	for _, d := range exp.disks {
		tempExp := AppExpectationFromURL(exp.ctrl, exp.device, d, "")
		tempExp.imageFormat = string(exp.volumesType)
		image := tempExp.Image()
		if image != nil {
			drive := &config.Drive{
				Image:        image,
				Maxsizebytes: defaults.DefaultVolumeSize,
			}
			ind := len(bundle.volumes)
			contentTree := exp.imageToContentTree(image, fmt.Sprintf("%s-%d", exp.appName, ind))
			bundle.contentTrees = append(bundle.contentTrees, contentTree)
			volume := exp.driveToVolume(drive, ind+1, contentTree)
			bundle.volumes = append(bundle.volumes, volume)
			bundle.appInstanceConfig.VolumeRefList = append(bundle.appInstanceConfig.VolumeRefList, &config.VolumeRef{Uuid: volume.Uuid})
		}
	}
	if exp.virtualizationMode == config.VmMode_PV {
		bundle.appInstanceConfig.Fixedresources.Rootdev = "/dev/xvda1"
		bundle.appInstanceConfig.Fixedresources.Bootloader = "/usr/lib/xen/boot/ovmf.bin"
	}
	bundle.appInstanceConfig.Interfaces = []*config.NetworkAdapter{}

	for k, ni := range netInstances {
		bundle.appInstanceConfig.Interfaces = append(bundle.appInstanceConfig.Interfaces, &config.NetworkAdapter{
			Name:      "default",
			NetworkId: ni.Uuidandversion.Uuid,
			Acls:      exp.getAcls(k),
		})
	}
	if exp.vncDisplay != 0 {
		bundle.appInstanceConfig.Fixedresources.EnableVnc = true
		bundle.appInstanceConfig.Fixedresources.VncDisplay = exp.vncDisplay
		bundle.appInstanceConfig.Fixedresources.VncPasswd = exp.vncPassword
	}
	var adapters []*config.Adapter
	for _, adapterName := range exp.appAdapters {
		adapters = append(adapters, &config.Adapter{
			Type: evecommon.PhyIoType_PhyIoUSB,
			Name: adapterName,
		})
	}
	bundle.appInstanceConfig.Adapters = adapters
	return bundle, nil
}

//Application expectation gets or creates Image definition, gets or create NetworkInstance definition,
//gets AppInstanceConfig and returns it or creates AppInstanceConfig, adds it into internal controller and returns it
func (exp *AppExpectation) Application() *config.AppInstanceConfig {
	image := exp.Image()
	networkInstances := exp.NetworkInstances()
	for _, appID := range exp.device.GetApplicationInstances() {
		app, err := exp.ctrl.GetApplicationInstanceConfig(appID)
		if err != nil {
			log.Fatalf("no app %s found in controller: %s", appID, err)
		}
		if exp.checkAppInstanceConfig(app) {
			return app
		}
	}
	bundle, err := exp.createAppInstanceConfig(image, networkInstances)
	if err != nil {
		log.Fatalf("cannot create app: %s", err)
	}
	if err = exp.ctrl.AddApplicationInstanceConfig(bundle.appInstanceConfig); err != nil {
		log.Fatalf("AddApplicationInstanceConfig: %s", err)
	}
	if exp.appLink == defaults.DefaultDummyExpect {
		log.Debug("skip modify of entities")
	} else {
		for _, el := range bundle.contentTrees {
			_ = exp.ctrl.AddContentTree(el)
			exp.device.SetContentTreeConfig(append(exp.device.GetContentTrees(), el.Uuid))
		}
		for _, el := range bundle.volumes {
			_ = exp.ctrl.AddVolume(el)
			exp.device.SetVolumeConfigs(append(exp.device.GetVolumes(), el.Uuid))
		}
	}
	return bundle.appInstanceConfig
}
