package sec_test

import (
	"os"
	"strings"
	"testing"

	tk "github.com/lf-edge/eden/pkg/evetestkit"
	log "github.com/sirupsen/logrus"
)

const projectName = "security-test"
const appArmorStatus = "/sys/module/apparmor/parameters/enabled"

var eveNode *tk.EveNode

func TestMain(m *testing.M) {
	log.Println("Security Test Suite started")
	defer log.Println("Security Test Suite finished")

	node, err := tk.InitilizeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		log.Fatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	res := m.Run()
	os.Exit(res)
}

func TestAppArmorEnabled(t *testing.T) {
	log.Println("TestAppArmorEnabled started")
	defer log.Println("TestAppArmorEnabled finished")
	t.Parallel()

	out, err := eveNode.EveReadFile(appArmorStatus)
	if err != nil {
		t.Fatal(err)
	}

	exits := strings.TrimSpace(string(out))
	if exits != "Y" {
		t.Fatal("AppArmor is not enabled")
	}
}
