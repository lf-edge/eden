# Eden testing

For testing Eden provides two essential services:

* A place where an abstract library that drives most of test functionality gets maintained and developed
* An overall harness for launching all the required components for test runs

You can read more about the general principles of Eden testing at:
https://wiki.lfedge.org/display/EVE/EVE+Integration+Testing

## Test building
Tests is a standard Go test (https://golang.org/pkg/testing) compiled by the `go test -c` command to obtain test binaries. Such “standalone” binaries and individual tests/subtests from them can be combined in test scripts with the ability to configure runtime parameters and a specific EDEN configuration environment. In test scenarios and configuration files may be used standard Go templates machinery with some EDEN-specific extensions.

Each test is placed in its own directory. This directory may contain a Makefile (with `build` and `test` targets), a test script and the configuration file `eden-config.yml` with the corresponding fields `eden.test-bin` and `eden.test-script`. If you have all of these components in the test directory and you have added the build and test commands to [tests/Makefile](tests/Makefile), you will be able to compile the main `eden` programm and all the tests, including your test, using `make build`. In this case, the binary test file and the test script will be placed in the directory specified in the EDEN `eden.bin-dist` configuration parameter.

## Test running
The tool for running binary tests is `eden test` command:
```
./eden test -h
Run tests from test binary. Verbose testing works with any level of general verbosity above "info"

test <test_dir> [-s <scenario>] [-t <timewait>] [-v <level>]
test <test_dir> -l <regexp>
test <test_dir> -o
test <test_dir> -r <regexp> [-t <timewait>] [-v <level>]

Usage:
  eden test [test_dir] [flags]

Flags:
  -a, --args string       Arguments for test binary
  -h, --help              help for test
  -l, --list string       list tests matching the regular expression
  -o, --opts              Options description for test binary which may be used in test scenarious and '-a|--args' option
  -p, --prog string       program binary to run tests
  -r, --run string        run only those tests matching the regular expression
  -s, --scenario string   scenario for tests bunch running
  -t, --timeout string    panic if test exceded the timeout

Global Flags:
      --config-file string   path to config file (default "~/.eden/contexts/default.yml")
  -v, --verbosity string     Log level (debug, info, warn, error, fatal, panic (default "info")
```
If you have the `eden.test-scene` setting in your EDEN configuration, you can run it with this command:
```
./eden test
```
or
```
./eden test -v debug
```
for more verbose output.

More that -- you can run tests from only one test file as follows:
```
./eden test tests/integration/ -v debug
```
or with the selected test/subtest combination, possibly with some test-specific parameters:
```
./eden test tests/lim/ -v debug -a '-timewait 600 -number 3'
```
You can get a list of tests included in the test-binary:
```
$ ./eden test tests/lim/ -l '.*'
Log/Info/Metric Test
TestLog
TestInfo
TestMetrics
```
and descriptions of test-binary options that can be used for test scripts and '-a | --args' option parameters:
```
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
You will also see standard Go test run-time arguments that you can use to run a test script, as well as arguments for specific tests, for example, for stress testing:
```
...
  -test.count n
    	run tests and benchmarks n times (default 1)
...
  -test.parallel n
    	run at most n tests in parallel (default 4)
```

## Go template support

EDEN supports [Go Templates](https://golang.org/pkg/text/template) with the ability to use the Eden framework. You can use the `RenderTemplate` function from the EDEN SDK and the `eden utils template` command to render any file (for example, a configuration file or shell script) using tmplates. Test scenarious also support templates.

Functions related to EDEN that can be used in templates:
* `{{EdenConfig "<config_parameter>"}}` -- get value of Eden config parameter;
* `{{EdenPath "<dir_of_file>"}}` -- resolve path relative to the Eden root directory;
* `{{EdenConfigPath "<config_parameter>"}}` -- combination of EdenConfig and EdenPath.

This allows you to use not only hardcoded parameters in test scenarios, eden-config.yml or any other file which may be handled by `eden utils template`, but to get them directly from the current configuration. Some examples about using of templates you can see at [Unit Tests](tests/units/).

## Test configuring

Specific arguments for testing binary files may be passed by two ways:

* placing them in the local eden-config.yaml for this test, for ex.:
https://github.com/lf-edge/eden/blob/master/tests/reboot/eden-config.tmpl
* use of test arguments in test scripts or a selected test from an executable binary test for ex.:
https://github.com/lf-edge/eden/blob/master/tests/integration/eden.integration.tests.txt

The second option is more flexible, because we can run the same test several times with different parameters in the same scenario.
