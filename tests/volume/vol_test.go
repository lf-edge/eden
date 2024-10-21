package lim

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/testcontext"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/info"
)

type volState struct {
	state     string
	timestamp time.Time
}

// This test wait for the volume's state with a timewait.
var (
	timewait = flag.Duration("timewait", time.Minute, "Timewait for items waiting")
	newitems = flag.Bool("check-new", false, "Check only new info messages")
	tc       *testcontext.TestContext
	states   map[string][]volState
	eveState *eve.State
)

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("Docker volume's state test")

	tc = testcontext.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestVolState", time.Now())

	tc.InitProject(projectName)

	tc.AddEdgeNodesFromDescription()

	eveState = eve.Init(tc.GetController(), tc.GetEdgeNode())

	tc.StartTrackingState(false)

	res := m.Run()

	os.Exit(res)
}

// checkNewLastState returns true if provided state not equals with last
func checkNewLastState(volName, state string) bool {
	volStates, ok := states[volName]
	if ok {
		lastState := volStates[len(volStates)-1]
		if lastState.state != state {
			return true
		}
	}
	return false
}

func checkAndAppendState(volName, state string) {
	if checkNewLastState(volName, state) {
		states[volName] = append(states[volName], volState{state: state, timestamp: time.Now()})
		fmt.Println(utils.AddTimestamp(fmt.Sprintf("\tvolName %s state changed to %s", volName, state)))
	}
}

func checkState(eveState *eve.State, state string, volNames []string) error {
	out := "\n"
	if state == "-" {
		foundAny := false
		if eveState.InfoAndMetrics().GetDinfo() == nil {
			//we need to wait for info
			return nil
		}
		for _, vol := range eveState.Volumes() {
			if _, inSlice := utils.FindEleInSlice(volNames, vol.Name); inSlice {
				checkAndAppendState(vol.Name, vol.EveState)
				foundAny = true
			}
		}
		if foundAny {
			return nil
		}
		for _, volName := range volNames {
			out += fmt.Sprintf(
				"no volume with %s found\n",
				volName)
		}
		return fmt.Errorf(out)
	}
	for _, vol := range eveState.Volumes() {
		if _, inSlice := utils.FindEleInSlice(volNames, vol.Name); inSlice {
			checkAndAppendState(vol.Name, vol.EveState)
		}
	}
	if len(states) == len(volNames) {
		for _, volName := range volNames {
			if !checkNewLastState(volName, state) {
				out += fmt.Sprintf(
					"volume %s state %s\n",
					volName, state)
			} else {
				return nil
			}
		}
		return fmt.Errorf(out)
	}
	return nil
}

// checkVol wait for info of ZInfoApp type with state
func checkVol(state string, volNames []string) testcontext.ProcInfoFunc {
	return func(msg *info.ZInfoMsg) error {
		eveState.InfoCallback()(msg) //feed state with new info
		return checkState(eveState, state, volNames)
	}
}

// TestVolStatus wait for application reaching the selected state
// with a timewait
func TestVolStatus(t *testing.T) {
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	args := flag.Args()
	if len(args) == 0 {
		t.Fatalf("Usage: %s [options] state vol_name...\n", os.Args[0])
	} else {
		secs := int(timewait.Seconds())
		state := args[0]
		t.Log(utils.AddTimestamp(fmt.Sprintf("volumes: '%s' state: '%s' secs: %d\n",
			args[1:], state, secs)))

		vols := args[1:]
		if vols[len(vols)-1] == "&" {
			vols = vols[:len(vols)-1]
		}
		states = make(map[string][]volState)
		for _, el := range vols {
			if el == "&" {
				continue
			}
			states[el] = []volState{{state: "no info from controller", timestamp: time.Now()}}
		}

		if !*newitems {
			// observe existing info object and feed them into eveState object
			if err := tc.GetController().InfoLastCallback(edgeNode.GetID(), nil, eveState.InfoCallback()); err != nil {
				t.Fatal(err)
			}
		}

		// we are done if our eveState object is in required state
		if ready := checkState(eveState, state, vols); ready == nil {

			tc.AddProcInfo(edgeNode, checkVol(state, vols))

			callback := func() {
				t.Errorf("ASSERTION FAILED (%s): expected volumes %s in %s state", time.Now().Format(time.RFC3339Nano), vols, state)
				for k, v := range states {
					t.Errorf("\tactual %s: %s", k, v[len(v)-1].state)
					if checkNewLastState(k, state) {
						t.Errorf("\thistory of states for %s:", k)
						for _, st := range v {
							t.Errorf("\t\tstate: %s received in: %s", st.state, st.timestamp.Format(time.RFC3339Nano))
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
