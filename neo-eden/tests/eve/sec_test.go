package eve

import (
	"strings"

	"github.com/bloomberg/go-testgroup"
)

func (grp *SmokeTests) TestAppArmorEnabled(t *testgroup.T) {
	eveNode.LogTimeInfof("TestAppArmorEnabled started")
	defer eveNode.LogTimeInfof("TestAppArmorEnabled finished")

	appArmorStatus := "/sys/module/apparmor/parameters/enabled"
	out, err := eveNode.EveReadFile(appArmorStatus)
	if err != nil {
		t.Fatal(err)
	}

	exits := strings.TrimSpace(string(out))
	if exits != "Y" {
		eveNode.LogTimeFatalf("AppArmor is not enabled")
	}
}
