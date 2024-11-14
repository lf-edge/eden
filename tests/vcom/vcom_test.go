package vcom

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/pkg/pillar/vcom"
)

var eveNode *tk.EveNode
var logT *testing.T

const (
	sshPort     = "8027"
	appLink     = "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img"
	projectName = "vcomlink"
	appWait     = 60 * 30
	sshWait     = 60 * 15
)

func logFatalf(format string, args ...interface{}) {
	out := utils.AddTimestampf(format+"\n", args...)
	if logT != nil {
		logT.Fatal(out)
	} else {
		fmt.Print(out)
		os.Exit(1)
	}
}

func logInfof(format string, args ...interface{}) {
	out := utils.AddTimestampf(format+"\n", args...)
	if logT != nil {
		logT.Logf(out)
	} else {
		fmt.Print(out)
	}
}

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

func dumpScript(name, content string) error {
	return os.WriteFile(name, []byte(content), 0644)
}

func TestMain(m *testing.M) {
	logInfof("VCOM Test started")
	defer logInfof("VCOM Test finished")

	node, err := tk.InitializeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		logFatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	res := m.Run()
	os.Exit(res)
}

func TestVcomLinkTpmRequestEK(t *testing.T) {
	// Initialize the the logger to use testing.T instance
	logT = t

	logInfof("TestvComLinkTpmRequestEK started")
	defer logInfof("TestvComLinkTpmRequestEK finished")

	if !eveNode.EveIsTpmEnabled() {
		t.Skip("TPM is not enabled, skipping test")
	}

	logInfof("Checking if vComLink is running on EVE...")
	stat, err := eveNode.EveRunCommand("eve exec pillar ss -l --vsock")
	if err != nil {
		logFatalf("Failed to check if vComLink is running: %v", err)
	}
	// vComLink listens on port 2000 and host cid is 2.
	// this is hacky way to check it is running, but it works ¯\_(ツ)_/¯
	if !strings.Contains(string(stat), "2:2000") {
		logFatalf("vComLink is not running, ss output :\n%s", stat)
	}
	logInfof("vComLink agent is running")

	fileName := path.Base(appLink)
	baseName := strings.TrimSuffix(fileName, path.Ext(fileName))
	logInfof("Checking if vComLink is reachable from a VM, deploying %s...", baseName)

	appName := tk.GetRandomAppName(projectName + "-")
	pubPorts := []string{sshPort + ":22"}
	pc := tk.GetDefaultVMConfig(appName, tk.AppDefaultCloudConfig, pubPorts)
	err = eveNode.EveDeployApp(appLink, true, pc, tk.WithSSH(tk.AppDefaultSSHUser, tk.AppDefaultSSHPass, sshPort))
	if err != nil {
		logFatalf("Failed to deploy app: %v", err)
	}
	defer func() {
		err = eveNode.AppStopAndRemove(appName)
		if err != nil {
			logInfof("Failed to stop and remove app: %v", err)
		}
	}()
	// wait for the app to show up in the list
	time.Sleep(10 * time.Second)
	// wait 5 minutes for the app to start
	logInfof("Waiting for app %s to start...", appName)
	err = eveNode.AppWaitForRunningState(appName, appWait)
	if err != nil {
		logFatalf("Failed to wait for app to start: %v", err)
	}

	logInfof("Waiting for ssh to be ready...")
	err = eveNode.AppWaitForSSH(appName, sshWait)
	if err != nil {
		logFatalf("Failed to wait for ssh: %v", err)
	}
	logInfof("SSH connection with VM established.")

	logInfof("Copying test scripts to the vm...")
	err = dumpScript("testvsock.py", testScript)
	if err != nil {
		logFatalf("Failed to get path to testvsock.py: %v", err)
	}
	err = eveNode.AppSCPCopy(appName, "testvsock.py", "testvsock.py")
	if err != nil {
		logFatalf("Failed to copy testvsock.py to the vm: %v", err)
	}
	out, err := eveNode.AppSSHExec(appName, "python3 testvsock.py")
	if err != nil {
		logFatalf("Failed to communicate with host via vsock: %v", err)
	}

	logInfof("Processing vComLink<->VM response...")
	channel, err := getChannel([]byte(out))
	if err != nil {
		logFatalf("Failed to get channel from the output: %v", err)
	}
	if channel == uint(vcom.ChannelError) {
		errMsg, err := decodeError([]byte(out))
		if err != nil {
			logFatalf("Failed to decode error message: %v", err)
		}
		logFatalf("Received error message instead of EK: %s", errMsg.Error)
	}
	if channel != uint(vcom.ChannelTpm) {
		logFatalf("Expected channel %d, got %d", vcom.ChannelTpm, channel)
	}

	logInfof("Received expected TPM response from in the vm")
	tpmRes, err := decodeTpmResponseEK([]byte(out))
	if err != nil {
		logFatalf("Failed to decode tpm response: %v", err)
	}
	if tpmRes.Ek == "" {
		logFatalf("Received an empty EK from the vm")
	}
	logInfof("Received expected EK in the TPM response")

	logInfof("TestvComLinkTpmRequestEK passed")
}
