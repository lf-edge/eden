package network

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
)

type nwState struct {
	state     string
	timestamp time.Time
}

// This test wait for the network's state with a timewait.
var (
	timewait = flag.Duration("timewait", time.Minute, "Timewait for items waiting")
	checkNew = flag.Bool("check-new", false, "Check for the new state after state transition")
	tc       *projects.TestContext
	states   map[string][]nwState

	lastRebootTime time.Time
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

	tc.StartTrackingState(false)

	res := m.Run()

	os.Exit(res)
}

// checkNewLastState returns true if provided state not equals with last
func checkNewLastState(netName, state string) bool {
	netStates, ok := states[netName]
	if ok {
		lastState := netStates[len(netStates)-1]
		if lastState.state != state {
			return true
		}
	}
	return false
}

func checkAndAppendState(netName, state string) {
	if checkNewLastState(netName, state) {
		states[netName] = append(states[netName], nwState{state: state, timestamp: time.Now()})
		fmt.Println(utils.AddTimestamp(fmt.Sprintf("\tnetName %s state changed to %s", netName, state)))
	}
}

func checkState(eveState *eve.State, state string, netNames []string) error {
	out := "\n"
	if state == "-" {
		foundAny := false
		for _, net := range eveState.NotDeletedNetworks() {
			if _, inSlice := utils.FindEleInSlice(netNames, net.Name); inSlice {
				checkAndAppendState(net.Name, net.EveState)
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
	for _, net := range eveState.NotDeletedNetworks() {
		if _, inSlice := utils.FindEleInSlice(netNames, net.Name); inSlice {
			checkAndAppendState(net.Name, net.EveState)
		}
	}
	if len(states) == len(netNames) {
		for _, netName := range netNames {
			if !checkNewLastState(netName, state) {
				currentLastRebootTime := eveState.NodeState().LastRebootTime
				// if we rebooted we may miss state transition
				if *checkNew && !currentLastRebootTime.After(lastRebootTime) {
					// first one is no info from controller
					// the second is initial state
					// we want to wait for the third or later, thus new state
					if len(states[netName]) <= 2 {
						fmt.Println(utils.AddTimestamp(fmt.Sprintf("\tnetName %s wait for new state", netName)))
						return nil
					}
				}
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

// checkNet wait for info of ZInfoApp type with state
func checkNet(edgeNode *device.Ctx, state string, volNames []string) projects.ProcInfoFunc {
	return func(_ *info.ZInfoMsg) error {
		return checkState(tc.GetState(edgeNode).GetEVEState(), state, volNames)
	}
}

// TestNetworkStatus wait for networks reaching the selected state
// with a timewait
func TestNetworkStatus(t *testing.T) {
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	lastRebootTime = tc.GetState(edgeNode).GetEVEState().NodeState().LastRebootTime

	args := flag.Args()
	if len(args) == 0 {
		t.Fatalf("Usage: %s [options] state vol_name...\n", os.Args[0])
	} else {
		secs := int(timewait.Seconds())
		state := args[0]
		t.Log(utils.AddTimestamp(fmt.Sprintf("networks: '%s' expected state: '%s' secs: %d\n",
			args[1:], state, secs)))

		nws := args[1:]
		if nws[len(nws)-1] == "&" {
			nws = nws[:len(nws)-1]
		}
		states = make(map[string][]nwState)
		for _, el := range nws {
			states[el] = []nwState{{state: "no info from controller", timestamp: time.Now()}}
		}

		// we are done if our eveState object is in required state
		if ready := checkState(tc.GetState(edgeNode).GetEVEState(), state, nws); ready == nil {

			tc.AddProcInfo(edgeNode, checkNet(edgeNode, state, nws))

			callback := func() {
				t.Errorf("ASSERTION FAILED (%s): expected networks %s in %s state", time.Now().Format(time.RFC3339Nano), nws, state)
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
