package lim

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// TODO: Update this to work SDN

// This test deploys the VM with image https://cloud-images.ubuntu.com/releases/groovy/release-20210108/ubuntu-20.10-server-cloudimg-ARCH.img
// with ARCH from config and vncDisplay into EVE
// waits for the RUNNING state and checks access to VNC and SSH console
// and removes app from EVE

var (
	timewait = flag.Duration("timewait", 20*time.Minute, "Timewait for items waiting")

	expand       = flag.Duration("expand", 10*time.Minute, "Expand timewait on success of step")
	name         = flag.String("name", "", "Name of app, random if empty")
	vncDisplay   = flag.Int("vncDisplay", 1, "VNC display number")
	vncPassword  = flag.String("vncPassword", "12345678", "Password for VNC")
	sshPort      = flag.Int("sshPort", 8027, "Port to publish ssh")
	cpus         = flag.Uint("cpus", 2, "Cpu number for app")
	memory       = flag.String("memory", "1G", "Memory for app")
	direct       = flag.Bool("direct", true, "Load image from url, not from eserver")
	metadata     = flag.String("metadata", "#cloud-config\npassword: passw0rd\nchpasswd: { expire: False }\nssh_pwauth: True\n", "Metadata to pass into VM")
	appLink      = flag.String("applink", "https://cloud-images.ubuntu.com/releases/groovy/release-20210108/ubuntu-20.10-server-cloudimg-%s.img", "Link to qcow2 image. You can pass %s for automatically set of arch (amd64/arm64)")
	doPanic      = flag.Bool("panic", false, "Test kernel panic")
	doLogger     = flag.Bool("logger", false, "Test logger print to console")
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

func setAppName() {
	if appName == "" { //if previous appName not defined
		if *name == "" {
			rand.Seed(time.Now().UnixNano())
			appName = namesgenerator.GetRandomName(0) //generates new name if no flag set
		} else {
			appName = *name
		}
	}
}

//getVNCPort calculate port for vnc
//for qemu it is forwarded
//for rpi it is direct
func getVNCPort(edgeNode *device.Ctx, vncDisplay int) int {
	if edgeNode.GetRemote() {
		return 5900 + vncDisplay
	}
	return 5910 + vncDisplay //forwarded by qemu ports
}

//checkAppRunning wait for info of ZInfoApp type with mention of deployed AppName and ZSwState_RUNNING state
func checkAppRunning(t *testing.T, appName string) projects.ProcInfoFunc {
	lastState := info.ZSwState_INVALID
	return func(msg *info.ZInfoMsg) error {
		if msg.Ztype == info.ZInfoTypes_ZiApp {
			if msg.GetAinfo().AppName == appName {
				if lastState != msg.GetAinfo().State {
					lastState = msg.GetAinfo().State
					t.Logf("\t\tstate: %s received in: %s", lastState, time.Now().Format(time.RFC3339Nano))
					if lastState == info.ZSwState_RUNNING {
						return fmt.Errorf("app RUNNING with name %s", appName)
					}
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
			eveIPCIDR, err := tc.GetState(edgeNode).LookUp("Dinfo.Network[0].IPAddrs[0]")
			if err != nil {
				return nil
			}
			ip := net.ParseIP(eveIPCIDR.String())
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

//checkVNCAccess try to access APP via VNC with timer
func checkVNCAccess() projects.ProcTimerFunc {
	return func() error {
		if externalIP == "" {
			return nil
		}
		desktopName, err := utils.GetDesktopName(fmt.Sprintf("%s:%d", externalIP, externalPort), *vncPassword)
		if err != nil {
			return nil
		}
		return fmt.Errorf("VNC DesktopName: %s. You can access it via VNC on %s:%d with password %s", desktopName, externalIP, externalPort, *vncPassword)
	}
}

//checkAppAbsent check if APP undefined in EVE
func checkAppAbsent(t *testing.T, appName string) projects.ProcInfoFunc {
	lastState := info.ZSwState_INVALID
	return func(msg *info.ZInfoMsg) error {
		if msg.Ztype == info.ZInfoTypes_ZiDevice {
			for _, app := range msg.GetDinfo().AppInstances {
				if app.Name == appName {
					return nil
				}
			}
			return fmt.Errorf("no app with %s found", appName)
		}
		if msg.Ztype == info.ZInfoTypes_ZiApp {
			if msg.GetAinfo().AppName == appName {
				if lastState != msg.GetAinfo().State {
					lastState = msg.GetAinfo().State
					t.Logf("\t\tstate: %s received in: %s", lastState, time.Now().Format(time.RFC3339Nano))
				}
			}
		}
		return nil
	}
}

//TestVNCVMStart gets EdgeNode and deploys app, defined in appLink with VncDisplay
//it generates random appName and adds processing functions
//it checks if app processed by EVE, app in RUNNING state, VNC and SSH of app is accessible
//it uses timewait for processing all events
func TestVNCVMStart(t *testing.T) {

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	var appLinkFunc = func(arch string) string {
		if strings.Count(*appLink, "%s") == 1 {
			return fmt.Sprintf(*appLink, arch)
		}
		return *appLink
	}

	setAppName()

	var opts []expect.ExpectationOption

	appMemoryParsed, err := humanize.ParseBytes(*memory)
	if err != nil {
		log.Fatal(err)
	}

	opts = append(opts, expect.WithResources(uint32(*cpus), uint32(appMemoryParsed/1000)))

	opts = append(opts, expect.WithMetadata(*metadata))

	opts = append(opts, expect.WithVnc(uint32(*vncDisplay)))

	opts = append(opts, expect.WithVncPassword(*vncPassword))

	opts = append(opts, expect.WithHTTPDirectLoad(*direct))

	if *sshPort != 0 {

		portPublish := []string{fmt.Sprintf("%d:%d", *sshPort, 22)}

		opts = append(opts, expect.WithPortsPublish(portPublish))

	}

	expectation := expect.AppExpectationFromURL(tc.GetController(), edgeNode, appLinkFunc(tc.GetController().GetVars().ZArch), appName, opts...)

	appInstanceConfig := expectation.Application()

	externalPort = getVNCPort(edgeNode, *vncDisplay)

	t.Log("Add app to list")

	edgeNode.SetApplicationInstanceConfig(append(edgeNode.GetApplicationInstances(), appInstanceConfig.Uuidandversion.Uuid))

	tc.ConfigSync(edgeNode)

	t.Log("Add processing of app running messages")

	tc.AddProcInfo(edgeNode, checkAppRunning(t, appName))

	appID, err := uuid.FromString(appInstanceConfig.Uuidandversion.Uuid)
	if err != nil {
		t.Fatal(err)
	}

	callback := func() {
		fmt.Printf("--- app %s logs ---\n", appInstanceConfig.Displayname)
		if err = tc.GetController().LogAppsChecker(edgeNode.GetID(), appID, nil, eapps.HandleFactory(types.OutputFormatJSON, false), eapps.LogExist, 0); err != nil {
			t.Fatalf("LogAppsChecker: %s", err)
		}
		fmt.Println("------")
	}

	tc.WaitForProcWithErrorCallback(int(timewait.Seconds()), callback)
}

func getAppInstanceConfig(edgeNode *device.Ctx, appName string) *config.AppInstanceConfig {

	var appInstanceConfig *config.AppInstanceConfig

	for _, id := range edgeNode.GetApplicationInstances() {
		appConfig, _ := tc.GetController().GetApplicationInstanceConfig(id)
		if appConfig.Displayname == appName {
			appInstanceConfig = appConfig
			break
		}
	}
	return appInstanceConfig
}

//TestAccess checks if VNC and SSH of app is accessible
func TestAccess(t *testing.T) {

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	setAppName()

	appInstanceConfig := getAppInstanceConfig(edgeNode, appName)

	if appInstanceConfig == nil {
		t.Fatalf("No app found with name %s", appName)
	}

	appID, err := uuid.FromString(appInstanceConfig.Uuidandversion.Uuid)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(utils.AddTimestamp("Add function to obtain EVE IP"))

	tc.AddProcTimer(edgeNode, getEVEIP(edgeNode))

	t.Log(utils.AddTimestamp("Add trying to access VNC of app"))

	externalPort = getVNCPort(edgeNode, *vncDisplay)

	tc.AddProcTimer(edgeNode, checkVNCAccess())

	t.Log(utils.AddTimestamp("Add trying to access SSH of app"))

	tc.AddProcTimer(edgeNode, projects.SendCommandSSH(&externalIP, sshPort, "ubuntu", "passw0rd", "exit", true))

	tc.ExpandOnSuccess(int(expand.Seconds()))

	callback := func() {
		fmt.Printf("--- app %s logs ---\n", appInstanceConfig.Displayname)
		if err = tc.GetController().LogAppsChecker(edgeNode.GetID(), appID, nil, eapps.HandleFactory(types.OutputFormatJSON, false), eapps.LogExist, 0); err != nil {
			t.Fatalf("LogAppsChecker: %s", err)
		}
		fmt.Println("------")
	}

	tc.WaitForProcWithErrorCallback(int(timewait.Seconds()), callback)

}

//TestAppLogs checks if logs of app is accessible
func TestAppLogs(t *testing.T) {

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	setAppName()

	appInstanceConfig := getAppInstanceConfig(edgeNode, appName)

	if appInstanceConfig == nil {
		t.Fatalf("No app found with name %s", appName)
	}

	appID, err := uuid.FromString(appInstanceConfig.Uuidandversion.Uuid)
	if err != nil {
		t.Fatal(err)
	}
	panicCmd := projects.SendCommandSSH(&externalIP, sshPort, "ubuntu", "passw0rd", "sudo su -c 'echo c > /proc/sysrq-trigger'", false)
	if *doLogger {
		fmt.Println("will wait for uptime logs in test")
		callback := func() {
			if *doPanic { //do panic after logger
				tc.AddProcTimer(edgeNode, panicCmd)
			}
		}
		tc.AddProcTimer(edgeNode, tc.CheckMessageInAppLog(edgeNode, appID, "uptime: ", callback))
		tc.AddProcTimer(edgeNode, projects.SendCommandSSH(&externalIP, sshPort, "ubuntu", "passw0rd", "sudo su -c 'echo uptime: `uptime`>/dev/console'", true)) //prints uptime to /dev/console
	}
	if *doPanic {
		fmt.Println("will fire kernel panic in test")
		tc.AddProcTimer(edgeNode, tc.CheckMessageInAppLog(edgeNode, appID, "Kernel panic"))
		if !*doLogger { //do panic immediately
			tc.AddProcTimer(edgeNode, panicCmd)
		}
	}

	t.Log(utils.AddTimestamp("Add function to obtain EVE IP"))

	tc.AddProcTimer(edgeNode, getEVEIP(edgeNode))

	tc.WaitForProc(int(timewait.Seconds()))
}

//TestVNCVMDelete gets EdgeNode and deletes previously deployed app, defined in appName or in name flag
//it checks if app absent in EVE
//it uses timewait for processing all events
func TestVNCVMDelete(t *testing.T) {

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	setAppName()

	t.Log(utils.AddTimestamp(fmt.Sprintf("Add waiting for app %s absent", appName)))

	tc.AddProcInfo(edgeNode, checkAppAbsent(t, appName))

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
