# SDN Examples with application IP routing configurations

Most of the edge deployments will deploy applications (VMs or containers) that will have
connectivity to both WAN (Internet) and LAN (shop floor, machine floor).
There may be a single WAN interface and multiple LAN interfaces. With the IP route
configurability of Network instances provided by EVE, we can automatically (zero-touch)
provision routing for applications that have access to both the Internet and one or more LANs.
This is possible because EVE allows to:

- use DHCP to propagate the connected routes to applications (routes for external networks
  that uplink ports are connected to)
- configure a set of static IP routes for a network instance and have them propagated
  to applications

Another common case is using one application as a network gateway for other applications
running on the same device. The gateway application may provide some network function(s),
such as firewall, IDS, network monitoring, etc. Such application will connect on one side with
the external network(s) using directly attached network adapter(s) or via switch network
instance(s), and the other side will make use of an air-gap local network instance to connect
with other applications running on the device. Propagated static IP routes are necessary
to make the application traffic flow through the gateway app. In theory, multiple network
functions can be chained together in this way using several air-gap network instances
with static IP routes.

## Example 1: Application with multiple uplink ports

[In this example](./multiple-uplinks), we connect a single application into 3 local network
instances, each with a different uplink port. Network instance `ni-eth0` is connected
to the management port `eth0`, `ni-eth1` uses app-shared uplink `eth1` with static IP
configuration and default route disabled by setting `0.0.0.0` as the gateway IP in the network
config, and finally `ni-eth2` with app-shared `eth2` running DHCP client but not getting
the Router option (code: 3) from the DHCP server (i.e. not installing default route).

Within the Eden-SDN environment, a dedicated HTTP "hello-world" server is deployed for each
of these uplink ports, accessible exclusively through its assigned port. Note that these
HTTP servers run in dedicated networks and not in the subnets of uplinks.

Here is a diagram depicting the network topology:

```text
          +---------+    +-------------+     +-------------+
   ------>| ni-eth0 |----| eth0 (mgmt) | ... | httpserver0 |
   |      +---------+    +-------------+     +-------------+
   |
+-----+   +---------+    +-------------------------+     +-------------+
| app |-->| ni-eth1 |----|    eth1 (app-shared)    | ... | httpserver1 |
+-----+   +---------+    | (static, no def. route) |     +-------------+
   |                     +-------------------------+
   |
   |      +---------+    +-------------------------+     +-------------+
   ------>| ni-eth2 |----|    eth2 (app-shared)    | ... | httpserver2 |
          +---------+    |  (DHCP, no def. route)  |     +-------------+
                         +-------------------------+
```

Under typical circumstances, when network instances are configured with default IP routing
settings, application would receive routes exclusively for the internal subnets of these
local network instances. Consequently, the application would lack information about
the specific interface to utilize when reaching endpoints within a particular uplink subnet
or beyond.

In order for the application to be able to access endpoints inside subnets of individual
uplinks, we *enable propagation of connected routes* for `ni-eth0` and `ni-eth1` (excluding
`ni-eth2` just to show that eth2 subnet will not be routed):

```text
  "networkInstances": [
    {
      "displayname": "ni-eth0",
      ...
      "propagateConnectedRoutes": true
    },
    {
      "displayname": "ni-eth1",
      ...
      "propagateConnectedRoutes": true
    },
    ...
  ]
```

This will result in the following routes added into the routing table of the application:

```text
172.22.12.0/24 via 10.50.0.1 dev eth0
192.168.55.0/24 via 10.50.1.1 dev eth1
```

However, HTTP servers are not inside these networks but further in the routing path.
Therefore, we have to add static routes targeting networks where servers are deployed:

```text
  "networkInstances": [
    {
      "displayname": "ni-eth0",
      ...
      "staticRoutes": [
        {
          "destinationNetwork": "10.20.20.0/24",
          "gateway": "10.50.0.1"
        }
      ]
    },
    {
      "displayname": "ni-eth1",
      ...
      "staticRoutes": [
        {
          "destinationNetwork": "10.21.21.0/24",
          "gateway": "192.168.55.1"
        }
      ]
    },
    {
      "displayname": "ni-eth2",
      ...
      "staticRoutes": [
        {
          "destinationNetwork": "10.22.22.0/24",
          "gateway": "10.140.2.1"
        }
      ]
    },
    ...
  ]
```

