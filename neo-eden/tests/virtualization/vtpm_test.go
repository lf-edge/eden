package virtualization

import (
	"github.com/bloomberg/go-testgroup"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
)

func (grp *VirtualizationTests) TestVtpmIsStatePreservation(_ *testgroup.T) {
	eveNode.LogTimeInfof("TestVtpmIsStatePreservation started")
	defer eveNode.LogTimeInfof("TestVtpmIsStatePreservation finished")

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
	err = eveNode.CopyTestScripts(appName, testScriptBasePath, &vTPMTestScripts)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to copy test scripts to the vm: %v", err)
	}

	eveNode.LogTimeInfof("Executing the script to create the necessary TPM keys...")
	out, err := eveNode.AppSSHExec(appName, eveNode.GetCopiedScriptPath("make_tpm_keys.sh"))
	if err != nil {
		eveNode.LogTimeFatalf("Failed to execute make_tpm_keys.sh script in VM: %v,\nOutput : %s", err, out)
	}

	eveNode.LogTimeInfof("Rebooting the EVE node and check the vTPM state is preserved...")
	err = eveNode.EveRebootAndWait(nodeRebootWait)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to reboot EVE: %v", err)
	}
	err = waitForApp(appName, appWait, sshWait)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to wait for app: %v", err)
	}

	eveNode.LogTimeInfof("Rebooting the script to check the vTPM state is preserved...")
	out, err = eveNode.AppSSHExec(appName, eveNode.GetCopiedScriptPath("check_tpm_keys.sh"))
	if err != nil {
		eveNode.LogTimeFatalf("Failed to execute check_tpm_keys.sh script in VM: %v,\nOutput : %s", err, out)
	}

	eveNode.LogTimeInfof("TestVtpmIsStatePreservation passed")
}
