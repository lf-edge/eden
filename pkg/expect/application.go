package expect

import (
	"fmt"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"strconv"
)

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

//createAppInstanceConfig creates AppInstanceConfig with provided img and netInstId
//  it uses published ports info from appExpectation to create ACE
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
	case httpApp, httpsApp, fileApp:
		return exp.createAppInstanceConfigVM(img, netInstId, id, acls), nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

//Application expectation gets or creates Image definition, gets or create NetworkInstance definition,
//gets AppInstanceConfig and returns it or creates AppInstanceConfig, adds it into internal controller and returns it
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
	if appInstanceConfig == nil { //if appInstanceConfig not exists, create it
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
