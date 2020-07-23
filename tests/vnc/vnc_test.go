package lim

import (
	"flag"
	"fmt"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// This test deploys the VM with image http://cdimage.debian.org/cdimage/openstack/current/debian-10.4.3-20200610-openstack-ARCH.qcow2
// with ARCH from config and vncDisplay into EVE
// waits for the RUNNING state and checks access to VNC console
// and removes app from EVE

var (
	timewait     = flag.Int("timewait", 300, "Timewait for items waiting in seconds")
	name         = flag.String("name", "", "Name of app, random if empty")
	vncDisplay   = flag.Int("vncDisplay", 1, "VNC display number")
	metadata     = flag.String("metadata", "#cloud-config\npassword: passw0rd\nchpasswd: { expire: False }\nssh_pwauth: True\n", "Metadata to pass into VM")
	appLink      = flag.String("applink", "http://cdimage.debian.org/cdimage/openstack/current/debian-10.4.3-20200610-openstack-%s.qcow2", "Link to qcow2 image. You can pass %s for automatically set of arch (amd64/arm64)")
	tc           *projects.TestContext
	externalIP   string
	externalPort int
	appName      string
)

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("VNC access to app Test")

	tc = projects.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestVNCAccess", time.Now())

	tc.InitProject(projectName)

	tc.AddEdgeNodesFromDescription()

	tc.StartTrackingState(false)

	res := m.Run()

	os.Exit(res)
}

//getVNCPort calculate port for vnc
//for qemu it is forwarded
//for rpi it is direct
func getVNCPort(edgeNode *device.Ctx, vncDisplay int) int {
	if edgeNode.GetDevModel() == defaults.DefaultRPIModel {
		return 5900 + vncDisplay
	} else {
		return 5910 + vncDisplay //forwarded by qemu ports
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
			if eveIPCIDR, err := tc.GetState(edgeNode).LookUp("Dinfo.Network[0].IPAddrs[0]"); err != nil {
				return nil
			} else {
				if ip, _, err := net.ParseCIDR(eveIPCIDR.String()); err != nil {
					return nil
				} else {
					externalIP = ip.To4().String()
					return fmt.Errorf("external ip is: %s", externalIP)
				}
			}
		} else {
			externalIP = edgeNode.GetRemoteAddr()
			return fmt.Errorf("external ip is: %s", externalIP)
		}
	}
}

//checkAppAccess try to access APP via VNC with timer
func checkAppAccess() projects.ProcTimerFunc {
	return func() error {
		if externalIP == "" {
			return nil
		}
		desktopName, err := utils.GetDesktopName(fmt.Sprintf("%s:%d", externalIP, externalPort), "")
		if err != nil {
			return nil
		}
		return fmt.Errorf("VNC DesktopName: %s. You can access it via VNC on %s:%d", desktopName, externalIP, externalPort)
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

//TestVNCVMStart gets EdgeNode and deploys app, defined in appLink with VncDisplay
//it generates random appName and adds processing functions
//it checks if app processed by EVE, app in RUNNING state, VNC of app is accessible
//it uses timewait for processing all events
func TestVNCVMStart(t *testing.T) {

	if *name == "" {
		rand.Seed(time.Now().UnixNano())
		appName = namesgenerator.GetRandomName(0) //generates new name if no flag set
	} else {
		appName = *name
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	var appLinkFunc = func(arch string) string {
		if strings.Count(*appLink, "%s") == 1 {
			return fmt.Sprintf(*appLink, arch)
		}
		return *appLink
	}

	expectation := expect.AppExpectationFromUrl(tc.GetController(), appLinkFunc(tc.GetController().GetVars().ZArch), appName, expect.WithMetadata(*metadata), expect.WithVnc(uint32(*vncDisplay)))

	appInstanceConfig := expectation.Application()

	externalPort = getVNCPort(edgeNode, *vncDisplay)

	t.Log("Add app to list")

	edgeNode.SetApplicationInstanceConfig(append(edgeNode.GetApplicationInstances(), appInstanceConfig.Uuidandversion.Uuid))

	tc.ConfigSync(edgeNode)

	t.Log("Add processing of app running messages")

	tc.AddProcInfo(edgeNode, checkAppRunning(appName))

	t.Log("Add function to obtain EVE IP")

	tc.AddProcTimer(edgeNode, getEVEIP(edgeNode))

	t.Log("Add trying to access VNC of app")

	tc.AddProcTimer(edgeNode, checkAppAccess())

	tc.WaitForProc(*timewait)
}

//TestVNCVMDelete gets EdgeNode and deletes previously deployed app, defined in appName or in name flag
//it checks if app absent in EVE
//it uses timewait for processing all events
func TestVNCVMDelete(t *testing.T) {

	if appName == "" { //if previous appName not defined
		if *name == "" {
			rand.Seed(time.Now().UnixNano())
			appName = namesgenerator.GetRandomName(0) //generates new name if no flag set
		} else {
			appName = *name
		}
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Logf("Add waiting for app %s absent", appName)

	tc.AddProcInfo(edgeNode, checkAppAbsent(appName))

	for id, appUUID := range edgeNode.GetApplicationInstances() {
		appConfig, _ := tc.GetController().GetApplicationInstanceConfig(appUUID)
		if appConfig.Displayname == appName {
			configs := edgeNode.GetApplicationInstances()
			t.Log("Remove app from list")
			utils.DelEleInSlice(&configs, id)
			edgeNode.SetApplicationInstanceConfig(configs)
			if err := tc.GetController().RemoveApplicationInstanceConfig(appUUID); err != nil {
				log.Fatal(err)
			}
			tc.ConfigSync(edgeNode)
			break
		}
	}

	tc.WaitForProc(*timewait)
}
