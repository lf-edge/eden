package expect

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"strconv"
)

func (exp *appExpectation) checkAppInstanceConfig(app *config.AppInstanceConfig) bool {
	if app == nil {
		return false
	}
	if app.Displayname == exp.appName {
		return true
	}
	return false
}

func (exp *appExpectation) createAppInstanceConfig(img *config.Image, netInstId string) (*config.AppInstanceConfig, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	acls := []*config.ACE{{
		Matches: []*config.ACEMatch{{
			Type: "host",
		}},
		Id: 1,
	}}
	var aclID int32 = 2
	if exp.ports != nil {
		for po, pi := range exp.ports {
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
	switch exp.appType {
	case dockerApp:
		return exp.createAppInstanceConfigDocker(img, netInstId, id, acls), nil
	case httpApp, httpsApp:
		return exp.createAppInstanceConfigVM(img, netInstId, id, acls), nil
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
		if exp.checkAppInstanceConfig(app) {
			appInstanceConfig = app
			break
		}
	}
	if appInstanceConfig == nil {
		if appInstanceConfig, err = exp.createAppInstanceConfig(image, networkInstance.Uuidandversion.Uuid); err != nil {
			log.Fatalf("cannot create app: %s", err)
		}
		if err = exp.ctrl.AddApplicationInstanceConfig(appInstanceConfig); err != nil {
			log.Fatalf("AddApplicationInstanceConfig: %s", err)
		}
		log.Infof("new app created %s", appInstanceConfig.Uuidandversion.Uuid)
	}
	return
}
