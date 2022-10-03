# EVE Images

In order to launch an EVE instance, a base EVE disk image is required.
Eden takes care of retrieving the disk image and launching EVE for you.
However, you may want to customize it. For example, you may want
to use a different version of the official distribution, you may want
to use a different version, you may want to use a different disk image entirely.

This is especially useful as part of the EVE development lifecycle.

Retrieving disk images almost always is performed as part of `eden setup`.
The actual launching of an image is either part of `eden start` (when running
local qemu) or performed externally (for external devices).

## Eden EVE Image Steps

In the [Running Eden](../README.md#Running_Eden) guide, we described the
image setup steps within a context as follows:

1. create a named context to store all of your configuration - `eden config add <name>`
1. (optional) set options for the context - `eden config set <name> [options...]`
1. run setup - `eden setup --config <name>`, which extracts an eve-os qcow2 disk image from the docker image named in the context
1. start Eden's components - `eden start --config <name>`
   * if running EVE as qemu, entirely under eden control, it will start automatically
   * if running EVE as a separate EVE device:
     1. create or download the EVE image you want
     1. flash the image to the device's storage
     1. start the device

There are 3 elements here that control how an EVE image gets provisioned:

* `eden config set` - sets the OCI registry URL for an image that contains an eve-os bootable disk image
* `eden setup` - uses the registry URL to extract the eve-os bootable disk image, and reports to the user the path to that extracted disk image
* `eden start` - starts eden components and, if running local qemu, starts a virtual device booting from the extracted disk image

## Device Image Options

With two possibilities for device - local qemu and external device - and two
possibiliities for eve-os image - standard and custom - there are four possible
combinations of device and EVE image:

* local qemu, standard distributed eve-os image
* local qemu, custom eve-os image
* external device, standard distributed eve-os image
* external device, custom eve-os image

### QEMU and Standard

This is the most common use case. You run `eden`, which launches EVE in qemu
running a standard distributed eve-os.

The flow is as follows:

1. `eden config add <name>` - create a named config
1. `eden config set <name> --key eve.tag --value <tag>` - instruct it to use the desired eve tag from docker.io/lfedge/eve
1. `eden setup --config <name>` - in addition to normal eden setup
    1. pull the image `docker.io/lfedge/eve:<tag>`
    1. extract the disk image from the docker image
    1. configure the `config` partition customized for the context
    1. save the generated disk image to a local cache, normally `$PWD/dist/<name>-images/eve/`
    1. report to the user the path to the extracted disk image
1. `eden start` - start EVE in a virtual device via qemu, using the extracted disk as the boot device

### QEMU and Custom

TO have eden tell qemu to run your custom EVE image, you simply need to:

1. Make sure it is saved on the local device as a docker image with a unique tag
1. Tell eden to use that unique tag
1. Follow the usual steps of `eden setup` and `eden start`

The flow, then, is as follows:

1. Generate an EVE image in docker image format, saving it to `lfedge/eve:<some custom tag>`
1. `eden config add <name>` - create a named config
1. `eden config set <name> --key eve.tag --value <some custom tag>` - instruct it to use the desired eve tag from docker.io/lfedge/eve, which it will find where you placed it, in the local cache
1. `eden setup --config <name>` - generate the bootable disk image from the image in the docker cache
1. `eden start --config <name>` - start EVE in a virtual device via qemu, using the extracted disk as the boot device

For detailed instructions on how to generate the correct docker image, see
[Docker Image][Docker Image].

If you do not want to, or cannot, save the EVE image as a docker image, and have
the bootable image ready, you can do the following:

1. `eden config add <name>` - create a named config
1. `eden setup --config <name>` - generate the bootable disk image from the image in the docker cache
1. `eden start --config <name> --image-file=path/to/your/live-image` - start EVE in a virtual device via qemu, using the disk file at the provided path as the boot device

Note that in the live image option, `eden setup` _still_ will extract a file and
configure it, but it will not be used. It is your responsibility to configure
your custom bootable image as desired, for example controller address and
certificates.

### External and Standard

This use case is almost identical to [QEMU and Standard][QEMU and Standard]. The
only differences are:

* we extract the image after `eden setup` to flash it to our device
* we start our external device on our own
* we instruct `eden start` _not_ to start EVE via qemu; see [Starting EVE Locally][Starting EVE Locally]

The flow is as follows:

1. `eden config add <name> [--devmodel <model>]` - create a named config, optionally using a specific device model
1. `eden config set <name> --key eve.tag --value <tag>` - instruct it to use the desired eve tag from docker.io/lfedge/eve
1. (optional) `eden config set <name> --key eve.remote --value true` - required only if you did not pick a device model
1. `eden setup --config <name>` - this will report the path to the generated bootable disk image
1. flash the generated bootable disk image to your device's storage
1. start your device
1. `eden start --config <name>` - start all eden components except for EVE

### External and Custom

This use case is a combination of [QEMU and Custom][QEMU and Custom] and
[External and Standard][External and Standard].

* we generate the custom image before `eden setup`
* we extract the image after `eden setup` to flash it to our device
* we start our external device on our own
* we instruct `eden start` _not_ to start EVE via qemu; see [Starting EVE Locally][Starting EVE Locally]

The flow is as follows:

1. Generate an EVE image in docker image format, saving it to `lfedge/eve:<some custom tag>`
1. `eden config add <name> [--devmodel <model>]` - create a named config, optionally using a specific device model
1. `eden config set <name> --key eve.tag --value <some custom tag>` - instruct it to use the desired eve tag from docker.io/lfedge/eve
1. (optional) `eden config set <name> --key eve.remote --value true` - required only if you did not pick a device model
1. `eden setup --config <name>` - this will report the path to the generated bootable disk image
1. flash the generated bootable disk image to your device's storage
1. start your device
1. `eden start --config <name>` - start all eden components except for EVE

For detailed instructions on how to generate the correct docker image, see
[Docker Image][Docker Image].

If you do not want to, or cannot, save the EVE image as a docker image, and have
the bootable image ready, you can do the following:

1. `eden config add <name>` - create a named config
1. `eden setup --config <name>` - generate the bootable disk image from the image in the docker cache
1. `eden start --config <name> --image-file=path/to/your/live-image` - start EVE in a virtual device via qemu, using the disk file at the provided path as the boot device

Note that in the live image option, `eden setup` _still_ will extract a file and
configure it, but it will not be used. It is your responsibility to configure
your custom bootable image as desired, for example controller address and
certificates.

## Starting EVE Locally

`eden` decides whether or not to start a virtual device via QEMU with EVE on it,
based on the value of the key `eve.remote`:

* `false` (default): `eden` should start and control EVE locally
* `true`: the user has a remote device, `eden` should not start or control it locally

This can be set in one of two ways:

1. Explicitly: `eden config set <name> --key eve.remote --value true`
1. Implicitly: when creating a context with a device model _other than_ the default, the `eden config` command _also_ sets `eve.remote=true`

For example:

```console
eden config add mydevice --devmodel general
eden config add gcpinstance --devmodel GCP
```

With either of the above, `eden` will set `eve.remote=true` for you.

## Generating a Custom EVE Image

If you want to generate your own custom EVE image, you have two options:

* generate a docker image with your live image (preferred)
* just run your live image.

### Docker Image

The advantage of the docker image, is that it contains the utility
to generate the appropriate format of image combined with the correct
config partition. You will not have to do any work to get the config partition
"just right" in your image.

To generate the docker container with your image:

1. Work in the `github.com/lf-edge/eve` directory
1. Configure your code as desired
1. Run `make eve`, optionally setting the desired hypervisor, e.g. `make eve HV=kvm` (recommended with eden)

Note: If you build EVE with xen hypervisor (`make eve`), you should run
`eden config set default --key eve.hv --value xen` before `eden setup`.

When done, you will be provided with output telling you
the docker image name and tag, e.g.

```console
Successfully built a46458b4ce1a
Successfully tagged lfedge/eve:0.0.0-testbranch-b6a6d6fd-kvm-amd64
Tagging lfedge/eve:0.0.0-testbranch-b6a6d6fd-kvm-amd64 as lfedge/eve:0.0.0-testbranch-b6a6d6fd-kvm
```

You now can use the tag in `eden config set` or `eden setup`. In the above
example, the tag is `0.0.0-testbranch-b6a6d6fd`.

```sh
eden setup --eve-tag 0.0.0-testbranch-b6a6d6fd
```

Or you can save it, by setting it in the file:

```console
eden config set default --key eve.tag --value 0.0.0-testbranch-b6a6d6fd
eden setup
```

eden now will use the above container image to generate and configure
the live disk image.

### Live Image

To generate the live image:

1. Switch to the `github.com/lf-edge/eve` directory
1. Configure your code as desired
1. Run `make live`, optionally setting the desired hypervisor, e.g. `make live HV=kvm` (recommended with eden). When building you must include the config directory generated by `eden setup` by adding `make live CONF_DIR=<eden-conf-dir>`

When done, you have a live image file to be used, normally in `dist/<arch>/<file>`, e.g. `dist/amd64/live.qcow2`.
You can use that on your own external device, or when running via qemu as:

```console
eden start --image-file=path/to/your/live-image
```

### Overwrite config of EVE

You can add files into config partition of EVE (along with the files that are generated by EdenEden) by copying them into `eve-config-dir` directory.
You can select another directory you want with `--eve-config-dir` flag of `eden setup` command. To read more about config files please see
[EVE configuration readme](https://github.com/lf-edge/eve/blob/master/docs/CONFIG.md).
