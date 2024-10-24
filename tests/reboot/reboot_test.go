package reboot

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/testcontext"
	"github.com/lf-edge/eden/pkg/tests"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/info"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// This context holds all the configuration items in the same
// way that Eden context works: the commands line options override
// YAML settings. In addition to that, context is polymorphic in
// a sense that it abstracts away a particular controller (currently
// Adam and Zedcloud are supported)
/*
tc *TestContext // TestContext is at least {
                //    controller *Controller
                //    project *Project
                //    nodes []EdgeNode
                //    ...
                // }
*/

var (
	timewait = flag.Duration("timewait", time.Minute, "Timewait for waiting. Zero value will wait infinite.")
	reboot   = flag.Bool("reboot", true, "Reboot or not reboot...")
	count    = flag.Int("count", 1, "Number of reboots")

	number int

	tc *testcontext.TestContext

	lastRebootTime *timestamppb.Timestamp
)

func checkReboot(t *testing.T, edgeNode *device.Ctx) testcontext.ProcInfoFunc {
	return func(im *info.ZInfoMsg) error {
		if im.GetZtype() != info.ZInfoTypes_ZiDevice {
			return nil
		}
		currentLastRebootTime := im.GetDinfo().LastRebootTime
		if !proto.Equal(lastRebootTime, currentLastRebootTime) {
			if im.GetDinfo().LastRebootReason == "" &&
				currentLastRebootTime.AsTime().Unix() == 0 {
				// device may not fill the info
				return nil
			}
			lastRebootTime = currentLastRebootTime
			fmt.Printf("rebooted with reason %s at %s/n", im.GetDinfo().LastRebootReason, lastRebootTime.AsTime())
			if !strings.Contains(im.GetDinfo().LastRebootReason, "NORMAL") {
				err := fmt.Errorf("abnormal reboot: %s", im.GetDinfo().LastRebootReason)
				if *reboot {
					//if we use this test to do reboot, abnormal one must errored the test
					t.Fatal(err)
				} else {
					//if we use this test as detector, abnormal one must end the test
					return err
				}
			}
			number++
			if number < *count {
				if *reboot {
					edgeNode.Reboot()
					//sync config only if reboot needed
					tc.ConfigSync(edgeNode)
				}
			} else {
				return fmt.Errorf("rebooted %d times", number)
			}
		}
		return nil
	}
}

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("Reboot Test")

	tests.TestArgsParse()

	tc = testcontext.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestReboot", time.Now())

	// Registering our own project namespace with controller for easy cleanup
	tc.InitProject(projectName)

	// Create representation of EVE instances (based on the names
	// or UUIDs that were passed in) in the context. This is the first place
	// where we're using zcli-like API:
	for _, node := range tc.GetNodeDescriptions() {
		edgeNode := node.GetEdgeNode(tc)
		if edgeNode == nil {
			// Couldn't find existing edgeNode record in the controller.
			// Need to create it from scratch now:
			// this is modeled after: zcli edge-node create <name>
			// --project=<project> --model=<model> [--title=<title>]
			// ([--edge-node-certificate=<certificate>] |
			// [--onboarding-certificate=<certificate>] |
			// [(--onboarding-key=<key> --serial=<serial-number>)])
			// [--network=<network>...]
			//
			// XXX: not sure if struct (giving us optional fields) would be better
			edgeNode = tc.NewEdgeNode(tc.WithNodeDescription(node), tc.WithCurrentProject())
		} else {
			// make sure to move EdgeNode to the project we created, again
			// this is modeled after zcli edge-node update <name> [--title=<title>]
			// [--lisp-mode=experimental|default] [--project=<project>]
			// [--clear-onboarding-certs] [--config=<key:value>...] [--network=<network>...]
			edgeNode.SetProject(projectName)
		}

		tc.ConfigSync(edgeNode)

		// finally we need to make sure that the edgeNode is in a state that we need
		// it to be, before the test can run -- this could be multiple checks on its
		// status, but for example:
		if edgeNode.GetState() == device.NotOnboarded {
			fmt.Println("Node is not onboarded now")
			os.Exit(1)
		}

		// this is a good node -- lets add it to the test context
		tc.AddNode(edgeNode)
	}

	tc.StartTrackingState(false)

	// we now have a situation where TestContext has enough EVE nodes known
	// for the rest of the tests to run. So run them:
	res := m.Run()

	// Finally, we need to cleanup whatever objects may be in in the project we created
	// and then we can exit
	os.Exit(res)
}

func TestReboot(t *testing.T) {

	if timewait.Seconds() == 0 {
		timewaitNew := time.Duration(1<<63 - 1)
		timewait = &timewaitNew
	}
	// note that GetEdgeNode() without any argument is
	// equivalent to the default (first one). Otherwise
	// one can specify a name GetEdgeNode("foo")
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Log(utils.AddTimestamp(fmt.Sprintf("Wait for state of %s", edgeNode.GetID())))

	tc.WaitForState(edgeNode, 60)

	t.Log(utils.AddTimestamp(fmt.Sprintf("timewait: %s", timewait)))
	t.Log(utils.AddTimestamp(fmt.Sprintf("reboot: %t", *reboot)))
	t.Log(utils.AddTimestamp(fmt.Sprintf("count: %d", *count)))

	lastRebootTime = tc.GetState(edgeNode).GetDinfo().LastRebootTime

	t.Log(utils.AddTimestamp(fmt.Sprintf("LastRebootTime: %s", lastRebootTime.AsTime())))

	t.Log(utils.AddTimestamp(fmt.Sprintf("LastRebootReason: %s", tc.GetState(edgeNode).GetDinfo().LastRebootReason)))

	tc.AddProcInfo(edgeNode, checkReboot(t, edgeNode))

	if *reboot {
		edgeNode.Reboot()
		//sync config only if reboot needed
		tc.ConfigSync(edgeNode)
	}

	tc.WaitForProc(int(timewait.Seconds()))

	t.Log(utils.AddTimestamp(fmt.Sprintf("Number of reboots: %d\n", number)))
}
