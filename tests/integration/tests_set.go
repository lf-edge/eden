package integration

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

type EdenTestArgs map[string]string
type EdenTestFunc func(t *testing.T, name string, args EdenTestArgs) string

// EdenTest is a description of async test in tests set
type EdenTest struct {
	Name   string
	Test   EdenTestFunc
	Args   EdenTestArgs
	Result string // Regexp pattern
}

type EdenTestSets map[string][]EdenTest

// ETests is the main object for tests sets 
var ETests EdenTestSets = EdenTestSets{}

func runSubTest(t *testing.T, name string, test EdenTestFunc, args EdenTestArgs, result string, cntx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-cntx.Done():
			return
		default:
			res := test(t, name, args)
			matched, err := regexp.Match(result, []byte(res))
			if err != nil {
				t.Errorf("%s segexp '%s' for '%s' matching error: %s", name, res, result, err)
			}
			if !matched {
				t.Errorf("%s got '%v'; want '%v'", name, res, result)
			}
			cancel()
			return
		}
	}
}

// runTestSets start Go routines for every test in tests set ETests map
// by a subtest name
func runTestSets(t *testing.T) {
        ctx, cancel := context.WithCancel(context.Background())
        defer cancel() // cancel when we are finished tasks

	name := strings.Split(t.Name(),"/")[1]
	for _, tc := range ETests[name] {
		fmt.Printf("Running %s.%s\n", t.Name(), tc.Name)
		go runSubTest(t, tc.Name, tc.Test, tc.Args, tc.Result, ctx, cancel)
	}
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

