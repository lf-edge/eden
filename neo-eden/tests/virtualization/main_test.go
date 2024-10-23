package virtualization

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bloomberg/go-testgroup"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	log "github.com/sirupsen/logrus"
)

type VirtualizationTests struct{}

const testSuiteName = "Virtualization"
const projectName = testSuiteName + "-tests"

var eveNode *tk.EveNode
var group = &VirtualizationTests{}

// these are used by all tests
const (
	appWait            = 60 * 10 // 10 minutes
	sshWait            = 60 * 5  // 5 minutes
	nodeRebootWait     = 60 * 5  // 5 minutes
	aziotwait          = 30      // seconds
	testScriptBasePath = "/home/ubuntu/"
)

func TestMain(m *testing.M) {
	log.Printf("%s Test Suite started\n", testSuiteName)
	defer log.Printf("%s Test Suite finished\n", testSuiteName)

	node, err := tk.InitilizeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		log.Fatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	res := m.Run()
	os.Exit(res)
}

func TestVirtualization(t *testing.T) {
	if !strings.EqualFold(os.Getenv("WORKFLOW"), testSuiteName) {
		t.Skipf("Skip %s tests in non-%s workflow", testSuiteName, testSuiteName)
	}

	defer func() {
		for _, app := range eveNode.GetAppNames() {
			err := eveNode.AppStopAndRemove(app)
			if err != nil {
				eveNode.LogTimeFatalf("Failed to remove app %s: %v", app, err)
			}
		}
	}()

	eveNode.SetTesting(t)
	testgroup.RunSerially(t, group)
}

func waitForApp(appName string, appWait, sshWait uint) error {
	// wait for the app to show up in the list
	time.Sleep(5 * time.Second)

	// Wait for the app to start and ssh to be ready
	eveNode.LogTimeInfof("Waiting for app %s to start...", appName)
	err := eveNode.AppWaitForRunningState(appName, appWait)
	if err != nil {
		return fmt.Errorf("failed to wait for app to start: %v", err)
	}
	eveNode.LogTimeInfof("Waiting for ssh to be ready...")
	err = eveNode.AppWaitForSSH(appName, sshWait)
	if err != nil {
		return fmt.Errorf("failed to wait for ssh: %v", err)
	}

	eveNode.LogTimeInfof("SSH connection established")
	return nil
}
