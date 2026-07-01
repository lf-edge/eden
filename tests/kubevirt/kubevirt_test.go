package kubevirt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/defaults"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/info"
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

// TestClusterStorageHealth verifies that EVE-k reports distributed-storage
// (Longhorn) health to the controller in the ZInfoKubeCluster info message,
// and that eden can read it back from Adam. This is the observable end of the
// disk-sizing guard added in lf-edge/eve#6108: KubeStorageInfo.Health is set
// from Longhorn's daemonset readiness and node-disk schedulability, so a
// too-small /persist (disk unschedulable, no replica placeable) surfaces as
// DEGRADED instead of the field going unreported.
//
// The assertion is deliberately that a *defined* health status round-trips to
// the controller (not that it reaches HEALTHY): reaching HEALTHY requires
// Longhorn to fully install and schedule replicas, which is exactly the slow,
// disk-size-sensitive path this work is about. A reported DEGRADED/FAILED here
// is a strong hint the node's disk is too small for Longhorn (EVE-k needs a
// 64 GiB disk; 32 GiB is not enough).
func TestClusterStorageHealth(t *testing.T) {
	log.Println("TestClusterStorageHealth started")
	defer log.Println("TestClusterStorageHealth finished")

	// Longhorn install is slow; poll until the cluster info carries a defined
	// storage health, up to a generous deadline.
	deadline := time.Now().Add(20 * time.Minute)
	lastHealth := info.ServiceStatus_SERVICE_STATUS_UNSPECIFIED
	for time.Now().Before(deadline) {
		msgs, err := eveNode.GetInfoFromAdam(map[string]string{
			"ztype": ".*ZiKubeCluster.*",
		}, einfo.InfoExist, 2*time.Minute)
		if err != nil {
			t.Logf("no ZiKubeCluster info from Adam yet: %v", err)
			time.Sleep(30 * time.Second)
			continue
		}
		// Take the storage health from the most recent cluster info message.
		var storage *info.KubeStorageInfo
		for _, m := range msgs {
			if m.Ztype == info.ZInfoTypes_ZiKubeCluster {
				if s := m.GetClusterInfo().GetStorage(); s != nil {
					storage = s
				}
			}
		}
		if storage == nil {
			t.Logf("ZiKubeCluster info present but no storage section yet")
			time.Sleep(30 * time.Second)
			continue
		}
		lastHealth = storage.GetHealth()
		t.Logf("cluster storage health reported on Adam: %s", lastHealth)
		if lastHealth != info.ServiceStatus_SERVICE_STATUS_UNSPECIFIED {
			if lastHealth != info.ServiceStatus_SERVICE_STATUS_HEALTHY {
				t.Logf("WARNING: storage health is %s, not HEALTHY - the "+
					"Longhorn disk may be unschedulable (disk too small for "+
					"EVE-k? needs 64 GiB)", lastHealth)
			}
			return
		}
		time.Sleep(30 * time.Second)
	}
	t.Fatalf("EVE-k never reported a defined cluster storage health to Adam "+
		"(last=%s); expected KubeStorageInfo.Health to be populated once "+
		"Longhorn is installed", lastHealth)
}
