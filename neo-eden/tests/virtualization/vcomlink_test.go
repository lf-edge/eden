package virtualization

import (
	"encoding/json"
	"fmt"

	"github.com/bloomberg/go-testgroup"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eve/pkg/pillar/vcom"
)

func getChannel(data []byte) (uint, error) {
	var msg vcom.Base
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return 0, err
	}

	return uint(msg.Channel), nil
}

func decodeTpmResponseEK(data []byte) (*vcom.TpmResponseEk, error) {
	tpmRes := new(vcom.TpmResponseEk)
	err := json.Unmarshal(data, tpmRes)
	if err != nil {
		return nil, err
	}

	return tpmRes, nil
}

func decodeError(data []byte) (*vcom.Error, error) {
	errMsg := new(vcom.Error)
	err := json.Unmarshal(data, errMsg)
	if err != nil {
		return nil, err
	}

	return errMsg, nil
}

func (grp *VirtualizationTests) TestVcomLinkTpmRequestEK(t *testgroup.T) {
	eveNode.LogTimeInfof("TestvComLinkTpmRequestEK started")
	defer eveNode.LogTimeInfof("TestvComLinkTpmRequestEK finished")

	if !eveNode.EveIsTpmEnabled() {
		t.Skip("TPM is not enabled, skipping test")
	}

	eveNode.LogTimeInfof("Checking if vComLink is reachable from a VM, deploying Ubuntu %s...", tk.Ubuntu2204)
	appName := tk.GetRandomAppName(projectName + "-")
	appName, err := eveNode.EveDeployUbuntu(tk.Ubuntu2204, appName, false)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to deploy app: %v", err)
	}
	err = waitForApp(appName, appWait, sshWait)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to wait for app: %v", err)
	}

	eveNode.LogTimeInfof("Copying test scripts to the vm...")
	err = eveNode.CopyTestScripts(appName, testScriptBasePath, &vComLinkTestScript)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to copy test scripts to the vm: %v", err)
	}
	command := fmt.Sprintf("python3 %s", eveNode.GetCopiedScriptPath("vcomlink_test.py"))
	out, err := eveNode.AppSSHExec(appName, command)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to communicate with host via vsock: %v", err)
	}

	eveNode.LogTimeInfof("Processing vComLink<->VM response...")
	channel, err := getChannel([]byte(out))
	if err != nil {
		eveNode.LogTimeFatalf("Failed to get channel from the output: %v", err)
	}
	if channel == uint(vcom.ChannelError) {
		errMsg, err := decodeError([]byte(out))
		if err != nil {
			eveNode.LogTimeFatalf("Failed to decode error message: %v", err)
		}
		eveNode.LogTimeFatalf("Received error message instead of EK: %s", errMsg.Error)
	}
	if channel != uint(vcom.ChannelTpm) {
		eveNode.LogTimeFatalf("Expected channel %d, got %d", vcom.ChannelTpm, channel)
	}

	eveNode.LogTimeInfof("Received expected TPM response from in the vm")
	tpmRes, err := decodeTpmResponseEK([]byte(out))
	if err != nil {
		eveNode.LogTimeFatalf("Failed to decode tpm response: %v", err)
	}
	if tpmRes.Ek == "" {
		eveNode.LogTimeFatalf("Received an empty EK from the vm")
	}
	eveNode.LogTimeInfof("Received expected EK in the TPM response")

	eveNode.LogTimeInfof("TestvComLinkTpmRequestEK passed")
}
