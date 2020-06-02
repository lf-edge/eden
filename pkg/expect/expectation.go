package expect

import (
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

type appType int

var dockerApp appType = 1

//appExpectation is description of app, expected to run on EVE
type appExpectation struct {
	ctrl       controller.Cloud
	appType    appType
	appUrl     string
	appVersion string
	appName    string
}

//AppExpectationFromUrl init appExpectation with defined appLink
func AppExpectationFromUrl(ctrl controller.Cloud, appLink string, podName string) (expectation *appExpectation) {
	expectation = &appExpectation{ctrl: ctrl}
	rand.Seed(time.Now().UnixNano())
	expectation.appName = namesgenerator.GetRandomName(0)
	if podName != "" {
		expectation.appName = podName
	}
	params := utils.GetParams(appLink, defaults.DefaultPodLinkPattern)
	if len(params) == 0 {
		log.Fatalf("fail to parse <docker>://<TAG>[:<VERSION>] from argument (%s)", appLink)
	}
	expectation.appType = 0
	expectation.appUrl = ""
	expectation.appVersion = ""
	ok := false
	appType := ""
	if appType, ok = params["TYPE"]; !ok || appType == "" {
		log.Fatalf("cannot parse appType (not [docker]): %s", appLink)
	}
	switch appType {
	case "docker":
		expectation.appType = dockerApp
	case "":
		expectation.appType = dockerApp
	default:
		log.Fatalf("format not supported %s", appType)
	}
	if expectation.appUrl, ok = params["TAG"]; !ok || expectation.appUrl == "" {
		log.Fatalf("cannot parse appTag: %s", appLink)
	}
	if expectation.appVersion, ok = params["VERSION"]; !ok || expectation.appVersion == "" {
		log.Debugf("cannot parse appVersion from %s will use latest", appLink)
		expectation.appVersion = "latest"
	}
	return
}
