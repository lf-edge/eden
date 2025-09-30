package vcom

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/utils"
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
		logT.Log(out)
	} else {
		fmt.Print(out)
	}
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

	// make sure python3-protobuf is installed
	logInfof("Installing python3-protobuf...")
	_, err = eveNode.AppSSHExec(appName, "sudo apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y python3-pip && pip3 install protobuf")
	if err != nil {
		logFatalf("Failed to install python3-protobuf: %v", err)
	}

	logInfof("Copying test scripts to the vm...")
	// dump the protobuf file
	err = dumpScript("messages_pb2.py", protobufFile)
	if err != nil {
		logFatalf("Failed to get path to messages_pb2.py: %v", err)
	}
	err = eveNode.AppSCPCopy(appName, "messages_pb2.py", "messages_pb2.py")
	if err != nil {
		logFatalf("Failed to copy messages_pb2.py to the vm: %v", err)
	}

	// dump the test script
	err = dumpScript("testvsock.py", testScript)
	if err != nil {
		logFatalf("Failed to get path to testvsock.py: %v", err)
	}
	err = eveNode.AppSCPCopy(appName, "testvsock.py", "testvsock.py")
	if err != nil {
		logFatalf("Failed to copy testvsock.py to the vm: %v", err)
	}

	// run the test script
	logInfof("Testing TPM Get Public Key via vComLink...")
	out, err := eveNode.AppSSHExec(appName, "python3 testvsock.py")
	if err != nil {
		logFatalf("Failed to communicate with host via vsock: %v", err)
	}

	// check the response
	logInfof("Processing vComLink<->VM response...")
	logInfof("Output: %s", out)
	// The script should return something like this, so lets just check for test passed
	// Testing TPM Get Public Key via VSOCK HTTP...
	// Sending TPM GetPub request via VSOCK (CID: 1, Port: 2000)...
	// TPM EK: 0001000b000300b20020837197674484...
	// TPM EK Algorithm: 1
	// TPM EK Attributes: FlagFixedTPM | FlagFixedParent | FlagSensitiveDataOrigin | FlagAdminWithPolicy | FlagRestricted | FlagDecrypt
	// Test passed!
	if !strings.Contains(string(out), "passed") {
		logFatalf("vComLink<->VM communication failed, output: %s", out)
	}

	logInfof("TestvComLinkTpmRequestEK passed")
}
