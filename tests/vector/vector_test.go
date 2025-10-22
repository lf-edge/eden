package vector

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/controller/elog"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/utils"
)

var eveNode *tk.EveNode
var logT *testing.T

// we have to embed the config files since the test can be run
// from any directory and it's hard to resolve relative paths

//go:embed testdata/faulty.yaml
var faultyConfig []byte

//go:embed testdata/filter.yaml
var filterConfig []byte

const (
	projectName = "vector"
	logTimeout  = 2 * time.Minute // Timeout for log checks
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

type stepCounter struct {
	count int
}

func (s *stepCounter) AnnounceNext(msg string) {
	s.count++
	logInfof("STEP %d: %s", s.count, msg)
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
	steppy := &stepCounter{}

	logInfof("TestFaultyConfig started")
	defer logInfof("TestFaultyConfig finished")

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

	steppy.AnnounceNext("check vector is running")
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

	steppy.AnnounceNext("get hash of the default config")
	cmd = "sha256sum /persist/vector/config/vector.yaml"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	hashDefaultConfig := strings.Split(strings.TrimSpace(string(out)), " ")[0]

	steppy.AnnounceNext("set faulty vector config")
	hashFaultyConfig := fmt.Sprintf("%x", sha256.Sum256(faultyConfig))
	eveNode.UpdateNodeGlobalConfig(
		nil,
		map[string]string{
			"vector.config": base64.StdEncoding.EncodeToString(faultyConfig),
		},
	)

	steppy.AnnounceNext("wait for the faulty config to be applied")
	if err := eveNode.WaitForConfigApplied(60 * time.Second); err != nil {
		logFatalf("Failed to wait for the faulty config to be applied: %v", err)
	}

	steppy.AnnounceNext("check that vector is still running")
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

	steppy.AnnounceNext("check that vector is using the faulty config")
	cmd = "sha256sum /persist/vector/config/vector.yaml"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	hashConfig := strings.Split(strings.TrimSpace(string(out)), " ")[0]
	switch hashConfig {
	case hashDefaultConfig:
		logInfof("Still using valid config")
	case hashFaultyConfig:
		logFatalf("Faulty config was applied!")
	default:
		logFatalf("Unexpected config hash: expected %s or %s, got %s", hashDefaultConfig, hashFaultyConfig, hashConfig)
	}
}

func TestEmptyConfig(t *testing.T) {
	// Initialize the the logger to use testing.T instance
	logT = t
	steppy := &stepCounter{}

	logInfof("TestEmptyConfig started")
	defer logInfof("TestEmptyConfig finished")

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

	steppy.AnnounceNext("check vector is running")
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

	steppy.AnnounceNext("get hash of the current config")
	cmd = "sha256sum /persist/vector/config/vector.yaml"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	hashCurrentConfig := strings.Split(strings.TrimSpace(string(out)), " ")[0]

	steppy.AnnounceNext("set empty vector config")
	eveNode.UpdateNodeGlobalConfig(
		nil,
		map[string]string{
			"vector.config": "",
		},
	)

	steppy.AnnounceNext("wait for the new config to be applied")
	// Wait for the new config to be applied
	if err := eveNode.WaitForConfigApplied(60 * time.Second); err != nil {
		logFatalf("Failed to wait for the new config to be applied: %v", err)
	}

	steppy.AnnounceNext("check that vector is still running")
	cmd = "eve status | grep vector"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	if strings.Contains(string(out), "RUNNING") {
		logInfof("Vector is still running on EVE node despite the empty config")
	} else {
		logFatalf("Vector is not running on EVE node after applying the empty config")
	}

	steppy.AnnounceNext("check vector is using the default config")
	cmd = "eve exec vector sha256sum /etc/vector/vector.yaml"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	hashDefaultConfig := strings.Split(strings.TrimSpace(string(out)), " ")[0]

	cmd = "sha256sum /persist/vector/config/vector.yaml"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	hashConfig := strings.Split(strings.TrimSpace(string(out)), " ")[0]

	switch hashConfig {
	case hashDefaultConfig:
		logInfof("Using default config")
	case hashCurrentConfig:
		logFatalf("Still using the current config, not the default one!")
	default:
		logFatalf("Unexpected config hash: expected %s or %s, got %s", hashDefaultConfig, hashCurrentConfig, hashConfig)
	}
}

func TestWorkingConfig(t *testing.T) {
	// Initialize the the logger to use testing.T instance
	logT = t
	steppy := &stepCounter{}

	logInfof("TestWorkingConfig started")
	defer logInfof("TestWorkingConfig finished")

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

	steppy.AnnounceNext("check vector is running")
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

	steppy.AnnounceNext("set filter vector config")
	eveNode.UpdateNodeGlobalConfig(
		nil,
		map[string]string{
			"vector.config": base64.StdEncoding.EncodeToString(filterConfig),
		},
	)
	if err := eveNode.WaitForConfigApplied(60 * time.Second); err != nil {
		logFatalf("Failed to wait for the config to be applied: %v", err)
	}

	steppy.AnnounceNext("give vector some time to apply the filter config")
	time.Sleep(30 * time.Second)

	steppy.AnnounceNext("check the filter config was applied")
	hashFilterConfig := fmt.Sprintf("%x", sha256.Sum256(filterConfig))
	cmd = "sha256sum /persist/vector/config/vector.yaml"
	out, err = eveNode.EveRunCommand(cmd)
	if err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}
	hashConfig := strings.Split(strings.TrimSpace(string(out)), " ")[0]
	switch hashConfig {
	case hashFilterConfig:
		logInfof("Config got applied correctly")
	default:
		logFatalf("Unexpected config hash: expected %s, got %s", hashFilterConfig, hashConfig)
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		query := map[string]string{
			"content": "Received disconnect from.*",
		}
		steppy.AnnounceNext(fmt.Sprintf("check that %q is still present (timeout %v)", query["content"], logTimeout))
		if err := eveNode.FindLogOnAdam(query, elog.LogNew, logTimeout); err == nil {
			logInfof("%q found, as expected", query["content"])
		} else {
			logFatalf("%q not found, but they should be present with the filter config", query["content"])
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		query := map[string]string{
			"content": "Disconnected from user.*",
		}
		steppy.AnnounceNext(fmt.Sprintf("check that %q doesn't appear anymore (timeout %v)", query["content"], logTimeout))
		if err := eveNode.FindLogOnAdam(query, elog.LogNew, logTimeout); err != nil {
			logInfof("%q not found, as expected", query["content"])
		} else {
			logFatalf("%q found, but it should not be present with the filter config", query["content"])
		}
	}()

	steppy.AnnounceNext("generate logs from debug container")
	cmd = "exit"
	if _, err := eveNode.EveRunCommand(cmd); err != nil {
		logFatalf("Failed to run ssh '%s': %v", cmd, err)
	}

	wg.Wait()
}
