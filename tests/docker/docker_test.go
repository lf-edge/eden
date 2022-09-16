package lim

import (
	"flag"
	"fmt"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"
)

// TODO: Update for SDN

// This test deploys the docker://nginx app into EVE with port forwarding 8028->80
// wait for the RUNNING state and checks access to HTTP endpoint
// and removes app from EVE
// you can replace defaults with flags
var (
	timewait     = flag.Duration("timewait", 10*time.Minute, "Timewait for items waiting")
	name         = flag.String("name", "", "Name of app, random if empty")
	externalPort = flag.Int("externalPort", 8028, "Port for access app from outside of EVE. Not publish if equals with 0.")
	internalPort = flag.Int("internalPort", 80, "Port for access app inside EVE")
	appLink      = flag.String("appLink", "docker://nginx", "Link to get app")
	cpus         = flag.Uint("cpus", 1, "Cpu number for app")
	memory       = flag.String("memory", "1G", "Memory for app")
	nohyper      = flag.Bool("nohyper", false, "Do not use a hypervisor")
	tc           *projects.TestContext
	externalIP   string
	portPublish  []string
	appName      string
)

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("Docker app deployment Test")

	tc = projects.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestDockerDeploy", time.Now())

	tc.InitProject(projectName)

	tc.AddEdgeNodesFromDescription()

	tc.StartTrackingState(false)

	res := m.Run()

	os.Exit(res)
}

//checkAppDeployStarted wait for info of ZInfoApp type with mention of deployed AppName
func checkAppDeployStarted(appName string) projects.ProcInfoFunc {
	return func(msg *info.ZInfoMsg) error {
		if msg.Ztype == info.ZInfoTypes_ZiApp {
			if msg.GetAinfo().AppName == appName {
				return fmt.Errorf("app found with name %s", appName)
			}
		}
		return nil
	}
}

//checkAppRunning wait for info of ZInfoApp type with mention of deployed AppName and ZSwState_RUNNING state
func checkAppRunning(appName string) projects.ProcInfoFunc {
	return func(msg *info.ZInfoMsg) error {
		if msg.Ztype == info.ZInfoTypes_ZiApp {
			if msg.GetAinfo().AppName == appName {
				if msg.GetAinfo().State == info.ZSwState_RUNNING {
					return fmt.Errorf("app RUNNING with name %s", appName)
				}
			}
		}
		return nil
	}
}

//getEVEIP wait for IPs of EVE and returns them
func getEVEIP(edgeNode *device.Ctx) projects.ProcTimerFunc {
	return func() error {
		if edgeNode.GetRemoteAddr() == "" { //no eve.remote-addr defined
			eveIP, err := tc.GetState(edgeNode).LookUp("Dinfo.Network[0].IPAddrs[0]")
			if err != nil {
				return nil
			}
			ip := net.ParseIP(eveIP.String())
			if ip == nil || ip.To4() == nil {
				return nil
			}
			externalIP = ip.To4().String()
		} else {
			externalIP = edgeNode.GetRemoteAddr()
		}
		return fmt.Errorf("external ip is: %s", externalIP)
	}
}

//checkAppAccess try to access APP with timer
func checkAppAccess() projects.ProcTimerFunc {
	return func() error {
		if externalIP == "" {
			return nil
		}
		res, err := utils.RequestHTTPWithTimeout(fmt.Sprintf("http://%s:%d", externalIP, *externalPort), time.Second)
		if err != nil {
			return nil
		}
		return fmt.Errorf(res)
	}
}

//checkAppAbsent check if APP undefined in EVE
func checkAppAbsent(appName string) projects.ProcInfoFunc {
	return func(msg *info.ZInfoMsg) error {
		if msg.Ztype == info.ZInfoTypes_ZiDevice {
			for _, app := range msg.GetDinfo().AppInstances {
				if app.Name == appName {
					return nil
				}
			}
			return fmt.Errorf("no app with %s found", appName)
		}
		return nil
	}
}

