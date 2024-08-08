package escript

import (
	"errors"
	"flag"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"testing"

	"github.com/lf-edge/eden/pkg/tests"
	"github.com/lf-edge/eden/tests/escript/go-internal/testscript"
)

var testData = flag.String("testdata", "testdata", "Test script directory")
var failScenario = flag.String("fail_scenario", "failScenario.txt", "Scenario that runs after a test fails")
var args = flag.String("args", "", "Flags to pass into test")

func TestEdenScripts(t *testing.T) {
	if _, err := os.Stat(*testData); os.IsNotExist(err) {
		log.Fatalf("can't find %s directory: %s\n", *testData, err)
	}

	flagsParsed := make(map[string]string)

	flags := strings.Split(strings.Trim(*args, "\""), ",")

	for _, el := range flags {
		fl := strings.TrimPrefix(el, "-")
		fl = strings.TrimPrefix(fl, "-")
		split := strings.SplitN(fl, "=", 2)
		if len(split) == 2 {
			flagsParsed[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
			// Also store the key=value argument into the environment variables so that
			// it can be used with EdenGetEnv inside Go templates.
			os.Setenv(split[0], split[1])
		}
	}

	log.Info("testData directory: ", *testData)
	testscript.Run(t, testscript.Params{
		Dir:       *testData,
		Flags:     flagsParsed,
		Condition: customConditions,
	})
}

// Function adds additional condition(s) for testscripts:
// - [env:<env-variable>] is satisfied if the environment variable has a non-empty string value assigned.
func customConditions(ts *testscript.TestScript, cond string) (bool, error) {
	if strings.HasPrefix(cond, "env:") {
		env := cond[len("env:"):]
		env = strings.TrimSpace(env)
		return ts.Getenv(env) != "", nil
	}
	return false, errors.New("unknown condition")
}

func TestMain(m *testing.M) {
	tests.TestArgsParse()

	result := m.Run()
	if result != 0 {
		tests.RunScenario(*failScenario, "", "", "", "", "")
	}
	os.Exit(result)
}
