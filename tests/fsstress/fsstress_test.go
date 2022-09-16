package lim

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
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
	"github.com/spf13/viper"
)

// TODO: Update this test to work with SDN

// This test deploys the VM with image https://cloud-images.ubuntu.com/releases/impish/release-20220201/ubuntu-21.10-server-cloudimg-ARCH.img
// waits for the RUNNING state and checks access to SSH console
// and removes app from EVE

var (
	timewait   = flag.Duration("timewait", 20*time.Minute, "Timewait for items waiting")
	name       = flag.String("name", "", "Name of app, random if empty")
	sshPort    = flag.Int("sshPort", 8028, "Port to publish ssh")
	cpus       = flag.Uint("cpus", 2, "Cpu number for app")
	memory     = flag.String("memory", "2G", "Memory for app")
	diskSize   = flag.String("disk_size", "3G", "Disk size")
	scriptpath = flag.String("script_path", "", "Full path to the script that will be sent to the guest machine")
	direct     = flag.Bool("direct", true, "Load image from url, not from eserver")
	password   = flag.String("password", "passw0rd", "Password to use for ssh")
	appLink    = flag.String("applink", "https://cloud-images.ubuntu.com/releases/impish/release-20220201/ubuntu-21.10-server-cloudimg-%s.img", "Link to qcow2 image. You can pass %s for automatically set of arch (amd64/arm64)")
	tc         *projects.TestContext
	externalIP string
	appName    string
)

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("FSstress test")

	tc = projects.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestFSstress", time.Now())

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

//CheckTimeWorkOfTest checks how much time is left,
//and returns success if less than 1 minutes left
//also it checks existence of fsstress process on VM
//and in case of not existence or some issues with connection it fails test immediately
func CheckTimeWorkOfTest(t *testing.T, timeStart time.Time) projects.ProcTimerFunc {
	return func() error {
		df := time.Since(timeStart)
		if df >= *timewait-time.Minute {
			return fmt.Errorf("stress test is stable, end test")
		}
		if externalIP == "" {
			return nil
		}
		sendSSHCommand := projects.SendCommandSSH(&externalIP, sshPort, "ubuntu", *password, "pgrep fsstress", true)
		result := sendSSHCommand()
		if result == nil {
			t.Fatal(utils.AddTimestamp("cannot find fsstress process or connection problem"))
		}
		return nil
	}
}

//TestFSStressVMStart gets EdgeNode and deploys app,
//it generates random appName and adds processing functions
//it checks if app processed by EVE, app in RUNNING state SSH of app is accessible
//it uses timewait for processing all events
//
func TestFSStressVMStart(t *testing.T) {

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

	diskSizeParsed, err := humanize.ParseBytes(*diskSize)
	if err != nil {
		log.Fatal(err)
	}

	opts = append(opts, expect.WithDiskSize(int64(diskSizeParsed)))

	opts = append(opts, expect.WithResources(uint32(*cpus), uint32(appMemoryParsed/1000)))

	metadata := fmt.Sprintf("#cloud-config\npassword: %s\nchpasswd: { expire: False }\nssh_pwauth: True\n", *password)

	opts = append(opts, expect.WithMetadata(metadata))

	opts = append(opts, expect.WithHTTPDirectLoad(*direct))

	if *sshPort != 0 {

		portPublish := []string{fmt.Sprintf("%d:%d", *sshPort, 22)}

		opts = append(opts, expect.WithPortsPublish(portPublish))

	}

	expectation := expect.AppExpectationFromURL(tc.GetController(), edgeNode, appLinkFunc(tc.GetController().GetVars().ZArch), appName, opts...)

	appInstanceConfig := expectation.Application()

	t.Log("Add app to list")

	edgeNode.SetApplicationInstanceConfig(append(edgeNode.GetApplicationInstances(), appInstanceConfig.Uuidandversion.Uuid))

	tc.ConfigSync(edgeNode)

	t.Log("Add processing of app running messages")

	tc.AddProcInfo(edgeNode, checkAppRunning(appName))

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

//TestAccess checks if SSH of app is accessible
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

	t.Log(utils.AddTimestamp("Add trying to access SSH of app"))

	tc.AddProcTimer(edgeNode, projects.SendCommandSSH(&externalIP, sshPort, "ubuntu", *password, "exit", true))

	callback := func() {
		fmt.Printf("--- app %s logs ---\n", appInstanceConfig.Displayname)
		if err = tc.GetController().LogAppsChecker(edgeNode.GetID(), appID, nil, eapps.HandleFactory(types.OutputFormatJSON, false), eapps.LogExist, 0); err != nil {
			t.Fatalf("LogAppsChecker: %s", err)
		}
		fmt.Println("------")
	}

	tc.WaitForProcWithErrorCallback(int(timewait.Seconds()), callback)
}

//TestRunStress run fsstress test on guest vm
func TestRunStress(t *testing.T) {
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))
	timeTestStart := time.Now()
	setAppName()

	appInstanceConfig := getAppInstanceConfig(edgeNode, appName)

	if appInstanceConfig == nil {
		t.Fatalf("No app found with name %s", appName)
	}

	appID, err := uuid.FromString(appInstanceConfig.Uuidandversion.Uuid)
	if err != nil {
		t.Fatal(err)
	}

	pathScript := *scriptpath
	if *scriptpath == "" {
		pathScript = filepath.Join(viper.GetString("eden.tests"), "/fsstress/testdata/run-script.sh")
	}

	result := getEVEIP(edgeNode)()
	if result == nil {
		t.Fatal(utils.AddTimestamp("Cannot get EVE IP"))
	}

	t.Log(utils.AddTimestamp("Send script on guest VM"))
	result = projects.SendFileSCP(&externalIP, sshPort, "ubuntu", *password, pathScript, "/home/ubuntu/run-script.sh")()
	if result == nil {
		t.Fatal(utils.AddTimestamp("Error in scp"))
	}

	t.Log(utils.AddTimestamp("Add options for running"))
	result = projects.SendCommandSSH(&externalIP, sshPort, "ubuntu", *password, "chmod +x ~/run-script.sh", true)()
	if result == nil {
		t.Fatal(utils.AddTimestamp("Error in chmod"))
	}

	t.Log(utils.AddTimestamp("Run script on guest VM"))
	result = projects.SendCommandSSH(&externalIP, sshPort, "ubuntu", *password, "~/run-script.sh", true)()
	if result == nil {
		t.Fatal(utils.AddTimestamp("Error in running script"))
	}

	tc.AddProcTimer(edgeNode, CheckTimeWorkOfTest(t, timeTestStart))

	callback := func() {
		fmt.Printf("--- app %s logs ---\n", appInstanceConfig.Displayname)
		if err = tc.GetController().LogAppsChecker(edgeNode.GetID(), appID, nil, eapps.HandleFactory(types.OutputFormatJSON, false), eapps.LogExist, 0); err != nil {
			t.Fatalf("LogAppsChecker: %s", err)
		}
		fmt.Println("------")
	}

	tc.WaitForProcWithErrorCallback(int(timewait.Seconds()), callback)
}

//TestFSStressVMDelete gets EdgeNode and deletes previously deployed app, defined in appName or in name flag
//it checks if app absent in EVE
//it uses timewait for processing all events
func TestFSStressVMDelete(t *testing.T) {

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	setAppName()

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
