//go:build !go1.18
// +build !go1.18

package testscript

import (
	"io"
	"testing"
)

type nopTestDeps struct{}

func (nopTestDeps) SetPanicOnExit0(_ bool) {}

func (nopTestDeps) MatchString(_, _ string) (result bool, err error) {
	return false, nil
}

func (nopTestDeps) StartCPUProfile(_ io.Writer) error {
	return nil
}

func (nopTestDeps) StopCPUProfile() {}

func (nopTestDeps) WriteProfileTo(_ string, _ io.Writer, _ int) error {
	return nil
}
func (nopTestDeps) ImportPath() string {
	return ""
}
func (nopTestDeps) StartTestLog(_ io.Writer) {}

func (nopTestDeps) StopTestLog() error {
	return nil
}

// Note: WriteHeapProfile is needed for Go 1.10 but not Go 1.11.
func (nopTestDeps) WriteHeapProfile(_ io.Writer) error {
	// Not needed for Go 1.10.
	return nil
}

func getTestingMain() *testing.M {
	return testing.MainStart(nopTestDeps{}, nil, nil, nil)
}
