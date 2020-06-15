package projects

import (
	"context"
	"testing"
	"time"
)

type AssertArgs interface{}

type AssertFunc func(t *testing.T, tc *TestContext, description string, args AssertArgs)

type Assert struct {
	Descr string
	Func  AssertFunc
	Args  AssertArgs
}

var (
	asserts = []Assert{}
)

func timeWait(t *testing.T, tc *TestContext, description string, args AssertArgs) {
	timeout := args.(time.Duration)
	t.Logf("timeWait %s started\n", timeout)
	time.Sleep(timeout)
	t.Fatalf("timeWait %s finished\n", timeout)
}

//AssertInfo method to return immediately and simply register a function with
//arguments.
func (tc *TestContext) AssertAdd(t *testing.T, descr string, assert AssertFunc, args AssertArgs) {
	a := Assert{Descr: descr, Func: assert, Args: args}
	asserts = append(asserts, a)
}

//AssertInfo method to return immediately and simply register a listener
//function that would check every incoming Info message and either exit
//with success on one of them OR exit with failure.
func (tc *TestContext) AssertInfo(t *testing.T, descr string, assert AssertFunc) {
	tc.AssertAdd(t, descr, assert, nil)
}

//WaitForAsserts blocking execution until the time elapses or asserts fires
func (tc *TestContext) WaitForAsserts(t *testing.T, secs int) {
	timeout := time.Duration(secs) * time.Second
	t.Log("Timewait: ", timeout)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // cancel when we are finished tasks

	tc.AssertAdd(t, "Test's timewait", timeWait, AssertArgs(timeout))

	for _, a := range asserts {
		go func(descr string, afn AssertFunc, args AssertArgs) {
			t.Logf("WaitForAsserts starting '%s'\n", descr)
			afn(t, tc, descr, args)
			cancel()
			return
		}(a.Descr, a.Func, a.Args)
	}

	select {
	case <-time.After(timeout):
		t.Fatal("assertFunc terminated by timewat", timeout)
	case <-ctx.Done():
		t.Log("assertFunc finished")
	}
}
