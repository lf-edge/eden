package sec_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
)

const projectName = "security-test"
const appArmorStatus = "/sys/module/apparmor/parameters/enabled"

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

	log.Println("Security Test Suite started")
	defer log.Println("Security Test Suite finished")

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
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

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
