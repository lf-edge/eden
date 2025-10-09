package kubevirt_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
)

const projectName = "kubevirt-test"
const k3sNodeReadyStatusCmd = "eve exec kube /usr/bin/kubectl get node -o jsonpath='{.items[].status.conditions[?(@.type==\"Ready\")].status}'"
const hvTypeKubevirt = "kubevirt"

var eveNode *tk.EveNode
var (
	// Global flags - parsed once across all test packages
	fileSystem  = flag.String("filesystem", "ext4", "File system type (ext4, zfs)")
	eveImage    = flag.String("eve-image", "", "Path to EVE OS image")
	eveLogLevel = flag.String("eve-log-level", "info", "EVE log level (debug, info, warn, error)")
	requireVirt = flag.Bool("require-virt", false, "Require HW-assisted virtualization support")
)

func TestMain(m *testing.M) {
	flag.Parse()

	log.Println("Kubevirt Test Suite started")
	defer log.Println("Kubevirt Suite finished")

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	twoLevelsUp := filepath.Dir(filepath.Dir(currentPath))

	node, cleanup, err := tk.SetupTestSuite(
		projectName,
		twoLevelsUp,
		openevec.WithAccelerator(*requireVirt, []string{}),
		openevec.WithEVEImage(*eveImage),
		openevec.WithFilesystem(*fileSystem),
		openevec.WithHypervisor(hvTypeKubevirt),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	eveNode = node
	res := m.Run()
	os.Exit(res)
}

// TestNodeReady to verify the kubernetes control plane becomes ready.
func TestNodeReady(t *testing.T) {
	log.Println("TestNodeReady started")
	defer log.Println("TestNodeReady finished")

	maxTries := 20 // 5 minutes of once every 15sec
	attempt := 1

	for attempt < maxTries {
		out, err := eveNode.EveRunCommand(k3sNodeReadyStatusCmd)
		if err == nil {
			condition := strings.TrimSpace(string(out))
			if condition == "True" {
				t.Logf("k3s node ready")
				return
			}
		}

		t.Logf("Warn: node ready command returned err:%v out:%s", err, string(out))
		time.Sleep(15 * time.Second)
		attempt++
	}

	t.Fatalf("k3s node did not become ready")
}
