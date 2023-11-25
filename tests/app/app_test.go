package lim

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/tests"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	uuid "github.com/satori/go.uuid"
)

type appState struct {
	state     string
	timestamp time.Time
}

// This test wait for the app's state with a timewait.
var (
	timewait = flag.Duration("timewait", 10*time.Minute, "Timewait for items waiting")
	checkNew = flag.Bool("check-new", false, "Check for the new state after state transition")
	tc       *projects.TestContext
	states   map[string][]appState

	lastRebootTime time.Time
)

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("Docker app's state test")

	tests.TestArgsParse()

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
		if lastState.state != state {
			return true
		}
	}
	return false
}

func checkAndAppendState(appName, state string) {
	if checkNewLastState(appName, state) {
		states[appName] = append(states[appName], appState{
			state:     state,
			timestamp: time.Now(),
		})
		fmt.Println(utils.AddTimestamp(fmt.Sprintf("\tappName %s state changed to %s", appName, state)))
	}
}

func checkState(eveState *eve.State, state string, appNames []string) error {
	out := "\n"
	if state == "-" {
		foundAny := false
		if !eveState.Prepared() {
			//we need to wait for info
			return nil
		}
		for _, app := range eveState.NotDeletedApplications() {
			if _, inSlice := utils.FindEleInSlice(appNames, app.Name); inSlice {
				checkAndAppendState(app.Name, app.EVEState)
				foundAny = true
			}
		}
		if foundAny {
			return nil
		}
		for _, appName := range appNames {
			out += utils.AddTimestamp(fmt.Sprintf(
				"no app with %s found\n",
				appName))
		}
		return fmt.Errorf(out)
	}
	for _, app := range eveState.NotDeletedApplications() {
		if _, inSlice := utils.FindEleInSlice(appNames, app.Name); inSlice {
			checkAndAppendState(app.Name, app.EVEState)
		}
	}
	if len(states) == len(appNames) {
		for _, appName := range appNames {
			if !checkNewLastState(appName, state) {
				currentLastRebootTime := eveState.NodeState().LastRebootTime
				// if we rebooted we may miss state transition
				if *checkNew && !currentLastRebootTime.After(lastRebootTime) {
					// first one is no info from controller
					// the second is initial state
					// we want to wait for the third or later, thus new state
					if len(states[appName]) <= 2 {
						fmt.Println(utils.AddTimestamp(fmt.Sprintf("\tappName %s wait for new state", appName)))
						return nil
					}
				}
				out += utils.AddTimestamp(fmt.Sprintf(
					"app %s state %s\n",
					appName, state))
			} else {
				return nil
			}
		}
		return fmt.Errorf(out)
	}
	return nil
}

// checkApp wait for info of ZInfoApp type with state
func checkApp(edgeNode *device.Ctx, state string, appNames []string) projects.ProcInfoFunc {
	return func(msg *info.ZInfoMsg) error {
		return checkState(tc.GetState(edgeNode).GetEVEState(), state, appNames)
	}
}

// TestAppStatus wait for application reaching the selected state
// with a timewait
func TestAppStatus(t *testing.T) {
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	lastRebootTime = tc.GetState(edgeNode).GetEVEState().NodeState().LastRebootTime

	args := flag.Args()
	if len(args) == 0 {
		t.Fatalf("Usage: %s [options] state app_name...\n", os.Args[0])
	} else {
		secs := int(timewait.Seconds())
		state := args[0]
		fmt.Printf("apps: '%s' state: '%s' secs: %d\n",
			args[1:], state, secs)

		apps := args[1:]
		if apps[len(apps)-1] == "&" {
			apps = apps[:len(apps)-1]
		}
		states = make(map[string][]appState)
		for _, el := range apps {
			states[el] = []appState{{
				state:     "no info from controller",
				timestamp: time.Now()}}
		}

		if ready := checkState(tc.GetState(edgeNode).GetEVEState(), state, apps); ready == nil {

			tc.AddProcInfo(edgeNode, checkApp(edgeNode, state, apps))

			callback := func() {
				t.Errorf("ASSERTION FAILED (%s): expected apps %s in %s state", time.Now().Format(time.RFC3339Nano), apps, state)
				for k, v := range states {
					t.Errorf("\tactual %s: %s", k, v[len(v)-1].state)
					if checkNewLastState(k, state) {
						t.Errorf("\thistory of states for %s:", k)
						for _, st := range v {
							t.Errorf("\t\tstate: %s received in: %s", st.state, st.timestamp.Format(time.RFC3339Nano))
						}
					}
					for _, app := range tc.GetState(edgeNode).GetEVEState().NotDeletedApplications() {
						if app.Name == k {
							appID, err := uuid.FromString(app.UUID)
							if err != nil {
								t.Fatal(err)
							}
							fmt.Printf("--- app %s logs ---\n", app.Name)
							if err = tc.GetController().LogAppsChecker(edgeNode.GetID(), appID, nil, eapps.HandleFactory(types.OutputFormatJSON, false), eapps.LogExist, 0); err != nil {
								t.Fatalf("LogAppsChecker: %s", err)
							}
							fmt.Println("------")
						}
					}
				}
			}

			tc.WaitForProcWithErrorCallback(secs, callback)

		} else {
			t.Log(utils.AddTimestamp(ready.Error()))
		}

		// sleep to reduce concurrency effects
		time.Sleep(1 * time.Second)
	}
}
