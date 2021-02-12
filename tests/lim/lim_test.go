package lim

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/tests"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
)

var (
	number   = flag.Int("number", 1, "The number of items (0=unlimited) you need to get")
	timewait = flag.Duration("timewait", 10*time.Minute, "Timewait for items waiting")
	out      = flag.String("out", "", "Parameters for out separated by ':'")

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
	tc *projects.TestContext

	query = map[string]string{}
	found bool
	items int
)

func mkquery() error {
	for _, a := range flag.Args() {
		s := strings.Split(a, ":")
		if len(s) == 1 {
			return fmt.Errorf("incorrect query: %s", a)
		}
		query[s[0]] = s[1]
	}

	return nil
}

func count(msg string, node string) string {
	if found {
		if *number == 0 {
			return ""
		}
		items++
		if items >= *number {
			return fmt.Sprintf(msg, items, node)
		}
		return ""
	}
	return ""
}

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("Log/Info/Metric Test")

	tests.TestArgsParse()

	tc = projects.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestLogInfoMetric", time.Now())

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

	// Finally, we need to cleanup whatever objects may be in in the
	// project we created and then we can exit
	os.Exit(res)
}

func TestLog(t *testing.T) {
	err := mkquery()
	if err != nil {
		t.Fatal(err)
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Logf("Wait for log of %s number=%d timewait=%s\n",
		edgeNode.GetID(), *number, timewait)

	tc.AddProcLog(edgeNode, func(log *elog.FullLogEntry) error {
		return func(t *testing.T, edgeNode *device.Ctx,
			log *elog.FullLogEntry) error {
			name := edgeNode.GetID()
			if query != nil {
				if elog.LogItemFind(log, query) {
					found = true
				} else {
					return nil
				}
			}
			t.Logf("LOG %d(%d) from %s:\n", items+1, *number, name)
			if len(*out) == 0 {
				elog.LogPrn(log, elog.LogLines)
			} else {
				elog.LogItemPrint(log, elog.LogLines,
					strings.Split(*out, ":")).Print()
			}

			cnt := count("Recieved %d logs from %s", name.String())
			if cnt != "" {
				return fmt.Errorf(cnt)
			}
			return nil
		}(t, edgeNode, log)
	})

	tc.WaitForProc(int(timewait.Seconds()))
}

func TestInfo(t *testing.T) {
	err := mkquery()
	if err != nil {
		t.Fatal(err)
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Logf("Wait for info of %s number=%d timewait=%s\n",
		edgeNode.GetID(), *number, timewait)

	tc.AddProcInfo(edgeNode, func(ei *info.ZInfoMsg) error {
		return func(t *testing.T, edgeNode *device.Ctx,
			ei *info.ZInfoMsg) error {
			name := edgeNode.GetID()
			if query != nil {
				if einfo.ZInfoFind(ei, query) != nil {
					found = true
				} else {
					return nil
				}
			}

			t.Logf("INFO %d(%d) from %s:\n", items+1, *number, name)
			if len(*out) == 0 {
				einfo.InfoPrn(ei)
			} else {
				einfo.ZInfoPrint(ei,
					strings.Split(*out, ":")).Print()
			}
			cnt := count("Recieved %d infos from %s", name.String())
			if cnt != "" {
				return fmt.Errorf(cnt)
			}
			return nil
		}(t, edgeNode, ei)
	})

	tc.WaitForProc(int(timewait.Seconds()))
}

func TestMetrics(t *testing.T) {
	err := mkquery()
	if err != nil {
		t.Fatal(err)
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Logf("Wait for metric of %s number=%d timewait=%s\n",
		edgeNode.GetID(), *number, timewait)

	tc.AddProcMetric(edgeNode, func(metric *metrics.ZMetricMsg) error {
		return func(t *testing.T, edgeNode *device.Ctx,
			mtr *metrics.ZMetricMsg) error {
			name := edgeNode.GetID()
			if query != nil {
				if emetric.MetricItemFind(mtr, query) {
					found = true
				} else {
					return nil
				}
			}

			t.Logf("METRICS %d(%d) from %s:\n",
				items+1, *number, name)
			if len(*out) == 0 {
				emetric.MetricPrn(mtr)
			} else {
				emetric.MetricItemPrint(mtr,
					strings.Split(*out, ":")).Print()
			}

			cnt := count("Received %d metrics from %s", name.String())
			if cnt != "" {
				return fmt.Errorf(cnt)
			}
			return nil
		}(t, edgeNode, metric)
	})

	tc.WaitForProc(int(timewait.Seconds()))
}
