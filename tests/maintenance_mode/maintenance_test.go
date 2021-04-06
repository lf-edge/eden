package maintenance

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"

	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/tests"
	"github.com/lf-edge/eden/pkg/utils"
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
	// XXX lower default?
	timewait = flag.Duration("timewait", 3*time.Minute, "Timewait for waiting")

	tc *projects.TestContext

	configRebootCounter uint32
)

func checkRebootCounterMatch() projects.ProcInfoFunc {
	return func(im *info.ZInfoMsg) error {
		if im.GetZtype() != info.ZInfoTypes_ZiDevice {
			return nil
		}
		statusRebootCounter := im.GetDinfo().RebootConfigCounter
		if statusRebootCounter == configRebootCounter {
			return fmt.Errorf("reached RebootCounter: %d",
				configRebootCounter)
		}
		return nil
	}
}

func checkMaintenanceMode() projects.ProcInfoFunc {
	return func(im *info.ZInfoMsg) error {
		if im.GetZtype() != info.ZInfoTypes_ZiDevice {
			return nil
		}
		log.Infof("checkMaintenceMode: got %+v", im.GetDinfo())
		maintenanceMode := im.GetDinfo().MaintenanceMode
		maintenanceModeReason := im.GetDinfo().MaintenanceModeReason
		log.Infof("checkMaintenanceMode: maintenanceMode %t maintenanceModeReason %s",
			maintenanceMode, maintenanceModeReason)
		if maintenanceMode {
			return fmt.Errorf("entered mainteance mode with reason %s",
				maintenanceModeReason)
		}
		return nil
	}
}

func checkNoMaintenanceMode() projects.ProcInfoFunc {
	return func(im *info.ZInfoMsg) error {
		if im.GetZtype() != info.ZInfoTypes_ZiDevice {
			return nil
		}
		log.Infof("checkNoMaintenceMode: got %+v", im.GetDinfo())
		maintenanceMode := im.GetDinfo().MaintenanceMode
		maintenanceModeReason := im.GetDinfo().MaintenanceModeReason
		log.Infof("checkNoMaintenanceMode: maintenanceMode %t maintenanceModeReason %s",
			maintenanceMode, maintenanceModeReason)
		if !maintenanceMode {
			return fmt.Errorf("done with mainteance mode (reason %s)",
				maintenanceModeReason)
		}
		return nil
	}
}

// TestMain is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. For any action, if the instance
// is not specified explicitly it is assumed to be the first one in the slice
func TestMain(m *testing.M) {
	fmt.Println("Maintenance Test")

	tests.TestArgsParse()

	tc = projects.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestMaintenance", time.Now())

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

func TestMaintenance(t *testing.T) {
	// note that GetEdgeNode() without any argument is
	// equivalent to the default (first one). Otherwise
	// one can specify a name GetEdgeNode("foo")
	edgeNode := tc.GetEdgeNode(tc.WithTest(t))

	t.Log(utils.AddTimestamp(fmt.Sprintf("Wait for state of %s", edgeNode.GetID())))
	tc.WaitForState(edgeNode, int(timewait.Seconds()))

	t.Log(utils.AddTimestamp(fmt.Sprintf("timewait: %s", timewait)))

	statusRebootCounter := tc.GetState(edgeNode).GetDinfo().RebootConfigCounter
	configRebootCounter, _ = edgeNode.GetRebootCounter()
	if statusRebootCounter != configRebootCounter {
		t.Logf(utils.AddTimestamp(fmt.Sprintf("Wait for match: statusRebootCounter: %d, configRebootCounter %d",
			statusRebootCounter, configRebootCounter)))

		// Wait for match
		tc.AddProcInfo(edgeNode, checkRebootCounterMatch())
		tc.WaitForProc(int(timewait.Seconds()))
	}
	statusRebootCounter = tc.GetState(edgeNode).GetDinfo().RebootConfigCounter
	configRebootCounter, _ = edgeNode.GetRebootCounter()
	t.Logf(utils.AddTimestamp(fmt.Sprintf("Wait done: statusRebootCounter: %d, configRebootCounter %d",
		statusRebootCounter, configRebootCounter)))
	t.Logf(utils.AddTimestamp(fmt.Sprintf("lastRestartCounter: %d",
		tc.GetState(edgeNode).GetDinfo().RestartCounter)))

	// XXX part of sanitize? Otherwise we need to clear from the config as well.
	maintenanceMode := tc.GetState(edgeNode).GetDinfo().MaintenanceMode
	maintenanceModeReason := tc.GetState(edgeNode).GetDinfo().MaintenanceModeReason
	t.Logf(utils.AddTimestamp(fmt.Sprintf("maintenanceMode %t maintenanceModeReason %s",
		maintenanceMode, maintenanceModeReason)))
	if maintenanceMode {
		t.Logf(utils.AddTimestamp(fmt.Sprintf("Wait for maintenance mode clear for %s",
			edgeNode.GetID())))

		tc.AddProcInfo(edgeNode, checkNoMaintenanceMode())

		tc.WaitForProc(int(timewait.Seconds()))
	}
	maintenanceMode = tc.GetState(edgeNode).GetDinfo().MaintenanceMode
	maintenanceModeReason = tc.GetState(edgeNode).GetDinfo().MaintenanceModeReason
	t.Logf(utils.AddTimestamp(fmt.Sprintf("Wait clear done: maintenanceMode %t maintenanceModeReason %s",
		maintenanceMode, maintenanceModeReason)))

	// send command
	edgeNode.SetConfigItem("maintenance.mode", "enabled")
	tc.ConfigSync(edgeNode)
	t.Logf(utils.AddTimestamp(fmt.Sprintf("Wait for maintenance mode for %s", edgeNode.GetID())))

	tc.AddProcInfo(edgeNode, checkMaintenanceMode())

	tc.WaitForProc(int(timewait.Seconds()))

	maintenanceMode = tc.GetState(edgeNode).GetDinfo().MaintenanceMode
	maintenanceModeReason = tc.GetState(edgeNode).GetDinfo().MaintenanceModeReason
	t.Logf(utils.AddTimestamp(fmt.Sprintf("Wait done: maintenanceMode %t maintenanceModeReason %s",
		maintenanceMode, maintenanceModeReason)))

	// XXX check that changes do not take effect while in maintenance mode

	// XXX check that reboot does take effect while in maintenance mode

	// XXX restore or reboot? Manual means restore is only choice.
	// XXX for testability maybe we should add a triggered "local" maintenance mode?
	// send command
	edgeNode.SetConfigItem("maintenance.mode", "disabled")
	tc.ConfigSync(edgeNode)
	t.Logf(utils.AddTimestamp(fmt.Sprintf("Wait for maintenance mode clear for %s",
		edgeNode.GetID())))

	tc.AddProcInfo(edgeNode, checkNoMaintenanceMode())

	tc.WaitForProc(int(timewait.Seconds()))

	maintenanceMode = tc.GetState(edgeNode).GetDinfo().MaintenanceMode
	maintenanceModeReason = tc.GetState(edgeNode).GetDinfo().MaintenanceModeReason
	t.Logf(utils.AddTimestamp(fmt.Sprintf("Wait clear done: maintenanceMode %t maintenanceModeReason %s",
		maintenanceMode, maintenanceModeReason)))
}
