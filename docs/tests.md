# Writing eden tests

## Eden Test Structure

Each test suite or individual test must have:

* A makefile that builds the test
* A template eden-config.tmpl. A template contains the necessary information about binaries, config of the environment where the test runs, test scenario.
* A test scenario file named eden. **testname**.tests.txt A scenario can contain one or more tests. A scenario allows to run bash scripts, escripts and go binaries.

The test can also contain:

* Go files
* Escript files in ./testdata catalogue
* Docker images
* README file

### Makefile

Should have the following targets:

* clean
* test
* build
* setup
* image (When we have docker image)
* help

All tests are built whenever `make build` is called in the Eden directory.
The cleanup happens whenever `make clean` or `eden clean` is called in the Eden directory.

### A template file - eden-config.tmpl

File specifies the name of the test binary and the scenario. Those parameters overrides global parameters from config file.

Example:

```code
eden:
    #test binary
    test-bin: "eden.escript.test"

    #test scenario
    test-scenario: "eden.testname.tests.txt"
```

You can omit `test-bin` option if your test does not provide test binary or use specific one from another test.
In this case it will use default one `eden.escript.test`.

### A test scenario

The main test script is described in the file: `eden.testname.tests.txt`

The scenario is the file that is presented in the test root directory and that will be executed line-by-line by the escript interpreter. Escript interpreter can be considered as an extended version of bash interpreter + some additional features. [About test scenarios with escripts here.](https://github.com/itmo-eve/eden/blob/master/tests/escript/README.md)

A scenario runs one or more binaries or escripts. Using scenario one can create a set of test runs or sequentially execute smaller tests.

Scripts also have support for GO templates. [About Go template support](https://github.com/itmo-eve/eden/blob/master/tests/README.md#test-running)

An examples of a test scenario is:

When the script is only expected to run the executable (There is a similar implementation in the [/tests/reboot](https://github.com/itmo-eve/eden/blob/master/tests/reboot/eden.reboot.tests.txt)):

```code
eden.testname.test
```

And when the main script runs the scripts from the `testdata` directory one by one (There is a similar implementation in the [/tests/workflow](https://github.com/itmo-eve/eden/blob/master/tests/workflow/eden.workflow.tests.txt)):

```code
eden.escript.test -test.run TestEdenScripts/test-1
eden.escript.test -test.run TestEdenScripts/test-2
eden.escript.test -test.run TestEdenScripts/test-3
```

#### Testdata directory

Contains escript tests (consider as individual tests) They can be also called from another scenario. Note that the environment(template) will be used for calling the test.

## Running the test

Before running a test, you need to collect all tests from the main Eden directory with the command:

```console
make build-tests
```

In this case, the binary test file and the test script will be placed in the directory specified in the EDEN `eden.bin-dist` configuration parameter.

To run the test use command:

```console
./eden test ./tests/<testfolder>
```

More information on running tests and launch options can be found [here](https://github.com/itmo-eve/eden/blob/master/tests/README.md#test-running).

## Useful links

* [EVE+Integration+Testing](https://wiki.lfedge.org/display/EVE/EVE+Integration+Testing)
* [Eden testing README](https://github.com/itmo-eve/eden/blob/master/tests/README.md)
* [Description of existing tests](https://wiki.lfedge.org/display/EVE/Tests)

## Example

In this example, we will analyze an already existing test on Go for checking logging [TestLog](https://github.com/lf-edge/eden/blob/6e040a6eb8e010f06f646158404854d945368e9f/tests/lim/lim_test.go#L135). This test returns success when receiving 1 message in the log.

### About the test

Before considering the test itself, it should be noted that for this test, at the beginning of the file, the necessary package was imported:

```code
import (
...
 "github.com/lf-edge/eden/pkg/controller/elog"
...
)
```

In this test, we have the function `func TestMain(m *testing.M)` its purpose and process are well described in the [lim_test.go file](https://github.com/lf-edge/eden/blob/6e040a6eb8e010f06f646158404854d945368e9f/tests/lim/lim_test.go#L72). You can also read about this function [here](https://wiki.lfedge.org/display/EVE/EVE+Integration+Testing).

The function of the test itself: `func TestLog (t * testing.T)` is quite simple and we can divide it into 3 main parts:

1. In the first part, we check for the correctness of the query and initialize the edgeNode variable.
2. In the second part, we call the `tc.AddProcLog` function which takes 2 arguments, the first argument takes `edgeNode`, and the second argument takes the `func (log * elog.LogItem)` function (that will get all logs for edgeNode), which we immediately describe it, and further in the body of the function (right after return) a self-calling function is described, which performs our check.
3. In the third part, by calling `tc.WaitForProc (* timewait)` we block until the time expires or all processes have finished.

```code
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

> You can also see an example with pseudocode of the [TestReboot function here](https://wiki.lfedge.org/display/EVE/EVE+Integration+Testing)

### About running a test

In the main instruction file `eden.lim.tests.txt` we have the following call:

```code
eden.escript.test -test.run TestEdenScripts/log_test
...
```

This command runs the instruction described in the file [lim/testdata/log_test.txt](https://github.com/lf-edge/eden/blob/master/tests/lim/testdata/log_test.txt)

The instruction performs only 2 steps:

```code
# Trying to find eth0 or eth1 in msg.
test eden.lim.test -test.v -timewait 600 -test.run TestLog -out msg 'msg:.*eth[01].*'
stdout 'eth[01]'

# Checking msg for interfaces other than eth0 or eth1.
! test eden.lim.test -test.v -test.run TestLog -out msg msg:'.*dev eth[^01].*'
```

This file also contains Test's config `eden-config.yml`
