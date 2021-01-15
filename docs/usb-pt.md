# Host USB passthrough support

To pass host USB devices to guests, it is necessary to have the appropriate
descriptors for the devices in the `"deviceIoList"` property of the config.
These represent the device addresses on the host. In typical usecases, these
addresses do not change, since the topology is stable. Descriptor format is as
follows:

```json
        {
            "ptype": 2,
            "phylabel": "USB3:4",
            "phyaddrs": {
                "UsbAddr": "3:5"
            },
            "logicallabel": "USB3:4",
            "assigngrp": "USB6",
            "usage": 3
        }
```

`UsbAddr` is in the form `Bus:Address` (can be obtained via sysfs lookup
or `lsusb` on the EVE host)

## EDEN support

Eden by default generates a default edge device config for qemu, containing
9 device descriptors. For QEMU, additional devices can be specified in
the config file created by EDEN or via QEMU command line parameters
(please refer to the QEMU documentation for specific device types).
To assign a device to a guest VM, the `--adapters` option
is used during the deployment stage, e.g.:

```console
./eden pod deploy -p 8027:22 https://cloud-images.ubuntu.com/releases/groovy/release-20210108/ubuntu-20.10-server-cloudimg-amd64.img \
     -v debug --metadata='#cloud-config\npassword: passw0rd\nlock_passwd: False\nchpasswd: { expire: False }\nssh_pwauth: True\n'\
     --adapters USB3:4
```

This will assign an adapter with `phylabel` "USB3:4" to the newly
created VM instance. Devices from the same group are assigned together,
but currently they need to be specified in the --adapters parameter
as a comma-separated list.
