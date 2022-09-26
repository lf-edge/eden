# SDN Example with Explicit Proxy

Network model for this example is described by [network-model.json](./network-model.json).
Notice that EVE is connected into a single network (logical label `network0`), with
DHCP enabled. The DHCP server is configured to announce `dns-server-for-device` as the DNS
server to be used by the device. However, this DNS server is intentionally not able
to translate Adam (controller) domain name into the corresponding IP address (notice that
`staticEntries` for this DNS server does not include `mydomain.adam`). Moreover, the network
router is told not to provide route for endpoints outside the SDN VM (`"outsideReachability": false`).\
This, in combination with DNS settings, means that the device will not be able to access
the controller - unless it uses the provided HTTP/HTTPS proxy (logical label `my-proxy`).
This proxy is able to access the outside world (no firewall rules defined, i.e. all is allowed)
and furthermore it is using a different DNS server than the device, logically labeled
as `dns-server-for-proxy`, which is able to translate Adam's domain name.\
Note that DNS server used by the device (`dns-server-for-device`) is able to translate
only the domain name of the proxy. This is sufficient, because with explicitly configured
proxy DNS resolution of a destination address is being done by the proxy (which learns
the address from the header of the HTTPS CONNECT or HTTP GET/POST/etc. request),
not by the device. Destination IP address of requests coming out from the device is that
of the proxy.\
Also, the network to which the device is connected is configured to have a route leading
towards the proxy: `"reachableEndpoints": ["my-client", "dns-server-for-device", "my-proxy"]`.
The proxy performs MITM proxying for all destinations (including the Adam controller),
except for the `zededa.com` destination (which is only added as an example and not really
accessed by the device).

Note that `my-client` endpoint is not essential to the example, but can be used to test
connectivity from the "outside" and into the device, for example to exercise port forwarding rules.
It is also possible to connect to the device directly from the host and through the SDN VM using
the `eden sdn fwd` command (use with `--help` to learn more).


## Device onboarding

In order for the device to successfully access the controller, it needs to be told
to use the HTTP/HTTPS proxy. A proper device configuration for this scenario is
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
./eden setup --eve-bootstrap-file $(pwd)/sdn/examples/explicit-proxy/device-config.json
./eden start --sdn-network-model $(pwd)/sdn/examples/explicit-proxy/network-model.json 
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/explicit-proxy/device-config.json 
```

### 2. Using override.json

A legacy method ([bootstrap config](#1-using-bootstrap-config) is the preferred method)
where [DevicePortConfig][DPC] (network configuration for uplink interfaces) is prepared
manually as a JSON file (typically starting from the [template][override-template])
and copied into the `EVE` partition of the EVE installer, under the path `DevicePortConfig/override.json`.

See [override.json][override-json] prepared for this example with the proxy. Provided is
also [global.json][global-json], installed into the `EVE` partition of the installer under
`GlobalConfig/global.json`, with settings disabling the `lastresort` network config,
which would not provide working controller connectivity in this example anyway
(`global.json` is not necessary for successful onboarding, but it is recommended).

To test this approach using eden, run (from the repo root directory):

```
make clean && make build-tests
./eden config add default
./eden setup --eve-config-dir $(pwd)/sdn/examples/explicit-proxy/config-overrides
./eden start --sdn-network-model $(pwd)/sdn/examples/explicit-proxy/network-model.json 
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/explicit-proxy/device-config.json 
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
./eden start --sdn-network-model $(pwd)/sdn/examples/explicit-proxy/network-model.json \
             --eve-usbnetconf-file $(pwd)/sdn/examples/explicit-proxy/config-overrides/DevicePortConfig/override.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/explicit-proxy/device-config.json 
