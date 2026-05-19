package zedagent

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/lf-edge/eden/pkg/controller/eflowlog"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/testcontext"
	"github.com/lf-edge/eden/pkg/tests"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/flowlog"
	"github.com/lf-edge/eve-api/go/info"
	"github.com/lf-edge/eve-api/go/metrics"
)

var (
	number   = flag.Int("number", 1, "The number of items (0=unlimited) to collect")
	timewait = flag.Duration("timewait", 10*time.Minute, "Timeout for waiting on items")
	out      = flag.String("out", "", "Parameters for out separated by ':'")

	tc    *testcontext.TestContext
	query = map[string]string{}
	found bool
	items int
)

func mkquery() error {
	for _, arg := range flag.Args() {
		a := strings.TrimSuffix(arg, "&")
		for _, f := range strings.Split(a, " ") {
			s := strings.Split(f, ":")
			if len(s) == 1 {
				if s[0] == "" {
					continue
				}
				return fmt.Errorf("incorrect query: %s", f)
			}
			query[s[0]] = s[1]
		}
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

func TestMain(m *testing.M) {
	fmt.Println("zedagent integration tests")

	tests.TestArgsParse()

	tc = testcontext.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestZedAgent", time.Now())

	tc.InitProject(projectName)

	for _, node := range tc.GetNodeDescriptions() {
		edgeNode := node.GetEdgeNode(tc)
		if edgeNode == nil {
			edgeNode = tc.NewEdgeNode(tc.WithNodeDescription(node), tc.WithCurrentProject())
		} else {
			edgeNode.SetProject(projectName)
		}

		tc.ConfigSync(edgeNode)

		if edgeNode.GetState() == device.NotOnboarded {
			log.Fatal("Node is not onboarded now")
		}

		tc.AddNode(edgeNode)
	}

	tc.StartTrackingState(false)

	res := m.Run()

	os.Exit(res)
}

// TestInfo waits for a ZInfoMsg matching the provided query and prints
// the requested fields.  This is the primary test function used by
// most testdata scripts in this package.
func TestInfo(t *testing.T) {
	err := mkquery()
	if err != nil {
		t.Fatal(err)
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Log(utils.AddTimestamp(fmt.Sprintf("Wait for info of %s number=%d timewait=%s",
		edgeNode.GetID(), *number, timewait)))

	tc.AddProcInfo(edgeNode, func(ei *info.ZInfoMsg) error {
		return func(t *testing.T, edgeNode *device.Ctx, ei *info.ZInfoMsg) error {
			name := edgeNode.GetID()
			if query != nil {
				if einfo.ZInfoFind(ei, query) {
					found = true
				} else {
					return nil
				}
			}
			t.Log(utils.AddTimestamp(fmt.Sprintf("INFO %d(%d) from %s:", items+1, *number, name)))
			if len(*out) == 0 {
				einfo.ZInfoPrn(ei, types.OutputFormatLines)
			} else {
				einfo.ZInfoPrintFiltered(ei, strings.Split(*out, ":")).Print()
			}
			cnt := count("Received %d infos from %s", name.String())
			if cnt != "" {
				return errors.New(cnt)
			}
			return nil
		}(t, edgeNode, ei)
	})

	tc.WaitForProc(int(timewait.Seconds()))
}

// TestMetric waits for a ZMetricMsg matching the provided query and prints
// the requested fields.
func TestMetric(t *testing.T) {
	err := mkquery()
	if err != nil {
		t.Fatal(err)
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Log(utils.AddTimestamp(fmt.Sprintf("Wait for metric of %s number=%d timewait=%s",
		edgeNode.GetID(), *number, timewait)))

	tc.AddProcMetric(edgeNode, func(metric *metrics.ZMetricMsg) error {
		return func(t *testing.T, edgeNode *device.Ctx, mtr *metrics.ZMetricMsg) error {
			name := edgeNode.GetID()
			if query != nil {
				if emetric.MetricItemFind(mtr, query) {
					found = true
				} else {
					return nil
				}
			}
			t.Log(utils.AddTimestamp(fmt.Sprintf("METRICS %d(%d) from %s:", items+1, *number, name)))
			if len(*out) == 0 {
				emetric.MetricPrn(mtr, types.OutputFormatLines)
			} else {
				emetric.MetricItemPrint(mtr, strings.Split(*out, ":")).Print()
			}
			cnt := count("Received %d metrics from %s", name.String())
			if cnt != "" {
				return errors.New(cnt)
			}
			return nil
		}(t, edgeNode, metric)
	})

	tc.WaitForProc(int(timewait.Seconds()))
}

// TestFlowLog waits for a FlowMessage matching the provided query and prints
// the requested fields.
func TestFlowLog(t *testing.T) {
	err := mkquery()
	if err != nil {
		t.Fatal(err)
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Log(utils.AddTimestamp(fmt.Sprintf("Wait for FlowLog of %s number=%d timewait=%s",
		edgeNode.GetID(), *number, timewait)))

	tc.AddProcFlowLog(edgeNode, func(fl *flowlog.FlowMessage) error {
		return func(t *testing.T, edgeNode *device.Ctx, fl *flowlog.FlowMessage) error {
			name := edgeNode.GetID()
			if query != nil {
				if eflowlog.FlowLogItemFind(fl, query) {
					found = true
				} else {
					return nil
				}
			}
			t.Log(utils.AddTimestamp(fmt.Sprintf("FLOWLOG %d(%d) from %s:", items+1, *number, name)))
			if len(*out) == 0 {
				eflowlog.FlowLogPrn(fl, types.OutputFormatLines)
			} else {
				eflowlog.FlowLogItemPrint(fl, strings.Split(*out, ":")).Print()
			}
			cnt := count("Received %d FlowLog from %s", name.String())
			if cnt != "" {
				return errors.New(cnt)
			}
			return nil
		}(t, edgeNode, fl)
	})

	tc.WaitForProc(int(timewait.Seconds()))
}