Note that gateway can be either from the uplink subnet (e.g. gateway of eth1 is `192.168.55.1`),
or the NI bridge IP itself (`10.50.0.1` for ni-eth0). The latter is used when uplink
IP addressing is not known ahead or can change in run-time. In such case it is better
to first route the selected traffic towards the NI bridge and then let the NI default
route received from uplink to route it further.

Please refer to the [network model](multiple-uplinks/network-model.json) to find out the IP
addressing used for uplinks and HTTP servers in this example.

Run the example with:

```shell
make clean && make build-tests
./eden config add default
./eden config set default --key sdn.disable --value false
./eden setup
./eden start --sdn-network-model $(pwd)/sdn/examples/app-routing/multiple-uplinks/network-model.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/app-routing/multiple-uplinks/device-config.json
```

Once deployed, login to the application:

```shell
./eden eve ssh
CONSOLE="$(eve list-app-consoles | grep cee082fd-3a43-4599-bbd3-8216ffa8652d | grep CONTAINER | awk '{print $4}')"
eve attach-app-console "$CONSOLE"
```

Inside the app, check IP routing table:

```shell
app$ ip route
default via 10.50.0.1 dev eth0
10.20.20.0/24 via 10.50.0.1 dev eth0
10.21.21.0/24 via 10.50.1.1 dev eth1
10.22.22.0/24 via 10.50.2.1 dev eth2
10.50.0.0/24 dev eth0 proto kernel scope link src 10.50.0.2
10.50.1.0/24 dev eth1 proto kernel scope link src 10.50.1.2
10.50.2.0/24 dev eth2 proto kernel scope link src 10.50.2.2
172.22.12.0/24 via 10.50.0.1 dev eth0
192.168.55.0/24 via 10.50.1.1 dev eth1
```

Try to ping uplink gateways to test the propagation of connected routes. Note that `ni-eth2`
does not have connected routes propagated:

```shell
app$ ping -c 3 172.22.12.1
PING 172.22.12.1 (172.22.12.1): 56 data bytes
64 bytes from 172.22.12.1: seq=0 ttl=63 time=1.446 ms
64 bytes from 172.22.12.1: seq=1 ttl=63 time=2.574 ms
64 bytes from 172.22.12.1: seq=2 ttl=63 time=4.058 ms

--- 172.22.12.1 ping statistics ---
3 packets transmitted, 3 packets received, 0% packet loss
round-trip min/avg/max = 1.446/2.692/4.058 ms

app$ ping -c 3 192.168.55.1
PING 192.168.55.1 (192.168.55.1): 56 data bytes
64 bytes from 192.168.55.1: seq=0 ttl=63 time=2.691 ms
64 bytes from 192.168.55.1: seq=1 ttl=63 time=2.555 ms
64 bytes from 192.168.55.1: seq=2 ttl=63 time=3.555 ms

--- 192.168.55.1 ping statistics ---
3 packets transmitted, 3 packets received, 0% packet loss
round-trip min/avg/max = 2.555/2.933/3.555 ms

app$ ping -c 3 10.140.2.1
PING 10.140.2.1 (10.140.2.1): 56 data bytes

--- 10.140.2.1 ping statistics ---
3 packets transmitted, 0 packets received, 100% packet loss
```

Next, try to access HTTP servers to exercise static routes. Each will be routed through
a different app interface (`httpserver1` and `httpserver2` would not be accessible if
default route was taken):

```shell
app$ curl 10.20.20.70/helloworld
Hello world from HTTP server no. 0
app$ curl 10.21.21.70/helloworld
Hello world from HTTP server no. 1
app$ curl 10.22.22.70/helloworld
Hello world from HTTP server no. 2
```

Finally, in terms of the default route, only the DHCP server of `ni-eth0` will advertise it
(with NI bridge IP as the gateway). This is because both `eth1` and `eth2` are app-shared
(not for management traffic) and have no default route of their own. EVE policy is defined
to not propagate default route in such case. In this example, this leads to (typically desired)
state of the application having only one default route.

## Example 2: Application used as network gateway

