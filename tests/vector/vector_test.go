package vector

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/utils"
)

var eveNode *tk.EveNode
var logT *testing.T

const (
	projectName = "vector"
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

func TestMain(m *testing.M) {
	logInfof("%s Test started", projectName)
	defer logInfof("%s Test finished", projectName)

	node, err := tk.InitializeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		logFatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	res := m.Run()
	os.Exit(res)
}

func TestFaultyConfig(t *testing.T) {
	// Initialize the the logger to use testing.T instance
	logT = t

	logInfof("TestFaultyConfig started")
	defer logInfof("TestFaultyConfig finished")

	step := 1

	logInfof("secure the initial config")
	if err := eveNode.GetConfig("/tmp/initial_config"); err != nil {
		logFatalf("Failed to get initial config: %v", err)
	}

	defer func() {
		logInfof("revert to the initial config")
		if err := eveNode.SetConfig("/tmp/initial_config"); err != nil {
			logFatalf("Failed to get back to the initial config: %v", err)
		}
	}()

	logInfof("STEP %d: check vector is running", step)
	step++

	cmd := "eve status | grep vector"
	out, err := eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	if strings.Contains(string(out), "RUNNING") {
		logInfof("Vector is running on EVE node")
	} else {
		logFatalf("Vector is not running on EVE node")
	}

	logInfof("STEP %d: get hash of the valid config", step)
	step++

	cmd = "sha256sum /persist/vector/config/vector.yaml"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	hashValidConfig := strings.Split(strings.TrimSpace(string(out)), " ")[0]

	logInfof("STEP %d: set faulty vector config", step)
	step++

	// Read the faulty config from the file
	faultyConfig, err := os.ReadFile("testdata/faulty.yaml")
	if err != nil {
		logFatalf("Failed to read vector config file: %v", err)
	}
	hashFaultyConfig := fmt.Sprintf("%x", sha256.Sum256(faultyConfig))
	eveNode.UpdateNodeGlobalConfig(
		nil,
		map[string]string{
			"vector.config": base64.StdEncoding.EncodeToString(faultyConfig),
		},
	)

	logInfof("STEP %d: wait for the faulty config to be applied", step)
	step++

	// Wait for the faulty config to be applied
	if err := eveNode.WaitForConfigApplied(60 * time.Second); err != nil {
		logFatalf("Failed to wait for the faulty config to be applied: %v", err)
	}

	logInfof("STEP %d: check that vector is still running", step)
	step++

	cmd = "eve status | grep vector"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	if strings.Contains(string(out), "RUNNING") {
		logInfof("Vector is still running on EVE node despite the faulty config")
	} else {
		logFatalf("Vector is not running on EVE node after applying the faulty config")
	}

	logInfof("STEP %d: check the config hashes", step)
	step++

	cmd = "sha256sum /persist/vector/config/vector.yaml"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	hashConfig := strings.Split(strings.TrimSpace(string(out)), " ")[0]
	if hashConfig == hashFaultyConfig {
		logInfof("Faulty config was applied")
	} else {
		logFatalf("Faulty config was NOT applied. Expected hash %s. Actual hash %s", hashFaultyConfig, hashConfig)
	}

	cmd = "sha256sum /persist/vector/config/vector.yaml.bak"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	hashBackupConfig := strings.Split(strings.TrimSpace(string(out)), " ")[0]
	if hashBackupConfig == hashValidConfig {
		logInfof("Valid config was backed up")
	} else {
		logFatalf("Valid config was NOT backed up. Expected hash %s. Actual hash %s", hashValidConfig, hashBackupConfig)
	}
}
