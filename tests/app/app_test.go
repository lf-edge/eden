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
	states   map[string][]string
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

// checkNewLastState returns true if provided state not equals with last
func checkNewLastState(appName, state string) bool {
	appStates, ok := states[appName]
	if ok {
		lastState := appStates[len(appStates)-1]
		if lastState != state {
			return true
		}
	}
	return false
}

func checkAndAppendState(appName, state string) {
	if checkNewLastState(appName, state) {
		states[appName] = append(states[appName], state)
	}
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
						checkAndAppendState(app.Name, "EXISTS")
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
					if len(app.AppErr) > 0 {
						//if AppErr, show them
						checkAndAppendState(app.AppName, fmt.Sprintf("%s: %s", app.State.String(), app.AppErr))
					} else {
						checkAndAppendState(app.AppName, app.State.String())
					}
				}
			}
			if len(states) == len(appNames) {
				for _, appName := range appNames {
					if !checkNewLastState(appName, state) {
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
		states = make(map[string][]string)
		for _, el := range apps {
			states[el] = []string{"no info from controller"}
		}

		tc.AddProcInfo(edgeNode, checkApp(state, apps))

		callback := func() {
			t.Errorf("ASSERTION FAILED: expected apps %s in %s state", apps, state)
			for k, v := range states {
				t.Errorf("\tactual %s: %s", k, v[len(v)-1])
				if checkNewLastState(k, state) {
					t.Errorf("\thistory of states for %s:", k)
					for _, st := range v {
						t.Errorf("\t\t%s", st)
					}
				}
			}
		}

		tc.WaitForProcWithErrorCallback(secs, callback)
	}
}
