# Running Tests

## Task vs Test

From `eden`'s perspective, a test _is_ a task. A test is simply a task that gets
executed by `eden` with results reported. The terms "task" and "test" may be used
interchangeably.

## Understanding eden Tests

`eden` tests are in a single directory per test suite.

The simplest way to run a test directory is just:

```console
eden test test/dir/
```

This will execute whatever the default is for the test.

Each test directory supports, and `eden` can run, one of two kinds of tests:

* a test binary, with multiple tests built into it
* a test scenario script, which also can call the test binary

A test binary is just an executable binary located in a known path. It need not
actually be in the test directory, and might have nothing to do with the
directory. By convention, the source code for the test binary is in the
test directory. The test binary can have multiple, individual tests inside it.

For example, a test binary named `mytests` might have several tests, such as
`TestMyThings` and `TestOtherThings` defined in it.

To run one or more tests from the test binary:

```console
eden test test/dir/ -r <TestToRun>
```

Using our example above:

```console
eden test test/dir/ -r TestMyThings
```

A test scenario script, on the other hand, is a file in the test directory that
contains a series of test commands to run. It can run many different commands,
some of which might not even be part of the test binary.

To run a test scenario:

```console
eden test test/dir/ -s <scenario>
```

For example, if your scenario file is `eden.my.tests.txt`, you can run it as:

```console
eden test test/dir/ -s eden.my.tests
```

Finally, each test directory has a set of defaults both for the test binary and
for the scenario.

To understand more about how to write tests, read
[writing eden tests](./task-writing.md).

## Test running

The tool for running tasks / tests is `eden test`. Run `eden test -h` to see all
of the options.

To get verbose output, run:

```console
./eden test -v debug
```

If you run `eden test` alone, it will run _all_ of the integration tests in
the [../tests/](../tests/) directory. If you run `eden test test/dir/`, you will
run the tests from a single test directory:

```console
eden test              # run all tests in $EDEN/tests/ dir
eden test test/dir/    # run the tests in test/dir/
```

You can add both eden-specific and test-specific parameters:

```console
./eden test tests/lim/ -v debug -a '-timewait 600 -number 3'
```

You can get a list of tests included in the test-binary:

```console
$ ./eden test tests/lim/ -l '.*'
Log/Info/Metric Test
TestLog
TestAppLog
TestInfo
TestMetrics
TestFlowLog
```

and descriptions of test-binary options that can be used for test scripts
and '-a | --args' option parameters:

```console
$ ./eden test tests/integration/ --opts
Usage of /home/user/work/EVE/github/lf-edge/eden/dist/bin/eden.integration.test:
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

Standard [Go test](https://pkg.go.dev/testing) run-time arguments are available
when running a test script, as well as arguments for specific tests, for example,
for stress testing:

```console
...
  -test.count n
    run tests and benchmarks n times (default 1)
...
  -test.parallel n
    run at most n tests in parallel (default 4)
```
