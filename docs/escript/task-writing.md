# Writing eden Tasks

Any process can be an `eden` task. It need not be a test, nor need it even talk
to a controller or work with an EVE device. As long as the task exists in its
own dedicated directory, which conforms to this eden task API, `eden` happily
will run it using the `eden test` command, and report its results.

    `eden test some/dir/` --> run task that meets task API

In order to simplify working with controllers and EVE devices, this repository
includes utilities, as individual binaries, that can be executed from tests:

    `eden test some/dir/` --> run task that meets task API --> control device via utility

## Running a Test

To run a task/test:

```console
    ./eden test ./path/to/task/directory/
```

For more information on running tests and launch options, including verbosity
levels, running individual test suites, and running individual tests, read
[the task running docs](./test-running.md).

## What `eden test` Runs

Tasks are implemented in a dedicated directory.

When a user runs `eden test your/test/dir/`, it will try to execute one of:

* an executable binary with one or more individual tests in it
* a scenario file in that directory, which may, in turn, execute more scenarios, shell
commands, or the test binary

Your test is expected to provide those artifacts so that
`eden test test/dir/` can run your test. It also is expected to provide a
configuration file `eden-config.yml` indicating the name of the binary and the
name of the scenario file.

This document describes how `eden test` looks for the above, how it executes
them, and what you need to do to provide them.

## Eden Task Structure

A task directory must have:

* A configuration file `eden-config.yml`, which contains the names of the test binary and the test scenario
* A test scenario file named `eden.*testname*.tests.txt`, containing one or more commands to run

Anything else is allowed in the task directory, e.g.:

* Go source files
* Python files
* WASM compiled files
* Test artifacts in `./testdata/`
* Escript files in `./testdata/`
* Dockerfile
* README
* Makefile
* More scenarios
* jpg cat images

The rest of this file describes the structure of each section, and how to build
a new test suite.

### Configuration File

Each task directory must have a configuration file named `eden-config.yml`.

The structure of the file is:

```yml
eden:
    #test binary
    test-bin: "eden.escript.test"

    #test scenario
    test-scenario: "eden.testname.tests.txt"
```

There is one root key - `eden` - and two subsidiary keys:

* `test-bin`: the name of the executable binary to run, normally the same one created via `make build`, and must be in `${WORKDIR}/bin` or `PATH`
* `test-scenario`: the name of the scenario file to run, must be in the task root directory

These provide the defaults for `eden test`, and can be overridden via CLI flags.

You can omit `test-bin` option if your test does not provide a test binary or
uses specific one from another test. If no `test-bin` is provided in the config
file or in the CLI flag, `eden test` will default to using
`${WORKDIR}/bin/eden.escript.test`.

### Test Scenario

The test "scenario" is the main test script in the directory. It must be named
`eden.<testname>.tests.txt` and referenced in `eden-config.yml` as
`eden.test-scenario`.

Scenarios are plain text files, ending in `.txt`, with one line per command.
The most commonly used commands are just test binaries with arguments.
Scenarios support inline comments in the Shell (`#`) and Go (`//`) styles.
Comment blocks from Go templates {{/*comment*/}} can also be used.

When a user runs `eden test test/dir/`, `eden` will do the following:

1. Read the configuration file `eden-config.yml`
1. Locate the configuration file referenced in the configuration as `eden.test-scenario`
1. Pass the contents of the file through a Go-compatible [template engine][Go template support]
1. Execute each line by passing it to a specialized interpreter called `escript`

The `escript` interpreter can be considered a combination bash interpreter
with some additional features. Each line is one of:

* a shell command to execute
* a specialized command that `escript` understands

Using a scenario one can create a set of test runs or sequentially execute
smaller tests.

For detailed information on `escript`, including commands and common patterns,
read [the escript documentation](../tests/escript/README.md).

#### Subscripts

In addition to the main test scenario file, you can have multiple subsidiary
test "scenarios", or "subscripts". These, too, will be read by `eden`, passed
through the [template engine][Go template support], and then executed
line-by-line by `escript`.

