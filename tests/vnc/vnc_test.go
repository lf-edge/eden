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
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// This test deploys the VM with image https://cloud-images.ubuntu.com/releases/groovy/release-20201022.1/ubuntu-20.10-server-cloudimg-ARCH.img
// with ARCH from config and vncDisplay into EVE
// waits for the RUNNING state and checks access to VNC and SSH console
// and removes app from EVE

var (
	timewait     = flag.Int("timewait", 900, "Timewait for items waiting in seconds")
	expand       = flag.Int("expand", 400, "Expand timewait on success of step in seconds")
	name         = flag.String("name", "", "Name of app, random if empty")
	vncDisplay   = flag.Int("vncDisplay", 1, "VNC display number")
	vncPassword  = flag.String("vncPassword", "12345678", "Password for VNC")
	sshPort      = flag.Int("sshPort", 8027, "Port to publish ssh")
	cpus         = flag.Uint("cpus", 1, "Cpu number for app")
	memory       = flag.String("memory", "1G", "Memory for app")
	metadata     = flag.String("metadata", "#cloud-config\npassword: passw0rd\nchpasswd: { expire: False }\nssh_pwauth: True\n", "Metadata to pass into VM")
	appLink      = flag.String("applink", "https://cloud-images.ubuntu.com/releases/groovy/release-20201022.1/ubuntu-20.10-server-cloudimg-%s.img", "Link to qcow2 image. You can pass %s for automatically set of arch (amd64/arm64)")
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
	if edgeNode.GetDevModel() == defaults.DefaultRPIModel ||
		edgeNode.GetDevModel() == defaults.DefaultGCPModel ||
		edgeNode.GetDevModel() == defaults.DefaultGeneralModel {
		return 5900 + vncDisplay
	}
	return 5910 + vncDisplay //forwarded by qemu ports
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

//checkSSHAccess try to access SSH with timer
func checkSSHAccess() projects.ProcTimerFunc {
	return func() error {
		if externalIP == "" {
			return nil
		}
		config := &ssh.ClientConfig{
			User: "ubuntu",
			Auth: []ssh.AuthMethod{
				ssh.Password("passw0rd"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         defaults.DefaultRepeatTimeout,
		}
		_, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", externalIP, *sshPort), config)
		if err != nil {
			return nil
		}
		return fmt.Errorf("SSH success. You can access it via SSH on %s:%d", externalIP, *sshPort)
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
//it checks if app processed by EVE, app in RUNNING state, VNC and SSH of app is accessible
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

	var opts []expect.ExpectationOption

	appMemoryParsed, err := humanize.ParseBytes(*memory)
	if err != nil {
		log.Fatal(err)
	}

	opts = append(opts, expect.WithResources(uint32(*cpus), uint32(appMemoryParsed/1000)))

	opts = append(opts, expect.WithMetadata(*metadata))

	opts = append(opts, expect.WithVnc(uint32(*vncDisplay)))

	opts = append(opts, expect.WithVncPassword(*vncPassword))

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

	tc.AddProcInfo(edgeNode, checkAppRunning(appName))

	t.Log("Add function to obtain EVE IP")

	tc.AddProcTimer(edgeNode, getEVEIP(edgeNode))

	t.Log("Add trying to access VNC of app")

	tc.AddProcTimer(edgeNode, checkVNCAccess())

	if *sshPort != 0 {

		t.Log("Add trying to access SSH of app")

		tc.AddProcTimer(edgeNode, checkSSHAccess())

	}

	tc.ExpandOnSuccess(*expand)

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
