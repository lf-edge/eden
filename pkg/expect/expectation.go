package expect

import (
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type appType int

var (
	dockerApp appType = 1
	httpApp   appType = 2
	httpsApp  appType = 3
)

//appExpectation is description of app, expected to run on EVE
type appExpectation struct {
	ctrl       controller.Cloud
	appType    appType
	appUrl     string
	appVersion string
	appName    string
	appLink    string
	ports      map[int]int
	cpu        uint32
	mem        uint32
	metadata   string
}

//AppExpectationFromUrl init appExpectation with defined appLink
func AppExpectationFromUrl(ctrl controller.Cloud, appLink string, podName string, portPublish []string, qemuPorts map[string]string, metadata string) (expectation *appExpectation) {
	expectation = &appExpectation{
		ctrl:     ctrl,
		ports:    make(map[int]int),
		appLink:  appLink,
		cpu:      defaults.DefaultAppCpu,
		mem:      defaults.DefaultAppMem,
		metadata: strings.Replace(metadata, `\n`, "\n", -1),
	}
	if portPublish != nil && len(portPublish) > 0 {
	exit:
		for _, el := range portPublish {
			splitted := strings.Split(el, ":")
			if len(splitted) != 2 {
				log.Fatalf("Cannot use %s in format EXTERNAL_PORT:INTERNAL_PORT", el)
			}
			extPort, err := strconv.Atoi(splitted[0])
			if err != nil {
				log.Fatalf("Cannot use %s in format EXTERNAL_PORT:INTERNAL_PORT: %s", el, err)
			}
			if extPort == 22 {
				log.Fatalf("Port 22 already in use")
			}
			intPort, err := strconv.Atoi(splitted[1])
			if err != nil {
				log.Fatalf("Cannot use %s in format EXTERNAL_PORT:INTERNAL_PORT: %s", el, err)
			}
			for _, qv := range qemuPorts {
				if qv == strconv.Itoa(extPort) {
					expectation.ports[extPort] = intPort
					break exit
				}
			}
			log.Fatalf("Cannot use external port %d. Not in Qemu %s", extPort, qemuPorts)
		}
	}
	//check used ports
	if len(expectation.ports) > 0 {
		for _, app := range ctrl.ListApplicationInstanceConfig() {
			for _, iface := range app.Interfaces {
				for _, acl := range iface.Acls {
					for _, match := range acl.Matches {
						for ip := range expectation.ports {
							if match.Type == "lport" && match.Value == strconv.Itoa(ip) {
								log.Fatalf("Port %d already in use", ip)
							}
						}
					}
				}
			}
		}
	}
	rand.Seed(time.Now().UnixNano())
	expectation.appName = namesgenerator.GetRandomName(0)
	if podName != "" {
		expectation.appName = podName
	}
	params := utils.GetParams(appLink, defaults.DefaultPodLinkPattern)
	if len(params) == 0 {
		log.Fatalf("fail to parse (docker|http(s))://(<TAG>[:<VERSION>] | <URL>) from argument (%s)", appLink)
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
	case "http":
		expectation.appType = httpApp
	case "https":
		expectation.appType = httpsApp
	case "":
		expectation.appType = dockerApp
	default:
		log.Fatalf("format not supported %s", appType)
	}
	if expectation.appUrl, ok = params["TAG"]; !ok || expectation.appUrl == "" {
		log.Fatalf("cannot parse appTag: %s", appLink)
	}
	if expectation.appVersion, ok = params["VERSION"]; expectation.appType == dockerApp && (!ok || expectation.appVersion == "") {
		log.Debugf("cannot parse appVersion from %s will use latest", appLink)
		expectation.appVersion = "latest"
	}
	return
}
