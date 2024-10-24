//go:build go1.18
// +build go1.18

package testscript

import (
	"io"
	"reflect"
	"testing"
	"time"
)

type nopTestDeps struct{}

// corpusEntry is an alias to the same type as internal/fuzz.CorpusEntry.
// We use a type alias because we don't want to export this type, and we can't
// import internal/fuzz from testing.
type corpusEntry = struct {
	Parent     string
	Path       string
	Data       []byte
	Values     []interface{}
	Generation int
	IsSeed     bool
}

func (nopTestDeps) SetPanicOnExit0(_ bool) {}

func (nopTestDeps) MatchString(_, _ string) (result bool, err error) {
	return false, nil
}

func (nopTestDeps) StartCPUProfile(_ io.Writer) error {
	return nil
}

func (nopTestDeps) StopCPUProfile() {}

func (nopTestDeps) StartTestLog(_ io.Writer) {}

func (nopTestDeps) StopTestLog() error {
	return nil
}

func (nopTestDeps) WriteProfileTo(_ string, _ io.Writer, _ int) error {
	return nil
}

func (d nopTestDeps) CoordinateFuzzing(_ time.Duration, _ int64, _ time.Duration, _ int64, _ int, _ []corpusEntry, _ []reflect.Type, _ string, _ string) error {
	return nil
}
func (d nopTestDeps) RunFuzzWorker(_ func(corpusEntry) error) error {
	return nil
}

func (d nopTestDeps) ReadCorpus(_ string, _ []reflect.Type) ([]corpusEntry, error) {
	return nil, nil
}

func (d nopTestDeps) CheckCorpus(_ []interface{}, _ []reflect.Type) error {
	return nil
}

func (d nopTestDeps) ResetCoverage() {
	return
}

func (d nopTestDeps) SnapshotCoverage() {
	return
}

func (d nopTestDeps) InitRuntimeCoverage() (mode string, tearDown func(coverprofile string, gocoverdir string) (string, error), snapcov func() float64) {
	tmp := func(_, _ string) (string, error) { return "", nil }
	snapc := func() float64 { return 0 }
	return "", tmp, snapc
}

func (nopTestDeps) ImportPath() string {
	return ""
}

func getTestingMain() *testing.M {
	return testing.MainStart(nopTestDeps{}, nil, nil, nil, nil)
}
