package reboot

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
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
	timewait = flag.Int("timewait", 600, "Timewait for reboot waiting in seconds")
	reboot   = flag.Bool("reboot", true, "Reboot or not reboot...")
	count    = flag.Int("count", 1, "Number of reboots")

	tc *projects.TestContext

	lastRebootTime *timestamp.Timestamp
)

func checkReboot(im *info.ZInfoMsg) error {
	if im.GetZtype() != info.ZInfoTypes_ZiDevice {
		return nil
	}
	fmt.Println(im)
	currentLastRebootTime := im.GetDinfo().LastRebootTime
	if !proto.Equal(lastRebootTime, currentLastRebootTime) {
		lastRebootTime = currentLastRebootTime
		return fmt.Errorf("rebooted with reason %s at %s", im.GetDinfo().LastRebootReason, ptypes.TimestampString(lastRebootTime))
	}
	return nil
}

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("Reboot Test")

	tc = projects.NewTestContext()

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
			log.Fatal("Node is not onboarded now")
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
	// note that GetEdgeNode() without any argument is
	// equivalent to the default (first one). Otherwise
	// one can specify a name GetEdgeNode("foo")
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Logf("Wait for state of %s", edgeNode.GetName())

	t.Log("timewait: ", *timewait)
	t.Log("reboot: ", *reboot)
	t.Log("count: ", *count)

	tc.WaitForState(edgeNode, *timewait)

	lastRebootTime = tc.GetState(edgeNode).GetDinfo().LastRebootTime

	t.Logf("lastRebootTime: %s", ptypes.TimestampString(lastRebootTime))

	var number int
	for *count > number {
		// this is modeled after: zcli edge-node reboot [-f] <name>
		// this is expected to be a synchronous call for now
		if *reboot {
			edgeNode.Reboot()

			//sync config only if reboot needed
			tc.ConfigSync(edgeNode)
		}

		t.Logf("Wait for reboot of %s", edgeNode.GetName())

		tc.AddProcInfo(edgeNode, checkReboot)

		tc.WaitForProc(*timewait)

		number++
		t.Logf("Reboot number: %d\n", number)
	}

	t.Logf("Number of reboots: %d\n", number)
}
