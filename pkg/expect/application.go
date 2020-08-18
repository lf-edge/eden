package expect

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"strconv"
)

//appBundle type for aggregate objects, needed for application
type appBundle struct {
	appInstanceConfig *config.AppInstanceConfig
	contentTrees      []*config.ContentTree
	volumes           []*config.Volume
}

//checkAppInstanceConfig checks if provided app match expectation
func (exp *appExpectation) checkAppInstanceConfig(app *config.AppInstanceConfig) bool {
	if app == nil {
		return false
	}
	if app.Displayname == exp.appName {
		return true
	}
	return false
}

//createAppInstanceConfig creates AppInstanceConfig with provided img and netInstances
//  it uses published ports info from appExpectation to create ACE
func (exp *appExpectation) createAppInstanceConfig(img *config.Image, netInstances map[*netInstanceExpectation]*config.NetworkInstanceConfig) (*appBundle, error) {
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
	bundle.appInstanceConfig.Interfaces = []*config.NetworkAdapter{}

	for k, ni := range netInstances {
		var acls []*config.ACE
		if exp.onlyHostAcl {
			acls = append(acls, &config.ACE{
				Matches: []*config.ACEMatch{{
					Type: "host",
				}},
				Id: 1,
			})
		} else {
			acls = append(acls, &config.ACE{
				Matches: []*config.ACEMatch{{
					Type:  "ip",
					Value: "0.0.0.0/0",
				}},
				Id: 1,
			})
		}
		var aclID int32 = 2
		if k.ports != nil {
			for po, pi := range k.ports {
				acls = append(acls, &config.ACE{
					Id: aclID,
					Matches: []*config.ACEMatch{{
						Type:  "protocol",
						Value: "tcp",
					}, {
						Type:  "lport",
						Value: strconv.Itoa(po),
					}},
					Actions: []*config.ACEAction{{
						Portmap: true,
						AppPort: uint32(pi),
					}},
					Dir: config.ACEDirection_BOTH})
				aclID++
			}
		}
		bundle.appInstanceConfig.Interfaces = append(bundle.appInstanceConfig.Interfaces, &config.NetworkAdapter{
			Name:      "default",
			NetworkId: ni.Uuidandversion.Uuid,
			Acls:      acls,
		})
	}
	if exp.vncDisplay != 0 {
		bundle.appInstanceConfig.Fixedresources.EnableVnc = true
		bundle.appInstanceConfig.Fixedresources.VncDisplay = exp.vncDisplay
		bundle.appInstanceConfig.Fixedresources.VncPasswd = exp.vncPassword
	}
	return bundle, nil
}

//Application expectation gets or creates Image definition, gets or create NetworkInstance definition,
//gets AppInstanceConfig and returns it or creates AppInstanceConfig, adds it into internal controller and returns it
func (exp *appExpectation) Application() *config.AppInstanceConfig {
	image := exp.Image()
	networkInstances := exp.NetworkInstances()
	for _, app := range exp.ctrl.ListApplicationInstanceConfig() {
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
	for _, el := range bundle.contentTrees {
		_ = exp.ctrl.AddContentTree(el)
		exp.device.SetContentTreeConfig(append(exp.device.GetContentTrees(), el.Uuid))
	}
	for _, el := range bundle.volumes {
		_ = exp.ctrl.AddVolume(el)
		exp.device.SetVolumeConfigs(append(exp.device.GetVolumes(), el.Uuid))
	}
	log.Infof("new app created %s", bundle.appInstanceConfig.Uuidandversion.Uuid)
	return bundle.appInstanceConfig
}