[In this example](./app-gateway), we deploy 3 applications, where one of them will act as
an IP routing gateway for the other two. This simulates a scenario where "gateway" application
is used to provide some network function, such as firewall. However, in this case the app merely
routes traffic between the other two "client" applications and external endpoints. Air-gap network
instances are used to interconnect client apps with the gateway app. One of the client apps
directs a subset of its traffic through the gateway app, while the other designates
it as the default gateway for all traffic.
Inside Eden-SDN, we deploy an HTTP "hello-world" server to emulate an external endpoint.

Here is a diagram depicting the network topology:

```text
                  +---------+    +-------------+     +-------------+
       ---------->| ni-eth0 |----| eth0 (mgmt) | ... | httpserver0 |
       |          +---------+    +-------------+     +-------------+
       |
+-------------+   +---------+
| app-client1 |-->| airgap1 |
+-------------+   +---------+
                       |
                       v
                  +--------+    +------------------+    +-------------------+     +-------------+
                  | app-gw |--->| ni-eth1 (switch) |----| eth1 (app-shared) | ... | httpserver1 |
                  +--------+    +------------------+    +-------------------+     +-------------+
                       ^
                       |
+-------------+   +---------+
| app-client2 |-->| airgap2 |
+-------------+   +---------+
```

To make this possible, `airgap1` is configured with static IP route that sends traffic destined
to the network with `httpserver1` via `app-gw` IP. `app-client1` will receive default route
for `ni-eth0` and this static IP route for `airgap1`.
For `airgap2` we add static IP route sending all traffic via `app-gw`, i.e. `app-client2`
will receive it as the default route.

```text
  "networkInstances": [
    {
      "displayname": "airgap1",
      ...
      "staticRoutes": [
        {
          "destinationNetwork": "10.21.21.0/24",
          "gateway": "172.28.1.2"
        }
      ]
    },
    {
      "displayname": "airgap2",
      ...
      "staticRoutes": [
        {
          "destinationNetwork": "0.0.0.0/0",
          "gateway": "172.28.2.2"
        }
      ]
    },
    ...
  ]
```

Note that `172.28.1.2` and `172.28.2.2` are IP addresses of `app-gw` inside the respective
air-gap network instances (statically assigned).

Run the example with:

```shell
make clean && make build-tests
./eden config add default
./eden config set default --key sdn.disable --value false
./eden setup
./eden start --sdn-network-model $(pwd)/sdn/examples/app-routing/app-gateway/network-model.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/app-routing/app-gateway/device-config.json
```

Once deployed, login to `app-client1`:

```shell
./eden eve ssh
CONSOLE="$(eve list-app-consoles | grep cee082fd-3a43-4599-bbd3-8216ffa8652d | grep CONTAINER | awk '{print $4}')"
eve attach-app-console "$CONSOLE"
```

Inside the application, check IP routing table:

```shell
app-client1$ ip route
default via 10.50.0.1 dev eth0
10.21.21.0/24 via 172.28.1.2 dev eth1
10.50.0.0/24 dev eth0 proto kernel scope link src 10.50.0.2
172.28.1.0/24 dev eth1 proto kernel scope link src 172.28.1.3
```

Try to access `httpserver0`, this should not be routed via `app-gw`:

```shell
app-client1$ curl 10.20.20.70/helloworld
Hello world from HTTP server no. 0
```

Then try to access `httpserver1`, this will be routed via `app-gw`:

```shell
app-client1$ curl 10.21.21.70/helloworld
Hello world from HTTP server no. 1
```

Next, login to `app-client2`:

```shell
./eden eve ssh
CONSOLE="$(eve list-app-consoles | grep 5341bfb9-c828-4f98-807e-e9763d4dc316 | grep CONTAINER | awk '{print $4}')"
eve attach-app-console "$CONSOLE"
```

Check IP routing table:

```shell
app-client2$ ip route
default via 172.28.2.2 dev eth0
172.28.2.0/24 dev eth0 proto kernel scope link src 172.28.2.3
```

Try to access `httpserver1`. This, just like any request from `app-client2`, will be routed
via `app-gw`:

```shell
app-client2$ curl 10.21.21.70/helloworld
Hello world from HTTP server no. 1
```

At the same time, you can login to `app-gw` and observe traffic that passes through while
making requests from client apps:

```shell
./eden eve ssh
CONSOLE="$(eve list-app-consoles | grep 4d88a7c5-64fc-43ee-a58a-f5944bc7872c | grep CONTAINER | awk '{print $4}')"
eve attach-app-console "$CONSOLE"
tcpdump -i any -n
```
