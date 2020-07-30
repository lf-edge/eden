package lim

import (
	"flag"
	"fmt"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eve/api/go/info"
	"os"
	"testing"
	"time"
)

// This test wait for the app's state with a timewait.
var (
	timewait    = flag.Duration("timewait", time.Minute, "Timewait for items waiting")
	tc          *projects.TestContext
	externalIP  string
	portPublish []string
	//appName     string
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
func checkApp(appName string, state string) projects.ProcInfoFunc {
	return func(msg *info.ZInfoMsg) error {
		if state == "" {
			if msg.Ztype == info.ZInfoTypes_ZiDevice {
				for _, app := range msg.GetDinfo().AppInstances {
					if app.Name == appName {
						return nil
					}
				}
				return fmt.Errorf("no app with %s found", appName)
			}
		} else {
			if msg.Ztype == info.ZInfoTypes_ZiApp {
				if msg.GetAinfo().AppName == appName {
					astate := msg.GetAinfo().State.String()
					if state == astate {
						return fmt.Errorf("app %s in state %s", appName, state)
					}

				}
			}
		}
		return nil
	}
}

//TestAppSatus wait for application reaching the selected state
//with a timewait
func TestAppSatus(t *testing.T) {
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	args := flag.Args()
	if len(args) == 0 {
		t.Fatalf("Usage: %s [options] name [state]\n", os.Args[0])
	} else {
		secs := int(timewait.Seconds())
		var state string
		if len(args) > 1 {
			state = args[1]
		} else {
			state = ""
		}
		fmt.Printf("appName: '%s' state: '%s' secs: %d\n",
			args[0], state, secs)

		tc.AddProcInfo(edgeNode, checkApp(args[0], state))

		tc.WaitForProc(secs)
	}
}
