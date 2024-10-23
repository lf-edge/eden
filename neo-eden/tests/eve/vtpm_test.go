package eve

import (
	"strings"

	"github.com/bloomberg/go-testgroup"
)

// TestVtpmIsRunningOnEVE checks if the vTPM process is running on the EVE node,
// it does this by checking if the vTPM control socket is open and the vTPM process
// is listening on it.
func (grp *SmokeTests) TestVtpmIsRunningOnEVE(_ *testgroup.T) {
	eveNode.LogTimeInfof("TestVtpmIsRunningOnEVE started")
	defer eveNode.LogTimeInfof("TestVtpmIsRunningOnEVE finished")

	// find the vTPM control socket and see if the vTPM process is listening on it.
	command := "lsof -U | grep $(cat /proc/net/unix | grep vtpm | awk '{print $7}')"
	out, err := eveNode.EveRunCommand(command)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to check if vTPM is running on EVE: %v", err)
	}

	if len(out) == 0 || !strings.Contains(string(out), "vtpm") {
		eveNode.LogTimeFatalf("vTPM is not running on EVE : %s", out)
	}
}
