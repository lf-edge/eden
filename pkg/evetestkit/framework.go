package evetestkit

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
)

// SetupTestSuite initializes the test environment for a test suite
// Call this from TestMain in each test package
func SetupTestSuite(projectName string, configPath string, opts ...openevec.ConfigOption) (*EveNode, func(), error) {
	flag.Parse()

	builder := openevec.GetDefaultConfig(configPath)

	if len(builder.Err) > 0 {
		return nil, nil, errors.Join(builder.Err...)
	}

	for _, opt := range opts {
		opt(builder)
	}

	cfg := builder.Args

	if err := openevec.ConfigAdd(cfg, cfg.ConfigName, "", false); err != nil {
		return nil, nil, err
	}

	evec := openevec.CreateOpenEVEC(cfg)
	configDir := filepath.Join(configPath, "eve-config-dir")
	if err := evec.SetupEden("config", configDir, "", "", "", false, false); err != nil {
		return nil, nil, fmt.Errorf("Failed to setup Eden: %v", err)
	}
	if err := evec.StartEden(defaults.DefaultVBoxVMName, "", ""); err != nil {
		return nil, nil, fmt.Errorf("Start eden failed: %s", err)
	}
	if err := evec.OnboardEve(cfg.Eve.CertsUUID); err != nil {
		return nil, nil, fmt.Errorf("Eve onboard failed: %s", err)
	}

	node, err := InitializeTestFromConfig(
		projectName, cfg, WithControllerVerbosity("debug"))
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to initialize test: %v", err)
	}

	cleanup := func() {
		fmt.Printf("Cleaning up test suite: %s", projectName)
	}

	return node, cleanup, nil
}
