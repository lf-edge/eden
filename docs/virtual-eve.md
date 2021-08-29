# Running in VMs

When running the components that make up eden, most are run
as straight binaries or docker containers. One, the core EVE device,
runs as a VM via qemu, and it, in turn, may need to run further VMs.
In order to power this, when starting EVE devices, eden starts them
with hardware virtualization enabled. This is manifest in
the following commands being passed to qemu when started:

```console
--enable-kvm --cpu host
```

There are two instances where this might be a problem:

* Running on an older CPU without hardware virtualization support
* If you already are running on a VM, for example on a typical cloud provider instance

Running EVE in a VM, e.g. qemu, and then running workloads inside EVE as VMs, is
virtualization inside virtualization, or "nested virtualization".

While some cloud providers have started to support nested virtualization,
for example
[GCP](https://cloud.google.com/compute/docs/instances/enable-nested-virtualization-vm-instances),
many others do not.

If you tried to use nested virtualization without support, you might get an
error like this:

```console
Could not access KVM kernel module: No such file or directory
```

## How to Check

To check if this is a problem, you can do one of 2 things:

* Look for the above error message
* Check for the correct kernel module on Linux

The kernel module normally is `kvm_intel` on Intel devices:

```sh
lsmod | grep kvm
```

You should see both the `kvm` and the `kvm_intel` module.
If you do not, you do not have hardware acceleration support.

## Running Without

In order to enable eden to launch EVE in that environment,
you need to disable hardware acceleration. You can do that in one of two ways:

* Run `eden start` with the argument to disable it:

```sh
eden start --eve-accel=false
```

* Set the context for your device

```sh
eden config set default --key eve.accel --value false
```
