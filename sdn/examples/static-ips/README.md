# SDN Example with Static IP Configuration

Network model for this example is described by [network-model.json](./network-model.json).
Notice that EVE is connected into a single network (logical label `network0`), with 
DHCP service *disabled*. This means that for the device to have working connectivity,
it needs to have (static) IP settings provided inside the network config (`EdgeDevConfig.Networks`).
Device needs to learn what IP address it should assign to the (`eth0`) interface, what
subnet the address belongs to, what is the default gateway, which DNS servers to use, etc.

Note that `my-client` endpoint is not essential to the example, but can be used to test
connectivity from the "outside" and into the device, for example to exercise port forwarding rules.
It is also possible to connect to the device directly from the host and through the SDN VM using
the `eden sdn fwd` command (use with `--help` to learn more).

## Device onboarding

In order for the device to successfully access the controller, it needs to get and apply
IP settings for the single network interface. A proper device configuration for this scenario is
provided in [device-config.json](./device-config.json) (notice the `ip` config for the single
network). This is applied into the Adam controller using `eden controller edge-node set-config`
command. However, for the device to even onboard, it needs to learn at least the connectivity-related
part of the configuration BEFORE it tries to access the controller for the very first time.

This chicken-and-egg problem can be solved using one of the three supported methods:

### 1. Using bootstrap config

In this case, the same [device configuration](./device-config.json), which is applied into
the controller, is also installed into the config partition as `bootstrap-config.pb`,
signed using the controller's certificate and binary-proto marshalled (it is not intended
to be manually edited, but rather exported from the controller).

To test this approach using eden, run (from the repo root directory):

```
make clean && make build-tests
./eden config add default
./eden setup --eve-bootstrap-file $(pwd)/sdn/examples/static-ips/device-config.json
./eden start --sdn-network-model $(pwd)/sdn/examples/static-ips/network-model.json 
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/static-ips/device-config.json 
```

### 2. Using override.json

A legacy method ([bootstrap config](#1-using-bootstrap-config) is the preferred method)
where [DevicePortConfig][DPC] (network configuration for uplink interfaces) is prepared
manually as a JSON file (typically starting from the [template][override-template])
and copied into the `EVE` partition of the EVE installer, under the path `DevicePortConfig/override.json`.

See [override.json][override-json] prepared for this example with static IP settings.
Provided is also [global.json][global-json], installed into the `EVE` partition
of the installer under `GlobalConfig/global.json`, with settings disabling the `lastresort`
network config, which would not provide working controller connectivity in this example
anyway (`global.json` is not necessary for successful onboarding, but it is recommended).

To test this approach using eden, run (from the repo root directory):

```
make clean && make build-tests
./eden config add default
./eden setup --eve-config-dir $(pwd)/sdn/examples/static-ips/config-overrides
./eden start --sdn-network-model $(pwd)/sdn/examples/static-ips/network-model.json 
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/static-ips/device-config.json 
```

### 3. Using usb.json

A legacy method ([bootstrap config](#1-using-bootstrap-config) is the preferred method) where
[DevicePortConfig][DPC] (network configuration for uplink interfaces) is prepared manually
as a JSON file (typically starting from the [template][override-template]) and build into
a specially formatted USB stick, plugged into the device AFTER the EVE OS was installed
(i.e. the EVE installer and the drive carrying `usb.json` are two separate USB sticks).
Image for this special USB stick with network config is normally created using
[makeusbconf.sh script][makeusbconf].
However, with eden it is only required to prepare the json file (which can have any name,
not necessarily `usb.json`) and eden will prepare the drive and "insert" it into the EVE VM.
The same [override.json][override-json] provided for the [override method](#2-using-overridejson)
can be used here as well.

To test this approach using eden, run (from the repo root directory):

```
make clean && make build-tests
./eden config add default
./eden setup 
./eden start --sdn-network-model $(pwd)/sdn/examples/static-ips/network-model.json \
             --eve-usbnetconf-file $(pwd)/sdn/examples/static-ips/config-overrides/DevicePortConfig/override.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/static-ips/device-config.json 
```

## Single-use EVE installer with Bootstrap config

It is possible to instruct Eden to install and run EVE from a (single-use) EVE installer image,
generated by the controller with the bootstrap config included.
For example, to run this example with (proprietary) zedcloud instead of (open-source) Adam
controller, follow these steps:

1. Access zedcloud UI and create a new edge node with [ZedVirtual-4G model](../../../models/template_l1_ZedVirtual-4G.json).\
   Enable and configure only `eth0` interface. It should be used for both management and applications.\
   Assign network to `eth0` with static IP configuration:

```
subnet: 172.22.12.0/24
gateway: 172.22.12.1
eth0 IP: 172.22.12.10
```

2. Export a single-use EVE installer image for your device using zcli.\
   Consider also disabling last-resort network configuration (`network.fallback.any.eth:disabled`)
   (it will not provide working connectivity).

3. Assuming that the path to the installer is `$(pwd)/installer.raw`, run the following
   command to start your virtual device using eden:

```
make clean && make build-tests
./eden config add default
./eden config set default --key eve.disks --value 1
./eden config set default --key eve.disk --value 4096
./eden config set default --key eve.custom-installer.path --value "$(pwd)/installer.raw"
./eden config set default --key eve.custom-installer.format --value raw
./eden setup
./eden start --sdn-network-model $(pwd)/sdn/examples/static-ips/network-model.json
```

Note that traffic coming out from EVE VM will go through the SDN VM (through network
without DHCP service), then it will be S-NATed on the exit from the VM, continue into
the host networking and out into the Internet towards the zedcloud.


[override-json]: ./config-overrides/DevicePortConfig/override.json
[global-json]: ./config-overrides/GlobalConfig/global.json
[DPC]: https://github.com/lf-edge/eve/blob/8.10/pkg/pillar/types/zedroutertypes.go#L473-L487
[override-template]: https://github.com/lf-edge/eve/blob/8.10/conf/DevicePortConfig/override.json.template
[makeusbconf]: https://github.com/lf-edge/eve/blob/8.10/tools/makeusbconf.sh
