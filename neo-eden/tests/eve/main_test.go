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

var eveNode *tk.EveNode

const projectName = "eve-tests"

func TestMain(m *testing.M) {
	log.Println("EVE Test Suite started")
	defer log.Println("EVE Test Suite finished")

	node, err := tk.InitilizeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		log.Fatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	res := m.Run()
	os.Exit(res)
}

func TestSmoke(t *testing.T) {
	if strings.ToLower(os.Getenv("WORKFLOW")) != "smoke" {
		t.Skip("Skip smoke tests in workflow mode")
	}

	testgroup.RunSerially(t, &SmokeTests{})
}
