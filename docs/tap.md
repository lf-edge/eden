# Use tap interface to connect to app

## Create interfaces

```console
sudo tunctl -t tap0 -u `whoami`
sudo brctl addbr br0
sudo brctl addif br0 tap0
sudo ip link set up dev br0
sudo ip link set up dev tap0
sudo ip a add 11.11.11.1/24 dev br0
```

We define IP to access from the host to app inside EVE

## Create dnsmasq config (sample below)

```console
log-queries
log-dhcp
bind-interfaces
except-interface=lo
dhcp-leasefile=/run/dnsmasq.leases
interface=br0
dhcp-range=11.11.11.3,11.11.11.114,60m
dhcp-option=option:router,11.11.11.1
dhcp-host=d0:50:99:82:e7:2b,11.11.11.21
```

In the sample we define static host with MAC d0:50:99:82:e7:2b and IP 11.11.11.21.

## Run dnsmasq

`sudo dnsmasq --no-daemon --log-queries --conf-file=/conf/dhcp.conf`, where `/conf/dhcp.conf` - is path to configuration file.
*You can use your favourite DHCP server here.*

## Prepare and run EVE

```console
make build
./eden config add default
./eden setup
./eden start --with-tap=tap0
./eden eve onboard
```

## Create network and application

```console
./eden network create --type=switch --name=tap-net --uplink=eth2
./eden pod deploy docker://lfedge/eden-eclient:83cfe07 --networks=tap-net:d0:50:99:82:e7:2b
```

We use eth2 here, because of tap interface is connected as the third one (eth0 and eth1 used with qemu`s user networks).
We define the same MAC for the app as we described in dnsmasq config above.

## Try to connect to app

`ssh -i tests/eclient/image/cert/id_rsa root@11.11.11.21`
