# Overview

The eventual goal is to be able to
automatically run and compare benchmarks between different versions of EVE.

[Phoronix test suite](https://www.phoronix-test-suite.com/) is a framework for
running benchmarks natively on both Linux and Windows. Primarily it is
interatively driven, however some configuration files can be saved and restored.
In phoronix, test suites define a test via XML, and test results are
stored as XML.

# Framework Selection

[mmtests](https://github.com/gormanm/mmtests) was considered as an alternative
to Phoronix. The advantage of mmtests is that it has a concept of running
benchmarks on a VM and collecting statistics in the host. However, there was
no windows support and windows support became a strict requirement. It may be
possible to integrate components of mmtests and phoronix test suite together.

# Proposed Design

A VM preconfigured with phoronix and cloud-init boots and loads SSH
credentials via cloud-init. A test runner copies the test suite definition(s)
to be run to the VM, runs the test, and copies the test results back, all via
SSH. The current test results are compared with previous ones and if the results
are worse than some threshold, this is flagged.

# Status

## Images

For Windows, a win-phoronix role was created
(see `images/ansible/roles/win-phoronix/README.md`) which should be fit for
purpose as-is. This has a dependency on the
Microsoft Developer Virtual Machines and the license expires for it after 90 days.
It may need to be regenerated every 90 days.

For Linux, phoronix test suite is already released as a
[docker image](https://hub.docker.com/r/phoronix/pts).
It is probably not suitable for running non-interactively out of the box and
needs a `core.pt2so` and `user-config.xml` file added. There may or may not
already be a suitable base image with docker and cloud-init installed.

## Suites

A first pass at suites that cover all major system components and languages
has been performed. The test definitions are in `test-suites`, organized by
whether they run on Linux, Windows, or on both. Preference was given to
suites that run on both platforms if multiple options covering a given criteria
were available.

No network test is included right now because most of them require a third-party
component. An ideal situation would be running an iperf3 server either on the
same system or on the same local area network.

## Scripts

There is a simple proof of concept of copying over, running, and fetching the
results for a windows VM in `windows-runner.sh`.

`parse-test-result.py` is a proof of concept showing that Phoronix test results
can be parsed via XML and python.

`upload-latest-eve-via-zcli.py` downloads the latest tagged EVE image from
docker and uploads it to Zededa's zedcloud via
[zcli](https://hub.docker.com/r/zededa/zcli/).

