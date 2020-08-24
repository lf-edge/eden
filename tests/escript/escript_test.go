package escript

import (
	"flag"
	"github.com/lf-edge/eden/tests/escript/go-internal/testscript"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"testing"
)

var testData = flag.String("testdata", "testdata", "Test script directory")
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
