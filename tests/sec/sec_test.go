package sec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lf-edge/eden/pkg/defaults"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
)

const projectName = "security-test"
const appArmorStatus = "/sys/module/apparmor/parameters/enabled"

var eveNode *tk.EveNode

func TestMain(m *testing.M) {
	log.Println("Security Test Suite started")
	defer log.Println("Security Test Suite finished")

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	twoLevelsUp := filepath.Dir(filepath.Dir(currentPath))

	cfg, err := openevec.GetDefaultConfig(twoLevelsUp)
	if err != nil {
		log.Fatalf("Failed to generate default config %v\n", err)
	}

	if err = openevec.ConfigAdd(cfg, cfg.ConfigName, "", false); err != nil {
		log.Fatal(err)
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
