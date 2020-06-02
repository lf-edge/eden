package expect

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

func checkAppInstanceConfig(app *config.AppInstanceConfig, appName string, appType appType, appUrl string, appVersion string) bool {
	if app == nil {
		return false
	}
	if app.Displayname == appName {
		return true
	}
	return false
}

func createAppInstanceConfig(img *config.Image, appName string, netInstId string, appType appType, appUrl string, appVersion string) (*config.AppInstanceConfig, error) {
	var app *config.AppInstanceConfig
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	switch appType {
	case dockerApp:
		app = &config.AppInstanceConfig{
			Uuidandversion: &config.UUIDandVersion{
				Uuid:    id.String(),
				Version: "1",
			},
			Fixedresources: &config.VmConfig{
				Memory:     1024000,
				Maxmem:     1024000,
				Vcpus:      1,
				Rootdev:    "/dev/xvda1",
				Bootloader: "/usr/bin/pygrub",
			},
			Drives: []*config.Drive{{
				Image: img,
			}},
			Activate:    true,
			Displayname: appName,
			Interfaces: []*config.NetworkAdapter{{
				Name:      "default",
				NetworkId: netInstId,
				Acls: []*config.ACE{{
					Matches: []*config.ACEMatch{{
						Type: "host",
					}},
					Id: 1,
				}},
			}},
		}
		return app, nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

//Application expects application in controller
func (exp *appExpectation) Application() (appInstanceConfig *config.AppInstanceConfig) {
	var err error
	image := exp.Image()
	networkInstance := exp.NetworkInstance()
	for _, app := range exp.ctrl.ListApplicationInstanceConfig() {
		if checkAppInstanceConfig(app, exp.appName, exp.appType, exp.appUrl, exp.appVersion) {
			appInstanceConfig = app
			break
		}
	}
	if appInstanceConfig == nil {
		if appInstanceConfig, err = createAppInstanceConfig(image, exp.appName, networkInstance.Uuidandversion.Uuid, exp.appType, exp.appUrl, exp.appVersion); err != nil {
			log.Fatalf("cannot create app: %s", err)
		}
		if err = exp.ctrl.AddApplicationInstanceConfig(appInstanceConfig); err != nil {
			log.Fatalf("AddApplicationInstanceConfig: %s", err)
		}
		log.Infof("new app created %s", appInstanceConfig.Uuidandversion.Uuid)
	}
	return
}
