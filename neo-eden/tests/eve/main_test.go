package eve

import (
	"os"
	"strings"
	"testing"

	"github.com/bloomberg/go-testgroup"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	log "github.com/sirupsen/logrus"
)

type SmokeTests struct{}

const testSuiteName = "Smoke"
const projectName = testSuiteName + "-tests"

var eveNode *tk.EveNode
var group = &SmokeTests{}

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

func TestSmoke(t *testing.T) {
	if !strings.EqualFold(os.Getenv("WORKFLOW"), testSuiteName) {
		t.Skipf("Skip %s tests in non-%s workflow", testSuiteName, testSuiteName)
	}

	eveNode.SetTesting(t)
	testgroup.RunSerially(t, group)
}
