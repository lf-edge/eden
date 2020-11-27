# Writing eden tests

## About tests in Eden

Each test suite or individual test must have:

* A makefile that builds the test
* A template eden-config.tmpl. A template contains the nessesary information about binaries, config of the environment where the test runs, test scenario.
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

A template file contains 2 sections: eden and test

Eden section specifies the name of the test and the scenario.
Test section describes the environment required for the test. This data goes to config that the test sends to EVE. It is recommended to use parameters from EdenConfig which is the current config file.

Example:

```code
eden:
    #test binary
    test-bin: "eden.escript.test"

    #test scenario
    test-scenario: "eden.testname.tests.txt"

test:
    controller: adam://{{EdenConfig "adam.ip"}}:{{EdenConfig "adam.port"}}
    eve:
      {{EdenConfig "eve.name"}}:
        onboard-cert: {{EdenConfigPath "eve.cert"}}
        serial: "{{EdenConfig "eve.serial"}}"
        model: {{EdenConfig "eve.devmodel"}}
```

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

#### testdata directory

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

Useful links:
[EVE+Integration+Testing](https://wiki.lfedge.org/display/EVE/EVE+Integration+Testing)
[Eden testing README](https://github.com/itmo-eve/eden/blob/master/tests/README.md)
[Description of existing tests](https://wiki.lfedge.org/display/EVE/Tests)