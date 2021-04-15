# IPXE boot

You can ipxe to boot and install EVE onto your hardware. The booting process is
described [here](https://github.com/lf-edge/eve/blob/master/docs/BOOTING.md).

To make an image supporting EDEN, you need to have IP, which is resolvable by your hardware
(actually, you need to forward port of eden-http-server which one is `8888/tcp` by default
and Adam which one is `3333/tcp` by default).

Next, do the following (where `IP` below is the ip address of EDEN for access from EVE):

```bash
eden config add default --devmodel=general
eden config set default --key adam.eve-ip --value IP
eden setup --netboot=true
```

You will see in the output something like
`Please use /home/user/eden/dist/default-images/eve/tftp/ipxe.efi.cfg to boot your EVE via ipxe`.
You should add this file into your tftp server and point yours
dhcp option [67 Bootfile-Name](https://tools.ietf.org/html/rfc2132#section-9.5) onto it.

You can also use `ipxe.efi.cfg` uploaded into eserver, link to which will also be inside
output of setup command (something like
`ipxe.efi.cfg uploaded to eserver (http://IP:8888/eserver/ipxe.efi.cfg).`).

You can start your device and wait for installation process of EVE. Next, you can run
`eden start` and `eden eve onboard` as usual.

## Use Equinix Metal to boot EVE

You can use [Equinix Metal](https://metal.equinix.com/) for booting EVE on baremetal system.

### Step 1. Preparation

You must have access to API of Equinix Metal (PACKET_TOKEN and PROJECT_ID).

You must have node with publicly accessed IP address (forward of `8888/tcp` and `3333/tcp` ports) for use as Eden Node.
If you would like to deploy it on Equinix Metal you can start from `t1.small.x86` machine and deploy `Ubuntu 20.04 LTS`
on it.

Please, follow [Eden Install Prerequisites section](../README.md#Install Prerequisites),
[packet-cli installation](https://github.com/packethost/packet-cli#installation) and
[packet-cli authentication](https://github.com/packethost/packet-cli#authentication) via SSH to your Eden Node.

### Step 2. Eden configure and build

To build EVE via Eden and prepare for iPXE boot run the following on Eden Node
(please use `amd64` or `arm64` in `ARCH` placeholder and public IP of your Eden Node instead of `IP` placeholder):

```console
make build
./eden config add default --devmodel=general --arch ARCH
./eden config set default --key adam.eve-ip --value IP
./eden setup --netboot=true
```

### Step 3. EVE booting

To boot EVE with iPXE you should run:

```console
packet device create -f LOCATION -H NAME -i http://IP:8888/eserver/ipxe.efi.cfg -o custom_ipxe -P TYPE -p PROJECT_ID
```

where:

* `LOCATION` - datacenter you want to use (for example, `sjc1`)
* `NAME` - name of the node you want to create (for example, `eve-eden-node`)
* `IP` - public IP of your Eden Node
* `TYPE` - type of server you want to boot (for example, `c1.large.arm.xda` for arm64 or `t1.small.x86` for amd64)
* `PROJECT_ID` - your project on Equinix Metal

### Step 4. Onboarding of EVE

After booting the server (become active), you can run on your Eden Node the following:

```console
./eden start
./eden eve onboard
```

If you would like to build tests: `make build-tests`.

_Note: in this case you cannot use local docker registry deployed with Eden die to public IPs. Please, skip it in tests
with `EDEN_TEST_REGISTRY=n` environment variable._
