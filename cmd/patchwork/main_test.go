package openevec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
)

func setup() {
	cfg := GetDefaultConfig(curPath)

	err := SetupEden(&cfg.ConfigName,
		filepath.Join(curPath, "eve-config-dir"),
		filepath.Join(cfg.Eden.Root, "eve-config-dir"),
		"", "", "", []string{}, false, false, cfg)

	if err != nil {
		panic("Setup Failed")
	}

	if err = StartEden(cfg, defaults.DefaultVBoxVMName, "", ""); err != nil {
		panic("Start failed")
	}

	if err = OnboardEve(cfg.Eve.CertsUUID); err != nil {
		panic("Onboard failed")
	}
}

func teardown() {
	cfg := GetDefaultConfig()
	StopEden(false, false, false, false, cfg.Eve.Remote, cfg.Eve.Pid, swtpmPidFile(cfg), cfg.Sdn.PidFile, cfg.Eve.DevModel, defautls.DefaultVBoxVMName)

	configDist, err := utils.DefaultEdenDir()
	if err != nil {
		panic()
	}

	err = EdenClean(cfg, cfg.ConfigName, configDist, defaults.DefaultVBoxVMName, false)
	if err != nil {
		panic()
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}