```

## Single-use EVE installer with Bootstrap config

It is possible to instruct Eden to install and run EVE from a (single-use) EVE installer image,
generated by the controller with the bootstrap config included.
For example, to run this example with (proprietary) zedcloud instead of (open-source) Adam
controller, follow these steps:

1. Access zedcloud UI and create a new edge node with ...\
   Enable and configure only `eth0` interface. It should be used for both management and applications.\
   Assign network to `eth0` with HTTP and HTTPS proxies configured with address `my-proxy.sdn`
   and ports `9090` and `9091`, respectively. Import the following certificate as the CA
   certificate of the proxy:

```
-----BEGIN CERTIFICATE-----
MIIDVTCCAj2gAwIBAgIUPGtlx1k08RmWd9RxiCKTXYnAUkIwDQYJKoZIhvcNAQEL
BQAwOjETMBEGA1UEAwwKemVkZWRhLmNvbTELMAkGA1UEBhMCVVMxFjAUBgNVBAcM
DVNhbiBGcmFuY2lzY28wHhcNMjIwOTA3MTcwMDE0WhcNMzIwNjA2MTcwMDE0WjA6
MRMwEQYDVQQDDAp6ZWRlZGEuY29tMQswCQYDVQQGEwJVUzEWMBQGA1UEBwwNU2Fu
IEZyYW5jaXNjbzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALQsi7IG
M8KApujL71MJXbuPQNn/g+RItQeehaFRcqcCcpFW4k1YveMNdf5HReKlAfufFtaa
IF368t33UlleblopLM8m8r9Ev1sSJOS1yYgU1HABjyw54LXBqT4tAf0xjlRaLn4L
QBUAS0TTywTppGXtNwXpxqdDuQdigNskqzEFaGI52IQezfGt7L2CeeJ/YJNcbImR
eCXMPwTatUHLLE29Qv8GQQfy7TpCXdXVLvQAyfZJi7lY7DjPqBab5ocnVTRcEpKz
FwH2+KTokQkU1UF614IveRF3ZOqqmrQvy1AdSvekFLIz2uP7xsfy3I3HNQcPJ4DI
5vNzBaE/hF5xK40CAwEAAaNTMFEwHQYDVR0OBBYEFPxOB5cxsf89x6KdFSTTFV2L
wta1MB8GA1UdIwQYMBaAFPxOB5cxsf89x6KdFSTTFV2Lwta1MA8GA1UdEwEB/wQF
MAMBAf8wDQYJKoZIhvcNAQELBQADggEBAFXqCJuq4ifMw3Hre7+X23q25jOb1nzd
8qs+1Tij8osUC5ekD21x/k9g+xHvacoJIOzsAmpAPSnwXKMnvVdAeX6Scg1Bvejj
TdXfNEJ7jcvDROUNjlWYjwiY+7ahDkj56nahwGjjUQdgCCzRiSYPOq6N1tRkn97a
i6+jB8DnTSDnv5j8xiPDbWJ+nv2O1NNsoHS91UrTqkVXxNItrCdPPh21hzrTJxs4
oSf4wbaF5n3E2cPpSAaXBEyxBdXAqUCIhP0q9/pgBTYuJ+eW467u4xWqUVi4iBtN
wVfYelYC2v03Rn433kv624oJDQ7MM5bDUv3nqPtkUys0ARwxs8tQCgg=
-----END CERTIFICATE-----
```

2. Export a single-use EVE installer image for your device using zcli.\
   Consider also disabling last-resort network configuration
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
./eden start --sdn-network-model $(pwd)/sdn/examples/explicit-proxy/network-model.json
```

Note that traffic coming out from EVE VM will go through the SDN VM (where it will be proxied),
then it will be S-NATed on the exit from the VM, continue into the host networking and out
into the Internet towards the zedcloud.


[override-json]: ./config-overrides/DevicePortConfig/override.json
[global-json]: ./config-overrides/GlobalConfig/global.json
[DPC]: https://github.com/lf-edge/eve/blob/8.10/pkg/pillar/types/zedroutertypes.go#L473-L487
[override-template]: https://github.com/lf-edge/eve/blob/8.10/conf/DevicePortConfig/override.json.template
[makeusbconf]: https://github.com/lf-edge/eve/blob/8.10/tools/makeusbconf.sh
