package lim

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
)

// This test wait for the app's state with a timewait.
var (
	timewait = flag.Duration("timewait", time.Minute, "Timewait for items waiting")
	tc       *projects.TestContext
	found    map[string]string
)

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("Docker app's state test")

	tc = projects.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestAppState", time.Now())

	tc.InitProject(projectName)

	tc.AddEdgeNodesFromDescription()

	tc.StartTrackingState(false)

	res := m.Run()

	os.Exit(res)
}

//checkApp wait for info of ZInfoApp type with state
func checkApp(state string, appNames []string) projects.ProcInfoFunc {
	return func(msg *info.ZInfoMsg) error {
		out := "\n"
		if state == "-" {
			if msg.Ztype == info.ZInfoTypes_ZiDevice {
				foundAny := false
				for _, app := range msg.GetDinfo().AppInstances {
					if _, inSlice := utils.FindEleInSlice(appNames, app.Name); inSlice {
						found[app.Name] = "EXISTS"
						foundAny = true
					}
				}
				if foundAny {
					return nil
				}
				for _, appName := range appNames {
					out += fmt.Sprintf(
						"no app with %s found\n",
						appName)
				}
				return fmt.Errorf(out)
			}
		} else {
			if msg.Ztype == info.ZInfoTypes_ZiApp {
				app := msg.GetAinfo()
				if _, inSlice := utils.FindEleInSlice(appNames, app.AppName); inSlice {
					found[app.AppName] = app.State.String()
				}
			}
			if len(found) == len(appNames) {
				for _, appName := range appNames {
					if astate, inFoundSlice := found[appName]; inFoundSlice && astate == state {
						out += fmt.Sprintf(
							"app %s state %s\n",
							appName, state)
					} else {
						return nil
					}
				}
				return fmt.Errorf(out)
			}
			return nil
		}
		return nil
	}
}

//TestAppStatus wait for application reaching the selected state
//with a timewait
func TestAppStatus(t *testing.T) {
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	args := flag.Args()
	if len(args) == 0 {
		t.Fatalf("Usage: %s [options] state app_name...\n", os.Args[0])
	} else {
		secs := int(timewait.Seconds())
		state := args[0]
		fmt.Printf("apps: '%s' state: '%s' secs: %d\n",
			args[1:], state, secs)

		apps := args[1:]
		found = make(map[string]string)
		for _, el := range apps {
			found[el] = "no info from controller"
		}

		tc.AddProcInfo(edgeNode, checkApp(state, apps))

		callback := func() {
			t.Errorf("ASSERTION FAILED: expected apps %s in %s state", apps, state)
			for k, v := range found {
				t.Errorf("\tactual app %s: %s", k, v)
			}
		}

		tc.WaitForProcWithErrorCallback(secs, callback)
	}
}
