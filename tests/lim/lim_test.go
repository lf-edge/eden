package lim

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/controller/eflowlog"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/testcontext"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/flowlog"
	"github.com/lf-edge/eve-api/go/info"
	"github.com/lf-edge/eve-api/go/logs"
	"github.com/lf-edge/eve-api/go/metrics"
)

var (
	number   = flag.Int("number", 1, "The number of items (0=unlimited) you need to get")
	timewait = flag.Duration("timewait", 10*time.Minute, "Timewait for items waiting")
	out      = flag.String("out", "", "Parameters for out separated by ':'")
	app      = flag.String("app", "", "Name of app for TestAppLogs")

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
	tc *testcontext.TestContext

	query = map[string]string{}
	found bool
	items int
)

func mkquery() error {
	for _, arg := range flag.Args() {
		// we use & to indicate background process
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

func TestLog(t *testing.T) {
	err := mkquery()
	if err != nil {
		t.Fatal(err)
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Log(utils.AddTimestamp(fmt.Sprintf("Wait for log of %s number=%d timewait=%s\n",
		edgeNode.GetID(), *number, timewait)))

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
			t.Log(utils.AddTimestamp(fmt.Sprintf("LOG %d(%d) from %s:\n", items+1, *number, name)))
			if len(*out) == 0 {
				elog.LogPrint(log, types.OutputFormatLines)
			} else {
				elog.LogItemPrint(log, types.OutputFormatLines,
					strings.Split(*out, ":")).Print()
			}

			cnt := count("Received %d logs from %s", name.String())
			if cnt != "" {
				return errors.New(cnt)
			}
			return nil
		}(t, edgeNode, log)
	})

	tc.WaitForProc(int(timewait.Seconds()))
}

func TestAppLog(t *testing.T) {
	err := mkquery()
	if err != nil {
		t.Fatal(err)
	}

	if app == nil {
		t.Fatal("Please provide app flag")
	}

	appName := *app

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	appID := ""

	for _, appUUID := range edgeNode.GetApplicationInstances() {
		for _, appConfig := range tc.GetController().ListApplicationInstanceConfig() {
			if appConfig.Displayname == appName && appUUID == appConfig.Uuidandversion.GetUuid() {
				appID = appUUID
				break
			}
		}
		if appID != "" {
			break
		}
	}

	if appID == "" {
		t.Fatalf("No app with name %s found", appName)
	}

	appUUID, err := uuid.FromString(appID)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(utils.AddTimestamp(fmt.Sprintf("Wait for app log of %s app %s number=%d timewait=%s\n",
		edgeNode.GetID(), appName, *number, timewait)))

	tc.AddProcAppLog(edgeNode, appUUID, func(log *logs.LogEntry) error {
		return func(t *testing.T, edgeNode *device.Ctx,
			log *logs.LogEntry) error {
			name := edgeNode.GetID()
			if query != nil {
				if eapps.LogItemFind(log, query) {
					found = true
				} else {
					return nil
				}
			}
			t.Log(utils.AddTimestamp(fmt.Sprintf("APP LOG %d(%d) from %s:\n", items+1, *number, name)))
			if len(*out) == 0 {
				eapps.LogPrn(log, types.OutputFormatLines)
			} else {
				eapps.LogItemPrint(log, types.OutputFormatLines,
					strings.Split(*out, ":")).Print()
			}

			cnt := count("Received %d app logs from %s", name.String())
			if cnt != "" {
				return errors.New(cnt)
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

	t.Log(utils.AddTimestamp(fmt.Sprintf("Wait for info of %s number=%d timewait=%s\n",
		edgeNode.GetID(), *number, timewait)))

	tc.AddProcInfo(edgeNode, func(ei *info.ZInfoMsg) error {
		return func(t *testing.T, edgeNode *device.Ctx,
			ei *info.ZInfoMsg) error {
			name := edgeNode.GetID()
			if query != nil {
				if einfo.ZInfoFind(ei, query) {
					found = true
				} else {
					return nil
				}
			}

			t.Log(utils.AddTimestamp(fmt.Sprintf("INFO %d(%d) from %s:\n", items+1, *number, name)))
			if len(*out) == 0 {
				einfo.ZInfoPrn(ei, types.OutputFormatLines)
			} else {
				einfo.ZInfoPrintFiltered(ei,
					strings.Split(*out, ":")).Print()
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

func TestMetrics(t *testing.T) {
	err := mkquery()
	if err != nil {
		t.Fatal(err)
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Log(utils.AddTimestamp(fmt.Sprintf("Wait for metric of %s number=%d timewait=%s\n",
		edgeNode.GetID(), *number, timewait)))

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

			t.Log(utils.AddTimestamp(fmt.Sprintf("METRICS %d(%d) from %s:\n",
				items+1, *number, name)))
			if len(*out) == 0 {
				emetric.MetricPrn(mtr, types.OutputFormatLines)
			} else {
				emetric.MetricItemPrint(mtr,
					strings.Split(*out, ":")).Print()
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

func TestFlowLog(t *testing.T) {
	err := mkquery()
	if err != nil {
		t.Fatal(err)
	}

	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Log(utils.AddTimestamp(fmt.Sprintf("Wait for FlowLog of %s number=%d timewait=%s\n",
		edgeNode.GetID(), *number, timewait)))

	tc.AddProcFlowLog(edgeNode, func(log *flowlog.FlowMessage) error {
		return func(t *testing.T, edgeNode *device.Ctx,
			flowLog *flowlog.FlowMessage) error {
			name := edgeNode.GetID()
			if query != nil {
				if eflowlog.FlowLogItemFind(flowLog, query) {
					found = true
				} else {
					return nil
				}
			}
			t.Log(utils.AddTimestamp(fmt.Sprintf("FLOWLOG %d(%d) from %s:\n", items+1, *number, name)))
			if len(*out) == 0 {
				eflowlog.FlowLogPrn(flowLog, types.OutputFormatLines)
			} else {
				eflowlog.FlowLogItemPrint(flowLog, strings.Split(*out, ":")).Print()
			}

			cnt := count("Received %d FlowLog from %s", name.String())
			if cnt != "" {
				return errors.New(cnt)
			}
			return nil
		}(t, edgeNode, log)
	})

	tc.WaitForProc(int(timewait.Seconds()))
}
