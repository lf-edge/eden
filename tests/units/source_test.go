package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/tests"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/spf13/viper"
)

//TestSource runs escript test for nested escripts
func TestSource(t *testing.T) {
	configName := defaults.DefaultContext
	configFile := utils.GetConfig(configName)
	if _, err := utils.LoadConfigFile(configFile); err != nil {
		t.Fatalf("error reading config: %s", err.Error())
	}
	curDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if curDir != "" {
			err := os.Chdir(curDir)
			if err != nil {
				t.Fatal(err)
			}
		}
	}()
	if err := os.Chdir(filepath.Join(viper.GetString("eden.tests"), "escript")); err != nil {
		t.Fatal(err)
	}
	tests.RunTest("eden.escript.test", []string{"-test.run", "TestEdenScripts/source"}, "", "", "", configFile, "debug")
}
