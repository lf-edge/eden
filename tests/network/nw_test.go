package network

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
)

// This test wait for the network's state with a timewait.
var (
	timewait = flag.Duration("timewait", time.Minute, "Timewait for items waiting")
	tc       *projects.TestContext
	states   map[string][]string
	eveState *eve.State
)

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("Network's state test")

	tc = projects.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestNetState", time.Now())

	tc.InitProject(projectName)

	tc.AddEdgeNodesFromDescription()

	eveState = eve.Init(tc.GetController(), tc.GetEdgeNode())

	tc.StartTrackingState(true)

	res := m.Run()

	os.Exit(res)
}

// checkNewLastState returns true if provided state not equals with last
func checkNewLastState(volName, state string) bool {
	volStates, ok := states[volName]
	if ok {
		lastState := volStates[len(volStates)-1]
		fmt.Printf("lastState: %s state: %s\n", lastState, state)
		if lastState != state {
			return true
		}
	}
	return false
}

func checkAndAppendState(volName, state string) {
	if checkNewLastState(volName, state) {
		states[volName] = append(states[volName], state)
	}
}

func checkState(eveState *eve.State, state string, netNames []string) error {
	out := "\n"
	if state == "-" {
		foundAny := false
		for _, vol := range eveState.Networks() {
			if _, inSlice := utils.FindEleInSlice(netNames, vol.Name); inSlice {
				checkAndAppendState(vol.Name, "EXISTS")
				foundAny = true
			}
		}
		if foundAny {
			return nil
		}
		for _, netName := range netNames {
			out += fmt.Sprintf(
				"no network with %s found\n",
				netName)
		}
		return fmt.Errorf(out)
	}
	for _, net := range eveState.Networks() {
		if _, inSlice := utils.FindEleInSlice(netNames, net.Name); inSlice {
			fmt.Printf("%s: %s\n", net.Name, net.EveState)
			checkAndAppendState(net.Name, net.EveState)
		}
	}
	if len(states) == len(netNames) {
		for _, netName := range netNames {
			if !checkNewLastState(netName, state) {
				out += fmt.Sprintf(
					"network %s state %s\n",
					netName, state)
			} else {
				return nil
			}
		}
		return fmt.Errorf(out)
	}
	return nil
}

//checkNet wait for info of ZInfoApp type with state
func checkNet(state string, volNames []string) projects.ProcInfoFunc {
	return func(msg *info.ZInfoMsg) error {
		eveState.InfoCallback()(msg, nil) //feed state with new info
		return checkState(eveState, state, volNames)
	}
}

//TestNetworkStatus wait for networks reaching the selected state
//with a timewait
func TestNetworkStatus(t *testing.T) {
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	args := flag.Args()
	if len(args) == 0 {
		t.Fatalf("Usage: %s [options] state vol_name...\n", os.Args[0])
	} else {
		secs := int(timewait.Seconds())
		state := args[0]
		fmt.Printf("networks: '%s' state: '%s' secs: %d\n",
			args[1:], state, secs)

		vols := args[1:]
		states = make(map[string][]string)
		for _, el := range vols {
			states[el] = []string{"no info from controller"}
		}

		// observe existing info object and feed them into eveState object
		if err := tc.GetController().InfoLastCallback(edgeNode.GetID(), nil, eveState.InfoCallback()); err != nil {
			t.Fatal(err)
		}

		// we are done if our eveState object is in required state
		if ready := checkState(eveState, state, vols); ready == nil {

			tc.AddProcInfo(edgeNode, checkNet(state, vols))

			callback := func() {
				t.Errorf("ASSERTION FAILED: expected networks %s in %s state", vols, state)
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

		} else {
			t.Log(ready)
		}

		// sleep to reduce concurrency effects
		time.Sleep(1 * time.Second)
	}
}