Subscripts must:

* be located in the `testdata/` subdirectory of the main test, i.e. `path/to/test/testdata`
* end in `.txt`

When `eden test` runs, in addition to parsing the configuration file and
locating and parsing the scenario file, it looks for subscripts, and creates
tests for them named `TestEdenScripts/<filename>` (without the `.txt` ending).

As these are `escript` scripts, an existing scenario, itself an escript script,
can execute them by calling:

```code
    eden.escript.test -test.run TestEdenScripts/<filename>
```

For example, if your test directory contains a file named `testdata/footest.txt`,
then you can execute it as:

```code
    eden.escript.test -test.run TestEdenScripts/footest
```

#### Sample Scenarios

* Run a single executable, e.g. in [tests/reboot](../tests/reboot/eden.reboot.tests.txt). Note that the executable _must_ be in the `PATH` or `WORKDIR/bin`.
```code
    eden.testname.test
```
* Run two executables:
```code
    eden.testname1.test
    eden.testname2.test
```
* Run scenarios named `test-1.txt`, `test-2.txt` and `test-3.txt`
```code
    eden.escript.test -test.run TestEdenScripts/test-1
    eden.escript.test -test.run TestEdenScripts/test-2
    eden.escript.test -test.run TestEdenScripts/test-3
```

For a sample complex scenario, see
[tests/workflow](../tests/workflow/eden.workflow.tests.txt)):

## Go template support

`eden` will pass any escript file - scenario or subscript - through a
[Go Templates](https://golang.org/pkg/text/template) engine before passing it
on to the [escript interpreter](../tests/escript/).

In addition, you can run the template interpolation manually, to check its
output, via `eden utils template`.

In addition to native [Go Templates](https://golang.org/pkg/text/template)
functions, `eden` provides the following custom functions during template
processing:

* `{{EdenConfig "<config_parameter>"}}` -- get value of Eden config parameter;
* `{{EdenPath "<dir_of_file>"}}` -- resolve path relative to the Eden root directory;
* `{{EdenConfigPath "<config_parameter>"}}` -- combination of EdenConfig and EdenPath;
* `{{EdenGetEnv "<env_var_name>"}}` -- read a value from an environment variable,

This allows you to use not only hardcoded parameters in test scenarios or
`eden-config.yml`, but also to get them directly from the current runtime
configuration. Some template examples are available in
[Unit Tests](../tests/units).

## Test Arguments

If you need to pass arguments to your test binaries, you can do so
inside the scenario file. If you need those arguments to vary, use
[scenario template][Go template support].

Because the scenario file already is templated, you can pass varying arguments
to your test binaries using templates. This enables you to run the same test
several times with different parameters, all in the same scenario.

For example, to run the same test multiple times with different arguments:

```code
eden.testname.test {{$test_count}} first
eden.testname.test {{$test_count}} last
```

Or to run tests conditionally:

```code
{{if (ne $setup "y")}}
# Just restart EVE if not using the SETUP steps
# Is it QEMU?
{{if or (eq $devmodel "ZedVirtual-4G") (eq $devmodel "VBox") (eq $devmodel "parallels") }}
/bin/echo EVE restart (04/{{$tests}})
eden.escript.test -test.run TestEdenScripts/eve_restart
{{end}}
{{end}}
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

The easiest way to run a script from the test's `testdata` directory is to use the '-e' option:

```console
    eden test tests/escript -e message
```

This is the short form of:

```console
    eden test tests/escript -p eden.escript.test -r TestEdenScripts/message
```

List of detectors:

* [eden.reboot.test -- Reboot detector](reboot/README.md)
* [eden.lim.test -- Log/Info/Metric detector](lim/README.md)
* [eden.app.test -- Application state detector](app/README.md)
* [eden.network.test -- Network state detector](network/README.md)
* [eden.vol.test -- Volume state detector](volume/README.md)

You can read more about the test scripting for Eden testing
at [escript/README.md](escript/README.md).

## Example Test Walkthrough

An example test walkthrough is available [here](./test-anatomy-sample.md).
