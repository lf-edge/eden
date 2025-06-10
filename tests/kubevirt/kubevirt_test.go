package kubevirt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const projectName = "kubevirt-test"
const k3sNodeReadyStatusCmd = "eve exec kube /usr/bin/kubectl get node -o jsonpath='{.items[].status.conditions[?(@.type==\"Ready\")].status}'"
const hvTypeKubevirt = "kubevirt"

var eveNode *tk.EveNode

func TestMain(m *testing.M) {
	log.Println("Kubevirt Test Suite started")
	defer log.Println("Kubevirt Suite finished")

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	twoLevelsUp := filepath.Dir(filepath.Dir(currentPath))

	configPath := utils.GetConfig("default")
	cfg, err := openevec.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to get config %v\n", err)
	}

	if cfg.Eve.HV != hvTypeKubevirt {
		log.Fatalf("Incorrect eve.hv value %s, test only supports kubevirt", cfg.Eve.HV)
	}

	evec := openevec.CreateOpenEVEC(cfg)
	configDir := filepath.Join(twoLevelsUp, "eve-config-dir")
	if err := evec.SetupEden("config", configDir, "", "", "", []string{}, false, false); err != nil {
		log.Fatalf("Failed to setup Eden: %v", err)
	}
	if err := evec.StartEden(defaults.DefaultVBoxVMName, "", ""); err != nil {
		log.Fatalf("Start eden failed: %s", err)
	}
	if err := evec.OnboardEve(cfg.Eve.CertsUUID); err != nil {
		log.Fatalf("Eve onboard failed: %s", err)
	}

	node, err := tk.InitializeTestFromConfig(projectName, cfg, tk.WithControllerVerbosity("debug"))
	if err != nil {
		log.Fatalf("Failed to initialize test: %v", err)
	}

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