//TestDockerStart gets EdgeNode and deploys app, defined in appLink
//it generates random appName and adds processing functions
//it checks if app processed by EVE, app in RUNNING state, app is accessible by HTTP get
//it uses timewait for processing all events
func TestDockerStart(t *testing.T) {
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	if *name == "" {
		rand.Seed(time.Now().UnixNano())
		appName = namesgenerator.GetRandomName(0) //generates new name if no flag set
	} else {
		appName = *name
	}

	var opts []expect.ExpectationOption

	if *externalPort != 0 {

		portPublish = []string{fmt.Sprintf("%d:%d", *externalPort, *internalPort)}

		opts = append(opts, expect.WithPortsPublish(portPublish))

	}

	appMemoryParsed, err := humanize.ParseBytes(*memory)
	if err != nil {
		log.Fatal(err)
	}

	opts = append(opts, expect.WithResources(uint32(*cpus), uint32(appMemoryParsed/1000)))

	if *nohyper {
		t.Log(utils.AddTimestamp("will not use hypervisor"))
		opts = append(opts, expect.WithVirtualizationMode(config.VmMode_NOHYPER))
	}

	expectation := expect.AppExpectationFromURL(tc.GetController(), edgeNode, *appLink, appName, opts...)

	appInstanceConfig := expectation.Application()

	t.Log(utils.AddTimestamp("Add app to list"))

	edgeNode.SetApplicationInstanceConfig(append(edgeNode.GetApplicationInstances(), appInstanceConfig.Uuidandversion.Uuid))

	tc.ConfigSync(edgeNode)

	t.Log(utils.AddTimestamp("Add processing of app messages"))

	tc.AddProcInfo(edgeNode, checkAppDeployStarted(appName))

	t.Log(utils.AddTimestamp("Add processing of app running messages"))

	tc.AddProcInfo(edgeNode, checkAppRunning(appName))

	t.Log(utils.AddTimestamp("Add function to obtain EVE IP"))

	tc.AddProcTimer(edgeNode, getEVEIP(edgeNode))

	t.Log(utils.AddTimestamp("Add trying to access app via http"))

	if *externalPort != 0 {

		tc.AddProcTimer(edgeNode, checkAppAccess())

	}

	tc.WaitForProc(int(timewait.Seconds()))
}

//TestDockerDelete gets EdgeNode and deletes previously deployed app, defined in appName
//it checks if app absent in EVE
//it uses timewait for processing all events
func TestDockerDelete(t *testing.T) {

	if appName == "" { //if previous appName not defined
		if *name == "" {
			t.Fatal("No name of app, please set 'name' flag")
		} else {
			appName = *name
		}
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Log(utils.AddTimestamp(fmt.Sprintf("Add waiting for app %s absent", appName)))

	tc.AddProcInfo(edgeNode, checkAppAbsent(appName))

	for id, appUUID := range edgeNode.GetApplicationInstances() {
		appConfig, _ := tc.GetController().GetApplicationInstanceConfig(appUUID)
		if appConfig.Displayname == appName {
			volumeIDs := edgeNode.GetVolumes()
			utils.DelEleInSliceByFunction(&volumeIDs, func(i interface{}) bool {
				vol, err := tc.GetController().GetVolume(i.(string))
				if err != nil {
					log.Fatalf("no volume in cloud %s: %s", i.(string), err)
				}
				for _, volRef := range appConfig.VolumeRefList {
					if vol.Uuid == volRef.Uuid {
						return true
					}
				}
				return false
			})
			edgeNode.SetVolumeConfigs(volumeIDs)
			configs := edgeNode.GetApplicationInstances()
			t.Log(utils.AddTimestamp("Remove app from list"))
			utils.DelEleInSlice(&configs, id)
			edgeNode.SetApplicationInstanceConfig(configs)
			if err := tc.GetController().RemoveApplicationInstanceConfig(appUUID); err != nil {
				log.Fatal(err)
			}
			tc.ConfigSync(edgeNode)
			break
		}
	}

	tc.WaitForProc(int(timewait.Seconds()))
}
