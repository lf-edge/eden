package inventory

import (
	_ "embed"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/info"
)

var eveNode *tk.EveNode
var logT *testing.T

const (
	projectName = "inventory"
	infoTimeout = 2 * time.Minute // Timeout for log checks
)

func logFatalf(format string, args ...interface{}) {
	out := utils.AddTimestampf(format+"\n", args...)
	if logT != nil {
		logT.Helper()
		logT.Fatal(out)
	} else {
		fmt.Print(out)
		os.Exit(1)
	}
}

func logInfof(format string, args ...interface{}) {
	out := utils.AddTimestampf(format+"\n", args...)
	if logT != nil {
		logT.Helper()
		logT.Log(out)
	} else {
		fmt.Print(out)
	}
}

type stepCounter struct {
	count int
}

func (s *stepCounter) AnnounceNext(msg string) {
	s.count++
	logInfof("STEP %d: %s", s.count, msg)
}

func TestMain(m *testing.M) {
	logInfof("%s Test started", projectName)
	defer logInfof("%s Test finished", projectName)

	node, err := tk.InitializeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		logFatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	res := m.Run()
	os.Exit(res)
}

func TestHWInventory(t *testing.T) {
	// Initialize the the logger to use testing.T instance
	logT = t
	steppy := &stepCounter{}

	logInfof("TestHWInventory started")
	defer logInfof("TestHWInventory finished")

	steppy.AnnounceNext("check we received HW inventory capability on Adam")
	foundDeviceInfo, err := eveNode.GetInfoFromAdam(map[string]string{
		"ztype": ".*ZiDevice.*",
	}, einfo.InfoExist, infoTimeout)
	if err != nil {
		logFatalf("Failed to get ZiDevice info from Adam: %v", err)
	}
	var latestDeviceMsg *info.ZInfoDevice
	for _, infoMsg := range foundDeviceInfo {
		if infoMsg.Ztype == info.ZInfoTypes_ZiDevice {
			latestDeviceMsg = infoMsg.GetDinfo()
		}
	}
	logInfof("Found %d device info messages", len(foundDeviceInfo))
	if latestDeviceMsg == nil {
		logFatalf("Didn't find any device info message")
	}
	logInfof("Latest device info: %s", latestDeviceMsg)
	if latestDeviceMsg.GetOptionalCapabilities() == nil {
		logFatalf("Found a device info message without optional capabilities")
	}
	if !latestDeviceMsg.GetOptionalCapabilities().HwInventorySupport {
		logFatalf("HW inventory not supported according to the info message")
	}

	steppy.AnnounceNext("check we received the HW inventory message on Adam")
	foundInfoBeforeReboot, err := eveNode.GetInfoFromAdam(map[string]string{
		"ztype": ".*ZiHardware.*",
	}, einfo.InfoExist, infoTimeout)
	if err != nil {
		logFatalf("Failed to get ZiHardware info from Adam: %v", err)
	}
	var latestHWMsg *info.ZInfoHardware
	for _, infoMsg := range foundInfoBeforeReboot {
		if infoMsg.Ztype == info.ZInfoTypes_ZiHardware {
			latestHWMsg = infoMsg.GetHwinfo()
		}
	}
	logInfof("Found %d HW info messages (before reboot)", len(foundInfoBeforeReboot))
	if latestHWMsg == nil {
		logFatalf("Didn't find any HW info message")
	}
	logInfof("Latest HW message (before reboot): %s", latestHWMsg)
	firstHWInventory := latestHWMsg.GetInventory()
	if firstHWInventory == nil {
		logFatalf("Found a HW info message without inventory")
	}

	steppy.AnnounceNext("check some fields in the HW inventory message")

	steppy.AnnounceNext("reboot EVE node to generate new HW inventory")
	if err := eveNode.EveRebootAndWait(5 * 60); err != nil {
		logFatalf("Failed to reboot EVE node: %v", err)
	}
	time.Sleep(1 * time.Minute) // wait for a bit before checking logs

	steppy.AnnounceNext("check that HW inventory hasn't changed after reboot")
	foundInfoAfterReboot, err := eveNode.GetInfoFromAdam(map[string]string{
		"ztype": ".*ZiHardware.*",
	}, einfo.InfoExist, infoTimeout)
	if err != nil {
		logFatalf("Failed to get ZiHardware info from Adam: %v", err)
	}
	logInfof("Found %d messages after reboot (was %d)", len(foundInfoAfterReboot), len(foundInfoBeforeReboot))
	if len(foundInfoAfterReboot) <= len(foundInfoBeforeReboot) {
		logFatalf("Didn't find new HW info message after reboot")
	}
	latestHWMsg = nil
	for _, infoMsg := range foundInfoAfterReboot {
		if infoMsg.Ztype == info.ZInfoTypes_ZiHardware {
			latestHWMsg = infoMsg.GetHwinfo()
		}
	}
	if latestHWMsg == nil {
		logFatalf("Didn't find any HW info message")
	}
	logInfof("Latest HW message (after reboot): %s", latestHWMsg)
	secondHWInventory := latestHWMsg.GetInventory()
	if secondHWInventory.String() != firstHWInventory.String() {
		logFatalf("HW inventory changed after reboot: %s", cmp.Diff(firstHWInventory.String(), secondHWInventory.String()))
	}
}
