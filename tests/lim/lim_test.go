package lim

import (
	"flag"
	"fmt"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
	"time"
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
	number   = flag.Int("number", 1, "The number of items (0=unlimited) you need to get")
	timewait = flag.Int("timewait", 60, "Timewait for items waiting in seconds")
	tc       *projects.TestContext
)

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("Log/Info/Metric Test")

	tc = projects.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestLogInfoMetric", time.Now())

	// Registering our own project namespace with controller for easy cleanup
	tc.InitProject(projectName)

	// Create representation of EVE instances (based on the names
	// or UUIDs that were passed in) in the context. This is the first place
	// where we're using zcli-like API:
	for _, node := range tc.GetNodeDescriptions() {
		edgeNode := tc.GetController().GetEdgeNode(node.Name)
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

func TestLog(t *testing.T) {
	var logs int

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	tc.ConfigSync(edgeNode)

	t.Logf("Wait for log of %s number=%d timewait=%d\n", edgeNode.GetName(), *number, *timewait)

	tc.AddProcLog(edgeNode, func(log *elog.LogItem) error {
		return func(t *testing.T, edgeNode *device.Ctx, log *elog.LogItem) error {
			t.Logf("LOG from %s: %s", edgeNode.GetName(), log)
			if *number == 0 {
				return nil
			} else {
				logs += 1
				if logs >= *number {
					return fmt.Errorf("Recieved %d logs from %s", logs, edgeNode.GetName())
				} else {
					return nil
				}
			}
		}(t, edgeNode, log)
	})

	tc.WaitForProc(*timewait)
}

func TestInfo(t *testing.T) {
	var infos int

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	tc.ConfigSync(edgeNode)

	t.Logf("Wait for info of %s number=%d timewait=%d\n", edgeNode.GetName(), *number, *timewait)

	tc.AddProcInfo(edgeNode, func(einfo *info.ZInfoMsg) error {
		return func(t *testing.T, edgeNode *device.Ctx, einfo *info.ZInfoMsg) error {
			t.Logf("INFO from %s: %s", edgeNode.GetName(), einfo)
			if *number == 0 {
				return nil
			} else {
				infos += 1
				if infos >= *number {
					return fmt.Errorf("Recieved %d infos from %s", infos, edgeNode.GetName())
				} else {
					return nil
				}
			}
		}(t, edgeNode, einfo)
	})

	tc.WaitForProc(*timewait)
}

func TestMetrics(t *testing.T) {
	var metrs int

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	tc.ConfigSync(edgeNode)

	t.Logf("Wait for metric of %s number=%d timewait=%d\n", edgeNode.GetName(), *number, *timewait)

	tc.AddProcMetric(edgeNode, func(metric *metrics.ZMetricMsg) error {
		return func(t *testing.T, edgeNode *device.Ctx, metric *metrics.ZMetricMsg) error {
			t.Logf("METRIC from %s: %s", edgeNode.GetName(), metric)
			if *number == 0 {
				return nil
			} else {
				metrs += 1
				if metrs >= *number {
					return fmt.Errorf("Recieved %d metrics from %s", metrs, edgeNode.GetName())
				} else {
					return nil
				}
			}
		}(t, edgeNode, metric)
	})

	tc.WaitForProc(*timewait)
}
