# Overview

The eventual goal is to be able to run benchmarks as applications on EVE and
compare the results of those benchmarks, so that the performance impact to
applications resulting from changes to EVE can be measured.

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
Microsoft Developer Virtual Machines.

For Linux, phoronix test suite is already released as a
[docker image](https://hub.docker.com/r/phoronix/pts).
It is probably not suitable for running non-interactively out of the box and
needs a `core.pt2so` and `user-config.xml` file added. There may or may not
already be a suitable base image with docker and cloud-init installed.

Regenerating `core.pts2so` requires running `phoronix-test-suite batch-setup`
with:


    Save test results when in batch mode (Y/n): y
    Open the web browser automatically when in batch mode (y/N): n
    Auto upload the results to OpenBenchmarking.org (Y/n): n
    Prompt for test identifier (Y/n): n
    Prompt for test description (Y/n): n
    Prompt for saved results file-name (Y/n): n
    Run all test options (Y/n): n

## Suites

A first pass at suites that cover all major system components and languages
has been performed. The test definitions are in `test-suites`, organized by
whether they run on Linux, Windows, or on both. Preference was given to
suites that run on both platforms if multiple options covering a given criteria
were available.

Overall, there are two categories of tests included: some that measure core
peripherals and components, and others that benchmark applications similar to
what users may be interested in, such as specific languages like go or python.

Core benchmarks, ones run for every version of EVE, should likely include
openssl, fio, a network test, and osbench.

osbench and openssl are primarily to measure performance impacts from new
Linux/KVM and qemu versions.

fio measures disk performance and may be impacted by changes to the underlying
storage layer.

Network performance could be impacted by changes to the host configuration and
the network setup.

Any network test should use either a server on the same virtualization instance
or on the local network, to remove noise from general network traffic. Either
iperf(3) or ethr should work.

Adding to the supported benchmarks may require manually adding dependencies to
the Windows image to be able to run non-interactively.

## Scripts

There is a simple proof of concept of copying over, running, and fetching the
results for a windows VM in `windows-runner.sh`.

`parse-test-result.py` is a proof of concept showing that Phoronix test results
can be parsed via XML and python.

`upload-latest-eve-via-zcli.py` downloads the latest tagged EVE image from
docker and uploads it to Zededa's zedcloud via
[zcli](https://hub.docker.com/r/zededa/zcli/).

