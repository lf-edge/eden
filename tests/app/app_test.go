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
	found    []string
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
				for _, app := range msg.GetDinfo().AppInstances {
					if _, inSlice := utils.FindEleInSlice(appNames, app.Name); inSlice {
						return nil
					}
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
					astate := app.State.String()
					if state == astate {
						if _, inFoundSlice := utils.FindEleInSlice(found, app.AppName); !inFoundSlice {
							found = append(found, app.AppName)
						}
					}
				}
			}
			if len(found) == len(appNames) {
				for _, appName := range appNames {
					if _, inFoundSlice := utils.FindEleInSlice(found, appName); inFoundSlice {
						out += fmt.Sprintf(
							"app %s state %s\n",
							appName, state)
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
		var state string
		state = args[0]
		fmt.Printf("apps: '%s' state: '%s' secs: %d\n",
			args[1:], state, secs)

		apps := args[1:]
		tc.AddProcInfo(edgeNode, checkApp(state, apps))

		tc.WaitForProc(secs)
	}
}
