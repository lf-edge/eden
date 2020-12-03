# Eden testing

For testing Eden provides two essential services:

* A place where an abstract library that drives most of test functionality
 gets maintained and developed
* An overall harness for launching all the required components for test runs

You can read more about the general principles of Eden testing at:
[EVE+Integration+Testing](https://wiki.lfedge.org/display/EVE/EVE+Integration+Testing)

## Test building

Tests is a standard Go test [testing](https://golang.org/pkg/testing) compiled by
the `go test -c` command to obtain test binaries. Such “standalone” binaries
and individual tests/subtests from them can be combined in test scripts with
the ability to configure runtime parameters and a specific
EDEN configuration environment. In test scenarios and configuration files
may be used standard Go templates machinery with some EDEN-specific extensions.

Each test is placed in its own directory. This directory may contain
a Makefile (with `build`, `setup`, `clean` and `test` targets),
a test script and the configuration file `eden-config.yml` with
the corresponding fields `eden.test-bin` and `eden.test-script`.
If you have all of these components in the test directory,
you will be able to compile the main `eden` programm and all the tests,
including your test, using `make build`. In this case, the binary test file
and the test script will be placed in the directory specified in
the EDEN `eden.bin-dist` configuration parameter.

## Test running

The tool for running binary tests is `eden test` command:

```console
./eden test -h
Run tests from test binary. Verbose testing works with any level of general verbosity above "info"

test <test_dir> [-s <scenario>] [-t <timewait>] [-v <level>]
test <test_dir> -l <regexp>
test <test_dir> -o
test <test_dir> -r <regexp> [-t <timewait>] [-v <level>]

Usage:
  eden test [test_dir] [flags]

Flags:
  -a, --args string            Arguments for test binary
  -f, --fail_scenario string   scenario for test failing (default "failScenario.txt")
  -h, --help                   help for test
  -l, --list string            list tests matching the regular expression
  -o, --opts                   Options description for test binary which may be used in test scenarious and '-a|--args' option
  -p, --prog string            program binary to run tests
  -r, --run string             run only those tests matching the regular expression
  -s, --scenario string        scenario for tests bunch running
  -t, --timeout string         panic if test exceded the timeout

Global Flags:
      --config string      Name of config (default "default")
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic (default "info")
```

If you have the `eden.test-scene` setting in your EDEN configuration,
you can run it with this command:

```console
./eden test
```

or

```console
./eden test -v debug
```

for more verbose output.

More that -- you can run tests from only one test file as follows:

```console
./eden test tests/integration/ -v debug
```

or with the selected test/subtest combination, possibly with some test-specific parameters:

```console
./eden test tests/lim/ -v debug -a '-timewait 600 -number 3'
```

You can get a list of tests included in the test-binary:

```console
$ ./eden test tests/lim/ -l '.*'
Log/Info/Metric Test
TestLog
TestInfo
TestMetrics
```

and descriptions of test-binary options that can be used for test scripts
and '-a | --args' option parameters:

```console
$ ./eden test tests/integration/ --opts
Usage of /home/user/work/EVE/github/itmo-eve/eden/dist/bin/eden.integration.test:
  -app-docker.yml string
    docker yml file to build
  -app-vm.yml string
    vm yml file to build
  -baseos.eve.download
    EVE downloading flag (default true)
  -baseos.eve.location string
    location of EVE base os directory (default "evebaseos")
  -baseos.eve.tag string
    eve tag for base os
...
```

You will also see standard Go test run-time arguments that you can use
to run a test script, as well as arguments for specific tests, for example,
for stress testing:

```console
...
  -test.count n
    run tests and benchmarks n times (default 1)
...
  -test.parallel n
    run at most n tests in parallel (default 4)
```

## Go template support

EDEN supports [Go Templates](https://golang.org/pkg/text/template) with the ability
to use the Eden framework. You can use the `RenderTemplate` function from
the EDEN SDK and the `eden utils template` command to render any file
(for example, a configuration file or shell script) using tmplates.
Test scenarious also support templates.

Functions related to EDEN that can be used in templates:

* `{{EdenConfig "<config_parameter>"}}` -- get value of Eden config parameter;
* `{{EdenPath "<dir_of_file>"}}` -- resolve path relative to the Eden root directory;
* `{{EdenConfigPath "<config_parameter>"}}` -- combination of EdenConfig and EdenPath;
* `{{EdenGetEnv "<env_var_name>"}}` -- read a value from an environment variable,

can be used for passing external parameters in escripts.

This allows you to use not only hardcoded parameters in test scenarios,
eden-config.yml or any other file which may be handled by `eden utils template`,
but to get them directly from the current configuration. Some examples about using
of templates you can see at [Unit Tests](units).

## Test configuring

Specific arguments for testing binary files may be passed by two ways:

* placing them in the local eden-config.tmpl for this test, for ex.:
[tests/reboot/eden-config.tmpl](../tests/reboot/eden-config.tmpl)
then you must generate eden-config.yaml for this test (it is included into
`build` target of Makefiles of tests for simplicity):
`utils template eden-config.tmpl>eden-config.yml`. You must regenerate config
of tests if they use templates (or run `make build`) after switching context
to properly rendering of templates with data from new context if you choose
first variant (with eden-config.tmpl).
* use of test arguments in test scripts or a selected test from an executable
binary test for ex.:
[tests/vnc/eden.vnc.tests.txt](../tests/vnc/eden.vnc.tests.txt)

The second option is more flexible, because we can run the same test several
times with different parameters in the same scenario.

The most commonly used `eden-config` parameters for setting up a test are:

* `eden.escript.test-scenario` -- name of file with a default test scenario.
This file should be placed in the test's directory or in the `eden.root` directory.
* `eden.escript.test-bin`-- name of default test binary.
Can be used with the `eden test -run` command to run the selected test from
this binary. This file should be placed in the test's directory or in
the `eden.root/eden.bin-dist` directory.

## Test scenarios

Test scenarios are plain text files with one line per command structure.
The most commonly used commands are just test binaries with arguments.
In scenarios, you can use inline comments in the Shell (#) and Go (//) styles.
Comment blocks from Go templates {{/*comment*/}} can also be used.
Example of scenario: [workflow/eden.workflow.tests.txt](workflow/eden.workflow.tests.txt).

Scenarios to run after a fail:

* for eden test: `-f, --fail_scenario string` (default dist/failScenario.txt)
* for eden.escript.test: `-fail_scenario string`

For automatic reset after the test's FAIL in default dist/failScenario.txt you need to set EDEN_FAIL_RESET env. var.:

```console
EDEN_FAIL_RESET=y ./eden test tests/escript/ -p eden.escript.test -r TestEdenScripts/fail_test
```

## Test scripting

Escript test binary `eden.escript.test` provides support for defining
filesystem-based tests by creating scripts in a directory.
To basic implementation of internal Go testscripts added support of
`eden` commands, test-binaries and templates.

Test scripts can be used as glue logic for test binaries “detectors”
and “actors”. All components that are required for tests,
such as configuration files, test data, or external scripts,
can be placed in a test script and processed by the Eden template engine.

You can read more about the test scripting for Eden testing
at [escript/README.md](escript/README.md).
