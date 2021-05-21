# Flow

What is the flow of launching an entire setup?

How can you do something basic, like launching an edge container or updating
the base OS image?

This document explains how.

## Launching EVE

The following are the steps to launch eve running the default version of
adam with default options. See below for options on modifying the flow.

1. Make sure you have prerequisites, like `docker` and `qemu`. See
the main [README](../README.md).
1. Make sure you have `eden` the binary, or, if you prefer, this entire repository.
See the main [README](../README.md).
1. Create a basic config named `default` with: `eden config add default`.
   You can define `--arch` here, but do not forget about
   [disabling hardware acceleration](flow.md#Disabling hardware acceleration) if tou will run arm64 on amd64.
1. Run the setup with: `eden setup` - this does the following:
   * reads the configuration from your context and validates it
   * generates the certificates for adam and for eve
   * generates the config directory for eve, which includes the above certificates,
   as well as a `server` file pointing at the soon-to-be-started adam
   * gets a live eve image. This can be taken from one of:
   retrieved from your local docker cache; downloaded from docker hub; build
1. `eden start` - this does the following:
   * start redis in docker
   * start adam in docker
   * start docker registry server in docker
   * start eserver in docker
   * start eve in qemu
1. `eden eve onboard` - this does the following:
   * waits for eve to generate its device certificate
   * loads the device certificate into adam
   * waits for eve to onboard successfully to adam

To summarize:

```console
eden config add default
eden setup
eden start
eden eve onboard
```

### Modifying the Flow

You can modify the flow by passing options to various commands.
While the above flow controls everything, you can use only certain parts of it.
Common use cases are:

* Using a different live eve image, e.g. building a custom eve image,
but launching and controlling it via eden
* Disabling hardware acceleration, e.g. when running in unsupported nested virtualization
* Running onboarding manually

#### Setting the live EVE image

To use a different live EVE image, pass it when calling `eden setup`:

```console
eden setup --eve-tag <tag>
```

For example:

```console
eden setup --eve-tag 0.0.0-worker-rationalize-5a70468d
```

Note (this is very important) that eden will append the `HV` variant
and `ARCH` to get the platform. So if you pass in `--eve-tag abcdefg`,
then it will look, first in the local docker engine and then in
the registry, for `lfedge/eve:abcdefg-kvm-amd64`.

#### Disabling hardware acceleration

Sometimes, when running in a virtualized platform, like qemu,
on another virtualized platform, like an arm64 instance on amd64, you might want to disable
hardware acceleration, as it is not available. In that case, you can pass in:

```console
eden start --eve-accel=false
```

#### Manual Onboarding

To onboard manually, simply skip the `eden eve onboard` step.
The eve device already is configured to generate its device certificate
and attempt to communicate with the controller in `/var/config/server`,
i.e. the adam device. You simply skip the `eden eve onboard` step,
and communicate directly with adam.

`adam` can be controlled using the `adam admin` command.
If you run `eden status`, it will tell you exactly where it is reachable. If you
do not have the `adam` command installed, you can do so via the docker container:

```sh
$ docker exec -it eden_adam sh
# adam admin
```

## Starting Edge Containers

You can start several kinds of edge containers:

* docker containers (i.e. OCI containers)
* VMs from a URL
* VMs from a file
* VMs from an OCI image (i.e. VM disk wrapped in an OCI image)

In all cases, once your device is onboarded, just run the following to see the status:

```console
eden pod ps
```

### Docker Container

Run:

```console
eden pod deploy docker://docker.io/library/nginx:latest
```

Replace the provided image and tag with the one you want to deploy.

Several notes:

* The URL _must_ start with `docker:` as the protocol
* You must provide the full URL, including hostname and,
if relevant, `library`; `eden pod deploy` follows the official OCI rules,
and does not automatically insert `docker.io` for no-hostname
or `library` for no organization

### VMs from a URL

Run:

```console
eden pod deploy https://hostname/path/to/file.img
```

### VMs from a File

Run:

```console
eden pod deploy file:///full/path/to/file.img
```

EVE itself obviously doesn't support loading from a file,
since your filesystem is not accessible to it.
When you deploy a VM from a file, eden loads the file into
the `eserver` container, which then acts as an http server to
the running EVE device.

### VMs from an OCI Image

Run:

```console
eden pod deploy docker://docker.io/org/repository:tag --format qcow2
```

The image _must_ be structured in
the [edge containers](https://github.com/lf-edge/edge-containers) format.

## Updating the Base OS

To update the base OS, you need to take a few steps:

1. Ensure you have a different base OS image ready
1. Give the updated image an appropriate filename and location
1. Wait

### Getting a New Base OS Image

If you do not have a ready new base OS image ready, for example,
if you are testing a specific commit and build of EVE,
you can do the following:

1. Make a minor change to any file in the [eve](github.com/lf-edge/eve) repository
1. Create a new build with `make rootfs` or, if controlling the hypervisor,
`make rootfs HV=kvm`; use whatever options are appropriate for your use case
1. Track the location of the new image.

Note that you are _not_ building a docker image using `make eve`, or
a live image using `make live`, but instead _just_ the root filesystem
using `make rootfs`.

The image generally will be available in the eve repository under
`dist/amd64/installer/`, with a filename `rootfs-<tag>-<hypervisor>-<arch>.squashfs`
For example, as of this writing:

```console
$ ls -l dist/amd64/installer/
drwxr-xr-x 3 ubuntu ubuntu      4096 Jul  9 22:05 EFI
drwxr-xr-x 2 ubuntu ubuntu      4096 Aug  4 18:17 boot
-rw-rw-r-- 1 ubuntu ubuntu   1048576 Sep 11 08:11 config.img
-rw-r--r-- 1 ubuntu ubuntu   7703781 Jul  9 22:05 initrd.img
-rw-rw-r-- 1 ubuntu ubuntu   1048576 Oct 12 16:55 persist.img
-rw-rw-r-- 1 ubuntu ubuntu 234586112 Sep 21 15:58 rootfs-0.0.0-baseosmgr-cas-4b49b726-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234586112 Sep 21 18:35 rootfs-0.0.0-baseosmgr-cas-4b9b72622-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233467904 Oct  1 09:48 rootfs-0.0.0-ingestblob-check-1402f5aa-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233467904 Oct  2 11:11 rootfs-0.0.0-ingestblob-loop-a112b988-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233467904 Oct  2 11:54 rootfs-0.0.0-ingestblob-loop-ed375df2-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233467904 Oct  2 09:08 rootfs-0.0.0-ingestblob-loop-f35d3ab1-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234938368 Oct 12 17:38 rootfs-0.0.0-master-6ab0928d-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233426944 Sep 23 18:16 rootfs-0.0.0-worker-load-cas-11c5b1fc-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233426944 Sep 23 15:30 rootfs-0.0.0-worker-load-cas-1593c185-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233422848 Sep 23 13:44 rootfs-0.0.0-worker-load-cas-2012b74c-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233426944 Sep 23 18:29 rootfs-0.0.0-worker-load-cas-28535ae8-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233422848 Sep 23 10:31 rootfs-0.0.0-worker-load-cas-4d145b3a-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233426944 Sep 23 17:16 rootfs-0.0.0-worker-load-cas-63cad1e8-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233426944 Sep 23 16:47 rootfs-0.0.0-worker-load-cas-6c80fc3a-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233426944 Sep 23 18:12 rootfs-0.0.0-worker-load-cas-8319f9c8-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233426944 Sep 23 17:42 rootfs-0.0.0-worker-load-cas-83ac14c2-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 233426944 Sep 23 14:47 rootfs-0.0.0-worker-load-cas-fa72a3a6-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234930176 Oct 12 18:01 rootfs-0.0.0-worker-rationalize-2add1df5-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234930176 Oct 12 16:56 rootfs-0.0.0-worker-rationalize-431dce11-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234930176 Oct 14 08:37 rootfs-0.0.0-worker-rationalize-5a70468d-dirty-2020-10-14.08.33-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234930176 Oct 14 09:09 rootfs-0.0.0-worker-rationalize-5a70468d-dirty-2020-10-14.09.07-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234930176 Oct 14 09:57 rootfs-0.0.0-worker-rationalize-5a70468d-dirty-2020-10-14.09.55-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234930176 Oct 13 07:55 rootfs-0.0.0-worker-rationalize-5a70468d-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234930176 Oct 12 18:20 rootfs-0.0.0-worker-rationalize-87762bee-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234934272 Oct 13 07:11 rootfs-0.0.0-worker-rationalize-92f78ce9-dirty-2020-10-13.07.09-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234930176 Oct 13 07:40 rootfs-0.0.0-worker-rationalize-92f78ce9-dirty-2020-10-13.07.38-kvm-amd64.squash
-rw-rw-r-- 1 ubuntu ubuntu 234934272 Oct 12 18:56 rootfs-0.0.0-worker-rationalize-92f78ce9-kvm-amd64.squash
lrwxrwxrwx 1 ubuntu ubuntu        80 Oct 14 09:57 rootfs-kvm.img -> rootfs-0.0.0-worker-rationalize-5a70468d-dirty-2020-10-14.09.55-kvm-amd64.squash
lrwxrwxrwx 1 ubuntu ubuntu        14 Oct 14 09:57 rootfs.img -> rootfs-kvm.img
```

Conversely, if you have a base OS image from somewhere, simply download
it and place it in a location.

Note that the updated image must be distinct from the one that is running.
If the hash is identical, nothing will be updated.

### Update the Image

To update the image, run:

```console
eden controller edge-node eveimage-update -m adam:// <path-to-image>
```

or if you want to update from docker image (you should use image in full notation for example
`lfedge/eve:0.0.0-master-ad0d9030-kvm-amd64`):

```console
eden controller edge-node eveimage-update -m adam:// docker://lf-edge/eve:<tag of eve>
```

As with running a VM image from a file, this will load the image up to
the `eserver`, and then loaded up to EVE.

Note that eden sets default adam IP and port if you omit them in command,
but you can provide the adam IP and port.
This is available from running `eden status`, e.g.:

```console
$ eden status
✔ Adam status: container with name eden_adam is running
        Adam is expected at https://172.31.15.153:3333
        For local Adam you can run 'docker logs eden_adam' to see logs
```

So we would use `-m adam://172.31.15.153:3333`

EVE creates eve_version file in the same directory as image, so EDEN will try to use it for obtain version.
If the file does not exist and your filename does not precisely match the required pattern
(`<semver>-<free-form-text>-<hypervisor>-<arch>.squashfs`), you can override it as follows:

```console
eden controller edge-node eveimage-update -m adam:// --os-version=0.0.0-12345-kvm-amd64 <path-to-file>
```

The options are:

* `--os-version=<version>` - use the provided version, which
must match the pattern of `<semver>-<free-form-text>-<hypervisor>-<arch>`

### Wait

Wait for the update to take. Of course, you can use `eden log -f` to see the logs.
