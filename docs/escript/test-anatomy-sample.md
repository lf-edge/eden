# Anatomy of a Sample Test

In this example, we will analyze an already existing test on Go for checking
logging [lim](../tests/lim/). This test returns success when receiving 1 message
from the log.

## Test Structure

The test directory contains the expected files:

* `Makefile`
* `eden-config.yml` - eden config file
* `eden.lim.tests.txt` - test scenario
* source files for the test binary, in this case, `lim_test.go`
* subscripts in the `testdata/` directory

## Test Binary

The test binary is written in go, as are most of the integration test.

### Imports

The test relies on processing data structures of which `eden` is aware, and so
imports the relevant libraries
[in the go file](../blob/1ef2175ccf68b01f02ec88f37adc662da2896be1/tests/lim/lim_test.go#L13-L23):

```go
import (
...
"github.com/lf-edge/eden/pkg/controller/eflowlog"
"github.com/lf-edge/eden/pkg/controller/einfo"
"github.com/lf-edge/eden/pkg/controller/elog"
"github.com/lf-edge/eden/pkg/controller/emetric"
"github.com/lf-edge/eden/pkg/device"
"github.com/lf-edge/eden/pkg/projects"
"github.com/lf-edge/eden/pkg/tests"
"github.com/lf-edge/eden/pkg/utils"
"github.com/lf-edge/eve-api/go/flowlog"
"github.com/lf-edge/eve-api/go/info"
"github.com/lf-edge/eve-api/go/metrics"
...
)
```

There are 3 distinct categories of imports in our example:

* EVE API
* `eden` convenience
* `eden` test utility

#### EVE API

These imports are straight from the EVE API:

```go
"github.com/lf-edge/eve-api/go/flowlog"
"github.com/lf-edge/eve-api/go/info"
"github.com/lf-edge/eve-api/go/metrics"
```

If we want to use the data structures that come from the EVE device - which we
do - then it is convenient to have those data structures available.

#### `eden` Convenience

These imports provide data structures that are useful in parsing information
passed to us by the `eden` functions:

```go
"github.com/lf-edge/eden/pkg/controller/eflowlog"
"github.com/lf-edge/eden/pkg/controller/einfo"
"github.com/lf-edge/eden/pkg/controller/elog"
"github.com/lf-edge/eden/pkg/controller/emetric"
```

#### `eden` Utility

Finally, and perhaps most importantly, are the `eden` utility library functions.
We don't want every test to have to figure out what EVE edge device is being
used, how to communicate with it, how to reach the controller, etc. The entire
purpose of the test harness is to set all that up and simplify it, so that a
test can focus on doing its test job.

These imports provide those utilities:

```go
"github.com/lf-edge/eden/pkg/device"
"github.com/lf-edge/eden/pkg/projects"
"github.com/lf-edge/eden/pkg/tests"
"github.com/lf-edge/eden/pkg/utils"
```

Using these utility functions, we can test just one or a few aspects of a
device's behaviour, letting the utility library do all of the set up and
interfacing for us.

### Main Function

As the binary is a go test file, with setup and teardown, the main function is
[TestMain](../blob/1ef2175ccf68b01f02ec88f37adc662da2896be1/tests/lim/lim_test.go#L83-L142):

```go
func TestMain(m *testing.M) {
  fmt.Println("Log/Info/Metric Test")

	tests.TestArgsParse()

	tc = projects.NewTestContext()

	projectName := fmt.Sprintf("%s_%s", "TestLogInfoMetric", time.Now())

	// Registering our own project namespace with controller for easy cleanup
	tc.InitProject(projectName)

	// Create representation of EVE instances (based on the names
	// or UUIDs that were passed in) in the context. This is the first place
	// where we're using zcli-like API:
	for _, node := range tc.GetNodeDescriptions() {
		edgeNode := node.GetEdgeNode(tc)
		if edgeNode == nil {
			// Couldn't find existing edgeNode record in the controller.
			// Need to create it from scratch now:
			// this is modeled after: zcli edge-node create <name>
			// --project=<project> --model=<model> [--title=<title>]
			// ([--edge-node-certificate=<certificate>] |
			// [--onboarding-certificate=<certificate>] |
			// [(--onboarding-key=<key> --serial=<serial-number>)])
			// [--network=<network>...]
			//
			// XXX: not sure if struct (giving us optional fields) would be better
			edgeNode = tc.NewEdgeNode(tc.WithNodeDescription(node), tc.WithCurrentProject())
		} else {
			// make sure to move EdgeNode to the project we created, again
			// this is modeled after zcli edge-node update <name> [--title=<title>]
			// [--lisp-mode=experimental|default] [--project=<project>]
			// [--clear-onboarding-certs] [--config=<key:value>...] [--network=<network>...]
			edgeNode.SetProject(projectName)
		}

		tc.ConfigSync(edgeNode)

		// finally we need to make sure that the edgeNode is in a state that we need
		// it to be, before the test can run -- this could be multiple checks on its
		// status, but for example:
		if edgeNode.GetState() == device.NotOnboarded {
			log.Fatal("Node is not onboarded now")
		}

		// this is a good node -- lets add it to the test context
		tc.AddNode(edgeNode)
	}

	tc.StartTrackingState(false)

	// we now have a situation where TestContext has enough EVE nodes known
	// for the rest of the tests to run. So run them:
	res := m.Run()

	// Finally, we need to cleanup whatever objects may be in in the
	// project we created and then we can exit
	os.Exit(res)
}
```

The majority of the function is setup. At the end of it, we call `m.Run()` to
run the tests, and then exit with `os.Exit()`.

Let's take a deeper look at the setup. We ignore extraneous lines like printing
out logs.

First, we call [tests.TestArgsParse()](https://pkg.go.dev/github.com/lf-edge/eden/pkg/tests#TestArgsParse),
which parses all of the various args as setup. It is the `eden` equivalent
of [flag.Parse](https://pkg.go.dev/flag#Parse).

```go
tests.TestArgsParse()
```

Next, we get a [TestContext](https://pkg.go.dev/github.com/lf-edge/eden/pkg/projects#TestContext):

```go
tc = projects.NewTestContext()
```

The `TestContext` is the context within which the test will run. It will give
us access to the controller, edge nodes, and everything else that goes
with the test.

With a `TestContext` in hand, we register a unique project. As the comment
indicates, this makes cleanup easier by associating any work we do during the
test with the specific project.

```go
// Registering our own project namespace with controller for easy cleanup
tc.InitProject(projectName)
```

Finally, with the `TestContext` set up, we get the edge nodes:

```go
for _, node := range tc.GetNodeDescriptions() {
  edgeNode := node.GetEdgeNode(tc)
  if edgeNode == nil {
    edgeNode = tc.NewEdgeNode(tc.WithNodeDescription(node), tc.WithCurrentProject())
  } else {
    edgeNode.SetProject(projectName)
  }

  tc.ConfigSync(edgeNode)

  if edgeNode.GetState() == device.NotOnboarded {
    log.Fatal("Node is not onboarded now")
  }

  // this is a good node -- lets add it to the test context
  tc.AddNode(edgeNode)
}
```

The above goes through every edge node that the controller has listed, either
creating it or registering it. It then syncs the config to the node:

```go
tc.ConfigSync(edgeNode)
```

then checks that it was onboarded. Since we can do nothing with a device that
isn't onboarded, an error in finding the device onboarded is fatal.

Finally, we add the good and onboarded edge node to the `TestContext`:

```go
tc.AddNode(edgeNode)
```

At this point, the `TestContext` is fully set up, and has at least one edge node
that is onboarded with a fully-synced config. It is ready and waiting to execute
whichever commands our tests want to give it.

You can read more about `TestMain` in the
[go testing reference documentation](https://pkg.go.dev/testing#hdr-Main), and
usage for EVE
[on the EVE wiki](https://wiki.lfedge.org/display/EVE/EVE+Integration+Testing).

### Test Functions

In order to actually execute tests, you need to create test functions. These
all have the signature `func Test*` (other than `TestMain`, of course).
The lim testing file has four such functions:

* [TestLog](../blob/1ef2175ccf68b01f02ec88f37adc662da2896be1/tests/lim/lim_test.go#L144)
* [TestInfo](../blob/1ef2175ccf68b01f02ec88f37adc662da2896be1/tests/lim/lim_test.go#L185)
* [TestMetrics](../blob/1ef2175ccf68b01f02ec88f37adc662da2896be1/tests/lim/lim_test.go#L226)
* [TestFlowLog](../blob/1ef2175ccf68b01f02ec88f37adc662da2896be1/tests/lim/lim_test.go#L269)

#### Single Function

We will explore a single test function, [`TestLog`](../blob/1ef2175ccf68b01f02ec88f37adc662da2896be1/tests/lim/lim_test.go#L144),
which tests a log.

The function has 3 main parts:

1. In the first part, we check for the correctness of the query and initialize the edgeNode variable.
2. In the second part, we call [tc.AddProcLog](https://pkg.go.dev/github.com/lf-edge/eden/pkg/projects#TestContext.AddProcLog), to add a handler for processing any logs received from the device. Details are below.
3. In the third part, by calling `tc.WaitForProc (* timewait)` we block until the time expires or all processes have finished.

```go
func TestLog(t *testing.T) {
 err := mkquery()
 if err != nil {
  t.Fatal(err)
 }

 edgeNode := tc.GetEdgeNode(tc.WithTest(t))

 t.Logf("Wait for log of %s number=%d timewait=%d\n", edgeNode.GetName(), *number, *timewait)

 tc.AddProcLog(edgeNode, func(log *elog.LogItem) error {
  return func(t *testing.T, edgeNode *device.Ctx, log *elog.LogItem) error {
   name := edgeNode.GetName()
   if query != nil {
    if elog.LogItemFind(*log, query) {
     found = true
    } else {
     return nil
    }
   }
   t.Logf("LOG %d(%d) from %s:\n", items+1, *number, name)
   if len(*out) == 0 {
    elog.LogPrn(log, elog.LogLines)
   } else {
    elog.LogItemPrint(log, elog.LogLines,
     strings.Split(*out, ":")).Print()
   }

   cnt := count("Received %d logs from %s", name)
   if cnt != "" {
    return fmt.Errorf(cnt)
   }
   return nil
  }(t, edgeNode, log)
 })

 tc.WaitForProc(*timewait)
}
```

We used the [AddProcLog](https://pkg.go.dev/github.com/lf-edge/eden/pkg/projects#TestContext.AddProcLog)
function to add a handler for all logs received from the device. This enables us
to process logs we want until we are done. `AddProcLog` takes 2 arguments:

* The [edgeNode](https://pkg.go.dev/github.com/lf-edge/eden@v0.1.5-alpha/pkg/device#Ctx) whose logs we want to process
* `func (log *elog.LogItem) error` function which will be called for each log for the edgeNode, processing the log for our test. We return nil to indicate we have more logs to process, or a non-nil `error` to indicate we are done processing logs.

> You can also see an example with pseudocode of the [TestReboot function here](https://wiki.lfedge.org/display/EVE/EVE+Integration+Testing)

## Scenario

The main scenario file is
[eden.lim.tests.txt](../tests/lim/eden.lim.tests.txt). The content is simple:

```code
eden.escript.test -test.run TestEdenScripts/log_test
eden.escript.test -test.run TestEdenScripts/info_test
eden.escript.test -test.run TestEdenScripts/metric_test
```

Each line will be processed by [escript](../tests/escript/) in turn.

We will analyze the first line:

```code
eden.escript.test -test.run TestEdenScripts/log_test
```

This line says to execute `eden.escript.test` to run the subscript `log_test`,
which is located in [testdata/log_text.txt](../tests/lim/testdata/log_test.txt),
as indicated by `TestEdenScripts`, which as discussed above, references the
generated tests for the escript files in `testdata/`.

The `log_test` scenario file, in turn, contains the following content.

```code
{{$test1 := "test eden.lim.test -test.v -timewait 10m -test.run TestLog"}}

# ssh into EVE to force log creation
exec -t 5m bash ssh.sh &

# Trying to find messages about ssh in log
{{$test1}} -out content 'content:.*Disconnected.*'
stdout 'Disconnected from'

# Test's config. file
-- eden-config.yml --
test:
    controller: adam://{{EdenConfig "adam.ip"}}:{{EdenConfig "adam.port"}}
    eve:
      {{EdenConfig "eve.name"}}:
        onboard-cert: {{EdenConfigPath "eve.cert"}}
        serial: "{{EdenConfig "eve.serial"}}"
        model: {{EdenConfig "eve.devmodel"}}

-- ssh.sh --
EDEN={{EdenConfig "eden.root"}}/{{EdenConfig "eden.bin-dist"}}/{{EdenConfig "eden.eden-bin"}}
until $EDEN eve ssh sleep 10; do sleep 10; done
```

We will analyze it line-by-line.

The first line is a Go template that just sets a variable:

```code
{{$test1 := "test eden.lim.test -test.v -timewait 10m -test.run TestLog"}}
```

The next line executes a bash shell, sets a 5-minute timeout, and then executes
`ssh.sh` to ssh into the EVE device:

```code
# ssh into EVE to force log creation
exec -t 5m bash ssh.sh &
```

Then we launch the test:

```code
{{$test1}} -out content 'content:.*Disconnected.*'
```

If we expand the variable, the line is:

```code
test eden.lim.test -test.v -timewait 10m -test.run TestLog -out content 'content:.*Disconnected.*'
```

This line means (ignoring some of the flags):

* execute as a test
* the file `eden.lim.test`, which is the compiled binary
* specifically running the test named `TestLog`

Next we send some text to stdout:

```code
stdout 'Disconnected from'
```

We also create an override config file for the test:

```code
# Test's config. file
-- eden-config.yml --
test:
    controller: adam://{{EdenConfig "adam.ip"}}:{{EdenConfig "adam.port"}}
    eve:
      {{EdenConfig "eve.name"}}:
        onboard-cert: {{EdenConfigPath "eve.cert"}}
        serial: "{{EdenConfig "eve.serial"}}"
        model: {{EdenConfig "eve.devmodel"}}
```

The line `-- eden-config.yml --` indicates that all of the following lines will
be the `eden-config.yml` to use for the test, until a blank line is reached.

Finally, we create the `ssh.sh` script that we referenced earlier:

```code
-- ssh.sh --
EDEN={{EdenConfig "eden.root"}}/{{EdenConfig "eden.bin-dist"}}/{{EdenConfig "eden.eden-bin"}}
until $EDEN eve ssh sleep 10; do sleep 10; done
```

Note that the shell script itself uses Go template to interpolate:

* the root to eden's repository
* the path to the binary directory, where `make build` should have installed the test binary
* the name of the test binary, as configured in lim's root `eden-config.yml`, in the value `eden.test-bin`

Put together, this means that it does the following:

1. set the environment variable `EDEN` to the full path to the executable compiled by this directory, from the `lim_test.go` source file
1. a simple bash `until` loop repeats until the command `eden.lim.test eve ssh sleep 10` successfully returns
