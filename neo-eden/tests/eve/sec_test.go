package eve

import (
	"strings"

	"github.com/bloomberg/go-testgroup"
	log "github.com/sirupsen/logrus"
)

func (grp *SmokeTests) TestAppArmorEnabled(t *testgroup.T) {
	log.Println("TestAppArmorEnabled started")
	defer log.Println("TestAppArmorEnabled finished")

	appArmorStatus := "/sys/module/apparmor/parameters/enabled"
	out, err := eveNode.EveReadFile(appArmorStatus)
	if err != nil {
		t.Fatal(err)
	}

	exits := strings.TrimSpace(string(out))
	if exits != "Y" {
		t.Fatal("AppArmor is not enabled")
	}
}
