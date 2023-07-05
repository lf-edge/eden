package patchwork

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	ec "github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
)

func setup(cfg *ec.EdenSetupArgs) error {

	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}

	// We need eden dir since we are storing certificates there
	if _, err := os.Stat(edenDir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(edenDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	err = ec.SetupEden(cfg.ConfigName,
		filepath.Join(cfg.Eden.Root, "eve-config-dir"),
		"", "", "", []string{}, false, false, *cfg)

	if err != nil {
		return err
	}

	if err = ec.StartEden(cfg, defaults.DefaultVBoxVMName, "", ""); err != nil {
		return err
	}

	if err = ec.OnboardEve(cfg.Eve.CertsUUID); err != nil {
		return err
	}
	return nil
}

func teardown(cfg *ec.EdenSetupArgs) error {
	eden.StopEden(false, false, false, false, cfg.Eve.Remote, cfg.Eve.Pid, ec.SwtpmPidFile(cfg), cfg.Sdn.PidFile, cfg.Eve.DevModel, defaults.DefaultVBoxVMName)

	configDist, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}

	err = ec.EdenClean(*cfg, cfg.ConfigName, configDist, defaults.DefaultVBoxVMName, false)
	if err != nil {
		return err
	}
	return nil
}

func TestMain(m *testing.M) {
	curPath, err := os.Getwd()
	if err != nil {
		// Getwd failed
		os.Exit(1)
	}
	cfg := ec.GetDefaultConfig(curPath)

	if err = setup(cfg); err != nil {
		fmt.Printf("Error setup %v", err)
		os.Exit(1)
	}
	code := m.Run()
	if err = teardown(cfg); err != nil {
		fmt.Printf("Error setup %v", err)
		os.Exit(1)
	}
	os.Exit(code)
}
