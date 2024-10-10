package eve

import (
	"strings"

	"github.com/bloomberg/go-testgroup"
)

func (grp *SmokeTests) TestVcomLinkTpmRequestEK(t *testgroup.T) {
	eveNode.LogTimeInfof("TestVcomLinkTpmRequestEK started")
	defer eveNode.LogTimeInfof("TestVcomLinkTpmRequestEK finished")

	t.Log("Checking if vcomlink is running on EVE")
	stat, err := eveNode.EveRunCommand("eve exec pillar ss -l --vsock")
	if err != nil {
		eveNode.LogTimeFatalf("Failed to check if vcomlink is running: %v", err)
	}
	// vcomlink listens on port 2000 and host cid is 2.
	// this is hacky way to check it is running, but it works ¯\_(ツ)_/¯
	if !strings.Contains(string(stat), "2:2000") {
		eveNode.LogTimeFatalf("vcomlink is not running, ss output :\n%s", stat)
	}
}
