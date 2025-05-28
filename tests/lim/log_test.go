package lim

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
	projectName  = "lim"
	logToLookFor = "Disconnected"
	logTimeout   = 30 * time.Second // 30 sec should be enough since it only takes 10 sec to upload logs with fast upload
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

func TestMain(m *testing.M) {
	logInfof("%s Test started", projectName)
	defer logInfof("%s Test finished", projectName)

	node, err := tk.InitializeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		logFatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	tc = eveNode.GetTestContext()
	res := m.Run()
	os.Exit(res)
}

func TestLogInGolang(t *testing.T) {
	// Initialize the the logger to use testing.T instance
	logT = t

	logInfof("TestLogInGolang started")
	defer logInfof("TestLogInGolang finished")

	logInfof("secure the initial config")
	if err := eveNode.GetConfig("/tmp/initial_config"); err != nil {
		logFatalf("Failed to get initial config: %v", err)
	}

	defer func() {
		logInfof("revert to the initial config")
		if err := eveNode.SetConfig("/tmp/initial_config"); err != nil {
			logFatalf("Failed to get back to the initial config: %v", err)
		}
	}()

	logInfof("STEP 1: set log levels")
	eveNode.UpdateNodeGlobalConfig(
		nil,
		map[string]string{
			"debug.default.loglevel":        "info",
			"debug.default.remote.loglevel": "info",
		},
	)

	logInfof("STEP 2: wait for the log levels to be applied and the old logs to be sent")
	// TODO: change this to use the metric of when the last config was applied / changed
	// the waiting period includes the time for EVE to get & parse the config
	// as well as the time for the logs that were written to disk before the config was applied to be sent
	time.Sleep(60 * time.Second)

	logInfof("STEP 3: open and close SSH connection to EVE to generate some logs")
	_, err := eveNode.EveRunCommand("exit")
	if err != nil {
		logFatalf("Failed to run 'ssh exit': %v", err)
	}

	logInfof("STEP 4: check the logs")
	query := map[string]string{
		"content": ".*Disconnected.*",
	}
	if err := eveNode.FindLogOnAdam(query, elog.LogNew, logTimeout); err != nil {
		logFatalf("Failed to get logs from adam: %v", err)
	}
}
