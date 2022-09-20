# SDN Example with Transparent Proxy

Network model for this example is described by [network-model.json](./network-model.json).
Notice that EVE is connected into a single network (logical label `network0`), with 
transparent proxying enabled. As a result, every HTTP/HTTPS request sent from the device
will get transparently redirected into a proxy. The proxy performs MITM proxying for all
destinations (including the Adam controller), except for the `zededa.com` destination
(which is only added as an example and not really accessed by the device) - see `proxyRules`
configured for `transparentProxy`.
Both the device and the proxy use the same DNS server `my-dns-server`. In fact, for ever
HTTP/HTTPS request, the destination address will be translated twice - first by the device
(which is made redundant by the redirection) and then by the proxy.

For HTTPS, the proxy learns the destination address (which is lost by redirection)
from the SNI value in the TLS ClientHello message. Then it uses its own CA certificate
to generate destination server certificate on the fly and use it to create TLS
connection between itself and the client. Another TLS tunnel is created between the proxy
and the destination server. In other words, the proxy performs TLS termination, which allows
it to read what is otherwise encrypted content.
Obviously, for the client to accept this TLS connection, the CA certificate of the proxy
has to be installed as trusted. In EVE, this is done inside the network configuration,
where the certificate is Base64-encoded (without new lines) and put into the list
of trusted CA certificates under the `proxyCertPEM` field.

Note that `my-client` endpoint is not essential to the example, but can be used to test
connectivity from the "outside" and into the device, for example to exercise port forwarding rules.
It is also possible to connect to the device directly from the host and through the SDN VM using
the `eden sdn fwd` command (use with `--help` to learn more).


## Device onboarding

In order for the device to successfully access the controller, it needs to have the CA
certificate of the proxy installed as trusted. A proper device configuration for this scenario is
provided in [device-config.json](./device-config.json). This is applied into the Adam controller
using `eden controller edge-node set-config` command. However, for the device to even onboard,
it needs to learn at least the connectivity-related part of the configuration BEFORE it tries
to access the controller for the very first time.

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
./eden setup --eve-bootstrap-file $(pwd)/sdn/examples/tproxy/device-config.json
./eden start --sdn-network-model $(pwd)/sdn/examples/tproxy/network-model.json 
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/tproxy/device-config.json 
```

### 2. Using override.json

A legacy method ([bootstrap config](#1-using-bootstrap-config) is the preferred method)
where [DevicePortConfig][DPC] (network configuration for uplink interfaces) is prepared
manually as a JSON file (typically starting from the [template][override-template])
and copied into the `EVE` partition of the EVE installer, under the path `DevicePortConfig/override.json`.

See [override.json](./config-overrides/DevicePortConfig/override.json) prepared for
this example with the proxy. Provided is also [global.json](./config-overrides/GlobalConfig/global.json),
installed into the `EVE` partition of the installer under `GlobalConfig/global.json`,
with settings disabling the `lastresort` network config, which would not provide working
controller connectivity in this example anyway (`global.json` is not necessary for
successful onboarding, but it is recommended).

To test this approach using eden, run (from the repo root directory):

```
make clean && make build-tests
./eden config add default
./eden setup --eve-config-dir $(pwd)/sdn/examples/tproxy/config-overrides
./eden start --sdn-network-model $(pwd)/sdn/examples/tproxy/network-model.json 
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/tproxy/device-config.json 
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
The same [override.json](./config-overrides/DevicePortConfig/override.json) provided
for the [override method](#2-using-overridejson) can be used here as well.

To test this approach using eden, run (from the repo root directory):

```
make clean && make build-tests
./eden config add default
./eden setup 
./eden start --sdn-network-model $(pwd)/sdn/examples/tproxy/network-model.json \
             --eve-usbnetconf-file $(pwd)/sdn/examples/tproxy/config-overrides/DevicePortConfig/override.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/tproxy/device-config.json 
```



[DPC]: https://github.com/lf-edge/eve/blob/8.10/pkg/pillar/types/zedroutertypes.go#L473-L487
[override-template]: https://github.com/lf-edge/eve/blob/8.10/conf/DevicePortConfig/override.json.template
[makeusbconf]: https://github.com/lf-edge/eve/blob/8.10/tools/makeusbconf.sh
