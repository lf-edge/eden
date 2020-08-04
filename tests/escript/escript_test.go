package escript

import (
	"flag"
	"github.com/lf-edge/eden/tests/escript/go-internal/testscript"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

var testData = flag.String("testdata", "testdata", "Test script directory")

func TestEdenScripts(t *testing.T) {
	if _, err := os.Stat(*testData); os.IsNotExist(err) {
		log.Fatalf("can't find %s directory: %s\n", *testData, err)
	}

	log.Info("testData directory: ", *testData)
	testscript.Run(t, testscript.Params{
		Dir: *testData,
	})
}
