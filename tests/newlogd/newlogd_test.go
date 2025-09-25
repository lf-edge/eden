package newlogd

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/controller/elog"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/utils"
)

var eveNode *tk.EveNode
var logT *testing.T

const (
	projectName = "newlogd"
	logTimeout  = 2 * time.Minute
)

func logFatalf(format string, args ...interface{}) {
	out := utils.AddTimestampf(format+"\n", args...)
	if logT != nil {
		logT.Fatal(out)
	} else {
		fmt.Print(out)
		os.Exit(1)
	}
}

func logInfof(format string, args ...interface{}) {
	out := utils.AddTimestampf(format+"\n", args...)
	if logT != nil {
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

func TestLogLevelsDifferent(t *testing.T) {
	logT = t
	steppy := &stepCounter{}

	logInfof("TestLogLevelsDifferent started")
	defer logInfof("TestLogLevelsDifferent finished")

	steppy.AnnounceNext("secure the initial config")
	if err := eveNode.GetConfig("/tmp/initial_config"); err != nil {
		logFatalf("Failed to get initial config: %v", err)
	}
	defer func() {
		logInfof("revert to the initial config")
		if err := eveNode.SetConfig("/tmp/initial_config"); err != nil {
			logFatalf("Failed to get back to the initial config: %v", err)
		}
	}()

	steppy.AnnounceNext("set log levels")
	localLogLevel := "debug"
	remoteLogLevel := "none"
	eveNode.UpdateNodeGlobalConfig(
		nil,
		map[string]string{
			"debug.default.loglevel":        localLogLevel,
			"debug.default.remote.loglevel": remoteLogLevel,
			"debug.syslog.loglevel":         localLogLevel,
			"debug.syslog.remote.loglevel":  remoteLogLevel,
			"debug.kernel.loglevel":         localLogLevel,
			"debug.kernel.remote.loglevel":  remoteLogLevel,
		},
	)
	if err := eveNode.WaitForConfigApplied(60 * time.Second); err != nil {
		logFatalf("Failed to wait for config to be applied: %v", err)
	}

	steppy.AnnounceNext("reboot EVE to generate fresh logs")
	if err := eveNode.EveRebootAndWait(180); err != nil {
		logFatalf("Failed to reboot EVE: %v", err)
	}

	steppy.AnnounceNext(fmt.Sprintf("check for undesired logs (timeout %v)", logTimeout))
	query := map[string]string{
		"severity": ".*",
	}
	if err := eveNode.FindLogOnAdam(query, elog.LogNew, logTimeout); err != nil {
		logInfof("No logs found, as expected")
	} else {
		logFatalf("Logs found, but they should not be present with the remote log level '%s'", remoteLogLevel)
	}
}
