package escript

import (
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
		}
	}

	log.Info("testData directory: ", *testData)
	testscript.Run(t, testscript.Params{
		Dir:   *testData,
		Flags: flagsParsed,
	})
}

func TestMain(m *testing.M) {
	tests.TestArgsParse()

	result := m.Run()
	if result != 0 {
		tests.RunScenario(*failScenario, "", "", "", "", "")
	}
	os.Exit(result)
}
