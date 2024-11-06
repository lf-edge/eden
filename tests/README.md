# Eden Integration Tests

This directory contains a series of integration tests that meet the
[eden task API](../docs/escript/task-writing.md), and thus can be launched using
`eden test`. Each subdirectory contains an individual test suite with one or
more tests.

The general principles for integration testing of EVE are available at
[EVE+Integration+Testing](https://wiki.lfedge.org/display/EVE/EVE+Integration+Testing)

The testing workflow for CI/CD of this eden project is at [workflow/README.MD](workflow/README.MD)

## Running Integration Tests

You can run either the entire suite of tests in this repository, or an
individual test or suite.

### Running The Entire Test Suite

To run the entire suite of integration tests, run either:

* `make test` in the root of this repository
* `make test` in this `tests/` directory

### Running Individual Tests

To run any single suite of integration tests, launch them like any other
[eden test/task](../docs/escript/test-running.md):

```console
eden test tests/testdir/
```

You also can run `make test` in any individual test suite directory.

If you want to run one specific test, run:

```console
eden test tests/testdir/ -t <TestName>
```

See the [documentation for running eden tests](../docs/test-running.md) for
more options.

## Building Integration Tests

The integration tests under `tests/` ship as source code. If you want to
run them, you will need to build them. You can do this in one of several ways:

* To build an individual test, in its directory, run `make build`
* To build all tests, in this directory `tests/`, run `make build`
* To build all tests, in the root directory of the `eden` repository, run one of:
  * `make build-tests` - builds `eden` and all integration tests
  * `make testbin` - builds all integration tests

## Writing Integration Tests

### Test Suite Structure

Each test suite, in its own directory, must implement the
[eden task API](../docs/test-writing.md), and thus contains, at least:

* the configuration file `eden-config.yml` with the corresponding fields
  * `eden.test-bin` - references the name of the binary built by `make build`
	* `eden.test-script` - references the test script in the test directory
* a test script

In addition, each directory, in order to be part of the integration test suite,
contains a Makefile with at least the following targets:

* `build` - build the test binary, if any, and places it in a directed bindir
* `setup` - sets up the test environment
* `clean` - removes any artifacts
* `test` - executes the built test binary

The above targets must exist; if any is unneeded, it should be an empty target.

The bindir is expected to be `$(WORKDIR)/bin/`. For example, if the test binary
is to be named `footest`, then the following should be the behaviour:

* `make build WORKDIR=/tmp` -> `/tmp/bin/footest`
* `make build WORKDIR=/usr/local` -> `/usr/local/bin/footest`
* `make build` -> whatever the `Makefile` uses as its default

These `Makefile` are not requirements for the general
[eden task API](../docs/test-writing.md),but rather to be part of the
standard set of integration tests that are launched by this repository.

### Test Suite Language

The tests in this directory are implemented in Golang. They meet
the [Golang testing standard](https://golang.org/doc/code#Testing) by naming all
test files `_test.go`, and thus can be compiled via `go test -c`. The compiled
standalone binaries and individual tests/subtests from them can be combined in test scripts
with the ability to configure runtime parameters and a specific
EDEN configuration environment. In test scenarios and configuration files
may be used standard Go templates machinery with some EDEN-specific extensions.

However, it is not _required_ that a test be built in go. Future tests in this
directory may use different languages, provided that they meet the
[eden task API](../docs/test-writing.md) and the `Makefile` structure.

## Useful links

* [EVE+Integration+Testing](https://wiki.lfedge.org/display/EVE/EVE+Integration+Testing)
* [Eden testing README](https://github.com/lf-edge/eden/blob/master/tests/README.md)
* [Description of existing tests](https://wiki.lfedge.org/display/EVE/Tests)
