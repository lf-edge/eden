package expect

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/evecommon"
	log "github.com/sirupsen/logrus"
)

//appType is type of app according to provided appLink
type appType int

var (
	dockerApp    appType = 1 //for docker application
	httpApp      appType = 2 //for application with image from http link
	httpsApp     appType = 3 //for application with image from https link
	fileApp      appType = 4 //for application with image from file path
	directoryApp appType = 5 //for application with files from directory
)

//ACE is an access control entry (a single entry of ACL).
type ACE struct {
	Endpoint string
	Drop     bool
}

//ACLs is a map of access control lists assigned to network instances.
type ACLs map[string][]ACE // network instance -> ACL (list of ACEs)

//AppExpectation is description of app, expected to run on EVE
type AppExpectation struct {
	ctrl        controller.Cloud
	appType     appType
	appURL      string
	appVersion  string
	appName     string
	appLink     string
	appAdapters []string
	imageFormat string
	cpu         uint32
	mem         uint32
	metadata    string

	baseOSVersion string

	vncDisplay  uint32
	vncPassword string

	netInstances []*NetInstanceExpectation

	diskSize int64

	uplinkAdapter *config.Adapter

	virtualizationMode config.VmMode

	device *device.Ctx

	volumesType VolumeType
	volumeSize  int64

	registry string

	oldAppName string

	httpDirectLoad bool // use eserver for SHA calculation only
	sftpLoad       bool

	disks []string
	acl   ACLs
	vlans map[string]int // networkInstanceName -> VID

	openStackMetadata bool
	profiles          []string
}

//use provided appLink to try predict format of volume
func tryPredictAppType(appLink string) string {
	if len(strings.Split(appLink, "://")) < 2 {
		fi, err := os.Stat(appLink)
		if err != nil {
			log.Warnf("tryPredictAppType: %v", err)
		} else {
			switch mode := fi.Mode(); {
			case mode.IsDir():
				//appLink is directory
				return fmt.Sprintf("directory://%s", appLink)
			case mode.IsRegular():
				//appLink is file
				return fmt.Sprintf("file://%s", appLink)
			}
		}
	}
	return appLink
}

//AppExpectationFromURL init AppExpectation with defined:
//   appLink - docker url to pull or link to qcow2 image or path to qcow2 image file
//   podName - name of app
//   device - device to set updates in volumes and content trees
//   opts can be used to modify parameters of expectation
func AppExpectationFromURL(ctrl controller.Cloud, device *device.Ctx, appLink string, podName string, opts ...ExpectationOption) (expectation *AppExpectation) {
	var adapter = &config.Adapter{
		Name: "eth0",
		Type: evecommon.PhyIoType_PhyIoNetEth,
	}
	if ctrl.GetVars().EveSSID != "" {
		adapter = &config.Adapter{
			Name: "wlan0",
			Type: evecommon.PhyIoType_PhyIoNetWLAN,
		}
	}
	var qemuPorts map[string]string
	if ctrl.GetVars().EveQemuPorts != nil {
		qemuPorts = ctrl.GetVars().EveQemuPorts
	}
	expectation = &AppExpectation{
		ctrl:    ctrl,
		appLink: appLink,
		cpu:     defaults.DefaultAppCPU,
		mem:     defaults.DefaultAppMem,

		uplinkAdapter: adapter,
		device:        device,
		volumesType:   VolumeQcow2,
	}
	switch expectation.ctrl.GetVars().ZArch {
	case "amd64":
		expectation.virtualizationMode = config.VmMode_HVM
	case "arm64":
		expectation.virtualizationMode = config.VmMode_PV
	default:
		log.Fatalf("Unexpected arch %s", expectation.ctrl.GetVars().ZArch)
	}
	for _, opt := range opts {
		opt(expectation)
	}
	if expectation.netInstances == nil {
		expectation.netInstances = []*NetInstanceExpectation{{
			subnet: defaults.DefaultAppSubnet,
		}}
	}
	//check portPublish variable
	for _, ni := range expectation.netInstances {
	exit:
		for _, el := range ni.portsReceived {
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
			if len(qemuPorts) > 0 { //not empty forwarding rules, need to check for existing
				for _, qv := range qemuPorts {
					portNum, err := strconv.Atoi(qv)
					if err != nil {
						log.Fatalf("Port map port %s could not be converted to Integer", qv)
					}
					if portNum == extPort || (portNum+defaults.DefaultPortMapOffset) == extPort {
						ni.ports[extPort] = intPort
						continue exit
					}
				}
			}
			ni.ports[extPort] = intPort
		}
	}
	//check used ports
	for _, ni := range expectation.netInstances {
		if len(ni.ports) > 0 {
			for _, appID := range device.GetApplicationInstances() {
				app, err := ctrl.GetApplicationInstanceConfig(appID)
				if err != nil {
					log.Fatalf("app %s not found: %s", appID, err)
				}
				if app.Displayname == expectation.oldAppName {
					//if we try to modify the app, we skip this check
					continue
				}
				for _, iface := range app.Interfaces {
					for _, acl := range iface.Acls {
						for _, match := range acl.Matches {
							for ip := range ni.ports {
								if match.Type == "lport" && match.Value == strconv.Itoa(ip) {
									log.Fatalf("Port %d already in use", ip)
								}
							}
						}
					}
				}
			}
		}
	}
	//generate random name
	rand.Seed(time.Now().UnixNano())
	expectation.appName = namesgenerator.GetRandomName(0)
	if podName != "" {
		//set defined name if provided
		expectation.appName = podName
	}
	appLink = tryPredictAppType(appLink)
	//parse provided appLink to obtain params
	params := utils.GetParams(appLink, defaults.DefaultPodLinkPattern)
	if len(params) == 0 {
		log.Fatalf("fail to parse (oci|docker|http(s)|file|directory)://(<TAG>[:<VERSION>] | <URL> | <PATH>) from argument (%s)", appLink)
	}
	expectation.appType = 0
	expectation.appURL = ""
	expectation.appVersion = ""
	ok := false
	appType := ""
	if appType, ok = params["TYPE"]; !ok || appType == "" {
		log.Fatalf("cannot parse appType (not [docker]): %s", appLink)
	}
	switch appType {
	case "docker", "oci":
		expectation.appType = dockerApp
	case "http":
		expectation.appType = httpApp
	case "https":
		expectation.appType = httpsApp
	case "file":
		expectation.appType = fileApp
	case "directory":
		expectation.appType = directoryApp
	case "":
		expectation.appType = dockerApp
	default:
		log.Fatalf("format not supported %s", appType)
	}
	if expectation.appURL, ok = params["TAG"]; !ok || expectation.appURL == "" {
		log.Fatalf("cannot parse appTag: %s", appLink)
	}
	if expectation.appVersion, ok = params["VERSION"]; expectation.appType == dockerApp && (!ok || expectation.appVersion == "") {
		log.Debugf("cannot parse appVersion from %s will use latest", appLink)
		expectation.appVersion = "latest"
	}
	return
}
