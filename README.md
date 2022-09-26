# Eden

Eden is the simplest way to setup & test [EVE](https://github.com/lf-edge/eve)
and [Adam](https://github.com/lf-edge/adam).

Eden is a management harness that provides two layers of management.

* infrastructure: deploy and/or delete nodes running [EVE](https://github.com/lf-edge/eve),
  controller [Adam](https://github.com/lf-edge/adam) and [software-defined networks](./sdn/README.md)
  between EVE and the controller
* tasks: execute on EVE, via the controller, one or more tasks

Eden is particularly suited to running tests and test suites. These tests must
meet eden's test API. This repository also includes a framework for simplify
running the tests on the managed EVE via Adam, and reporting on
results.

Eden is inspired by Kubernetes workflows and CLI

Note that EVE by itself without a controller is useless in practice. It retrieves its entire
configuration from the controller, and has no console commands that can be used standalone,
like general-purpose Linux distributions. You use the controller to tell EVE which workloads
you want to run. EVE, in turn, runs those workloads in containers or VMs.

EVE supports the following workload formats:

* OCI image from any OCI compliant registry
* OS image in qcow2 format
* OS image in raw format

Eden is controlled by a single (secretly-named) command `eden`.
It has multiple sub-commands and options. Run `eden help` to see sub-commands.

## Running Eden

You need at least two devices, either or both of which can be virtual:

* Edge: this is the device on which you will run EVE, and launch tasks such as tests.
* Manager: this is where you will run Eden and, optionally, all of the management components.

A typical eden workflow is:

1. install the [prerequisites](#prerequisites)
1. create a named context to store all of your configuration - `eden config add <name>`
1. (optional) set options for the context - `eden config set <name> [options...]`
1. run setup - `eden setup`, which extracts an eve-os qcow2 disk image from the docker image named in the context
1. start Eden's components - `eden start`
   * if running EVE as qemu, entirely under eden control, it will start automatically
   * if running EVE as a separate EVE device:
     1. create or download the EVE image you want
     1. flash the image to the device's storage
     1. start the device
1. onboard EVE - `eden eve onboard`, explicitly allowing it to connect to the controller
1. use the Eden CLI to perform tasks, such as install apps or run tests
1. terminate Eden's components and, optionally, EVE - `eden stop`
1. clean up - `eden clean`

To customize the device on which EVE is running, or the image launched, see
[docs/eve-images.md](./docs/eve-images.md).

### Prerequisites

On the manager, you need:

* the `eden` binary
* [docker](https://docker.com), to run eden's components, including the ability to execute commands. For Linux:

```console
sudo usermod -aG docker $USER
newgrp docker
```

* test binaries - eden ships with pre-compiled ones
* a text editor to configure the system and create test scenarios
and scripts using `eden`

Eden itself -- the main executable file `eden`,
components, and tests -- ships as stand-alone applications
or docker images. You do not need to install and configure the development
environment on your computer.

The manager hosts:

* Eden - control layer CLI
* Adam - the controller, running as a daemon process
* SDN - software-defined networking between EVE (in Qemu) and the controller (to emulate various connectivity scenarios)
* Redis - the log database, running as a daemon process
* Eserver - the image/file database to expose http and ftp content for downloading to the edge device
* Registry - an OCI-compliant registry to expose images for downloading to the edge device

Eden currently does not support interfacing with a commercial controller,
although it is planned for the future.

![Components](/eden_eve_components.png)

As the manager hosts the controller, which the edge device must connect to,
the edge device _must_ be able to connect via network to port 3333.

![Architecture](/Eden_eve_architecture.png)

For the edge device, you need an [EVE](https://github.com/lf-edge/eve) OS image
to install.

**Important Note:** EVE's strong security model allows it to only connect to one
controller, during onboarding. Once onboarded EVE will **not** switch its
allegiance to any other controller. You simply cannot change the registered
certificates or controller IP for an onboarded device. This requires that you be
specify the right IP of the target controller when creating the EVE image.

There are additional requirements for certain use cases:

#### Local Virtual Device

If you intend to run EVE in a virtual local device, you also will need:

* qemu version 4.x or higher
* Linux: [KVM](https://www.linux-kvm.org/page/Main_Page), including the ability to execute commands. For Linux:

```console
  sudo usermod -aG kvm $USER
  newgrp kvm
```

* macOS: [machyve](https://github.com/machyve/xhyve) or [Parallels](./docs/parallels.md)
* `telnet`
* squashfs tools, available, depending on your OS, as `squashfs` or `squashfs-tools`

EVE uses virtualization; to run in VM-based environments, see [the cloud document](./docs/virtual-eve.md).

#### Raspberry Pi

If you want to use Eden with Raspberry Pi on Linux, you also need to install
`binfmt-support` and `qemu-user-static`.

### Quickstart Guides

All of the quickstart guides get you running quickly, and assume you already have
eden installed.
They start the official [nginx](https://hub.docker.com/_/nginx) image as a container,
serving the content of [./data/helloeve/](./data/helloeve ) on port 8028. You will be able to access it via `http://<EVE IP>:8028`.
Note that if you are running EVE as virtual device in Qemu with SDN enabled, you will not be able to
access EVE IP and your apps directly, instead `eden sdn fwd` command must be used (run with no arguments to print help).
Feel free to change the content in [./data/helloeve/](./data/helloeve) and redeploy the pod.

At any time, to get the status of what is running and where, including `<EVE IP>`, run:

```console
eden status
```

#### Quickstart Local (PC in qemu)

```console
eden config add default
eden setup
eden start
eden eve onboard
eden pod deploy docker://nginx -p 8028:80 --mount=src=./data/helloeve,dst=/usr/share/nginx/html
eden status

# if SDN is disabled (sdn.disable=true)
curl http://<EVE IP>:8028
# if SDN is enabled (sdn.disable=false)
eden sdn fwd eth0 8028 curl http://FWD_IP:FWD_PORT
```

When done be sure to clean up:

```console
make clean
# OR
eden stop && eden clean --current-context=false
```

#### Quickstart Hardware (Build an image for real x86)

```console
eden config add default --devmodel general
eden config set default --key adam.eve-ip --value <IP of Manager>
eden setup
```

Burn the image that was displayed to a the proper storage for you device, e.g.
SD card or USB drive. There are many utilities and tools to do so,
from the venerable `dd` to [balena etcher](https://www.balena.io/etcher/).

Boot your device from the storage medium.

Then on eden:

```console
eden start
eden eve onboard
eden pod deploy docker://nginx -p 8028:80 --mount=src=./data/helloeve,dst=/usr/share/nginx/html
eden status
curl http://<EVE IP>:8028
```

When done be sure to clean up:

```console
make clean
# OR
eden stop && eden clean --current-context=false
```

### Target Platforms

EVE can run on most platforms. However, there are some considerations when
running on certain platforms. See [docs/eve-platforms.md](./docs/eve-platforms.md).

## Eden shell settings

For more ease of use of Eden, you can use the automatically generated setup files for your shell:

* for BASH/ZSH -- `source ~/.eden/activate.sh`
* for TCSH -- `source ~/.eden/activate.csh`

These settings add the Eden's binaries directory to the PATH environment variable and add the "EDEN\_<current\_config>\_" label to the command prompt.

In setup files defined some functions (BASH/ZSH) and aliases (TCSH) for work with configs:

* eden+config <config\_name> -- add new config for Eden
* eden-config <config\_name> -- remove config from Eden and switch to 'default'
* eden_config <config\_name> -- switch Eden to config and change prompt

To deactivate this settings call `eden_deactivate` function.

You may configure Bash/Zsh/Fish shell completions for Eden by command `eden utils completion`.

## Eden Configurations

Eden's config is controlled via a yaml file, overriddable using command-line options.
In most cases, the defaults will work just fine for you. If you want to change
the configuration of any of the services, or use multiple stored setups, see
[docs/config.md](./docs/config.md).

## Remote access to eve

To get a shell on the EVE device, once the device is fully registered to its
controller:

```console
eden eve ssh
```

You can run a single command remotely by passing it as argument:

```console
eden eve ssh "ls -la"
```

If you need access to the actual device console, run:

```console
eden eve console
```

## Applications on EVE

Applications are controlled on an EVE device with the `eden pod` commands.
For details, see [applications](./docs/applications.md).

## Tests

Running tests is simple:

```console
eden test <test folder>
```

For example -- to run the reboot test:

```console
eden test tests/reboot/
```

Or to run the workflow tests:

```console
EDEN_TEST=small eden test tests/workflow -v debug
```

For tests that accept parameters, simply pass them after the test path. For
example, to run Log/Metrics/Info test
in debug mode with timeout of 600 seconds and requiring 3 messages of each type:

```console
eden test tests/lim/ -v debug -a '-timewait 600 -number 3'
```

As of this writing, you must _build_ the tests before running them:

```console
make build-tests
```

For more information about running tests, as well as creating your own,
start [here](tests/README.md).

## Help

You can get more information about `make` actions by running `make help`.

More information can be found at the
[wiki](https://wiki.lfedge.org/display/EVE/EDEN) or in the #eve-help channel in
[slack](https://slack.lfedge.org/).
