# SDN Example with Local Network Instance using multiple ports

When application is connected to multiple IP networks over multiple network ports,
IP routing must be utilized to properly route network flows so that they reach their
intended destination.

EVE offers two options for the application traffic routing:

1. Create a separate network instance for every port needed by app and use DHCP-based propagation
   of IP routes into applications. This option is shown in the [app-routing example](../app-routing)
   It is more difficult to configure but gives the application full control over the IP routing
2. Create a single local network instance with all those ports assigned and use the IP routing
   capabilities of the network instance. From the application perspective this is much simpler
   because it will be connected to only a single NI and thus have only one interface. On the app
   side a single default route for the single network interface is all that is needed.
   The complexity of routing it taken care of by EVE. This option is shown in this example.

Local Network Instance with multiple ports will have link-local and connected routes
from all the ports present in its routing table. Additionally, user may configure static
IP routes which will be added into the routing table. A static route may reference
a particular gateway IP as the next hop, or a logical label of a port to use as the output
device, or use a shared label to match a subset of NI ports if there are multiple
possible paths to reach the routed destination network.

For every multi-path route with shared port label, EVE will perform periodic probing
of all matched network ports to determine the connectivity status and select the best
port to use for the route. Note that every multi-path route will have at most one output
port selected and configured at any time - load-balancing is currently not supported.

The probing method can be customized by the user as part of the route configuration.
If enabled, EVE will check the reachability of the port's next hop (the gateway IP)
every 15 seconds using ICMP ping. The upside of using this probe is fairly quick fail-over
when the currently used port for a given multi-path route looses connectivity.
The downside is that it may generate quite a lot of traffic over time. User may limit
the use of this probe to only ports with low cost or disable this probe altogether.
Additionally, every 2.5 minutes, EVE will run user-defined probe if configured.
This can be either an ICMP ping towards a given IP or hostname, or a TCP handshake against
the given IP/hostname and port.

A connectivity probe must consecutively success/fail few times in a row to determine
the connectivity as being up/down. EVE will then consider changing the currently used port
for a given route. The requirement for multiple consecutive test passes/failures prevents
from port flapping, i.e. re-routing too often.

Additionally to connectivity status, there are some other metrics that can affect the port
selection decision. For example, user may enable lower-cost-preference for a given multi-path
route. In that case, with multiple connected ports, EVE will select the lowest-cost port.
Similarly, route that uses multiple wwan ports, can be configured to give preferential
selection to cellular modem with better network signal strength.

## Example scenario

In this example, we connect application into a single network instance with 4 ports
attached (using shared label `all`).
Ports `eth0` and `eth2` get their IP configs from DHCP, while `eth1` and `eth3` are assigned
IPs statically. `eth0` and `eth3` have Internet access and are used for management.
Additionally, `eth0`, `eth2` and `eth3` have access to an HTTP server deployed inside
the Eden SDN VM. Shared port label `httpserver` is created to mark these ports and use
them for a multi-path route with the subnet of the HTTP server as destination (`10.88.88.0/24`).
This route is configured to use a TCP handshake with the HTTP server to probe connectivity
of those ports and select the lowest-cost port with a working connectivity.

Moreover, a default multi-path route is created with a shared label `internet` assigned
to `eth0` and `eth3`. This route is configured to only perform ICMP ping of the gateway
to determine port connectivity status.

Lastly, port forwarding is configured to enable accessing the app from outside.
However, this is limited to `eth0`, `eth1` and `eth2` using a shared label `portfwd`
(purely for example purposes).

Here is a diagram depicting the network topology:

```text
                         +-------------------+
              ---------->|    eth0 (mgmt)    |---------------
              |          |  (DHCP, portfwd)  |              |
              |          +-------------------+              |
              |                                             |
+-----+   +----------+   +-------------------+              |
| app |-->| Local NI |-->| eth1 (app-shared) |              |
+-----+   +----------+   | (static, portfwd) |              |
              |          +-------------------+              |
              |                                             |
              |          +-------------------+          +--------+  +------------+
              ---------->| eth2 (app-shared) |----------| router |--| httpserver |
              |          |  (DHCP, portfwd)  |          +--------+  +------------+
              |          +-------------------+               |
              |                                              |
              |          +-------------------+               |
              ---------->|    eth3 (mgmt)    |----------------
                         | (static IP conf)  |
                         +-------------------+
```

Run the example with:

```shell
make clean && make build-tests
./eden config add default
./eden config set default --key sdn.disable --value false
./eden setup --eve-bootstrap-file $(pwd)/sdn/examples/local-ni-multiple-ports/device-config.json
./eden start --sdn-network-model $(pwd)/sdn/examples/local-ni-multiple-ports/network-model.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/local-ni-multiple-ports/device-config.json
```

Once deployed, check the routing table of the network instance:

```shell
./eden eve ssh
eve$ ip route show table 801
default via 172.22.10.1 dev eth0 proto static
unreachable default proto static metric 4294967295
10.40.40.0/24 dev eth3 proto static scope link src 10.40.40.30 metric 1009
10.50.0.0/24 dev bn1 proto static scope link src 10.50.0.1
10.88.88.0/24 via 172.22.10.1 dev eth0 proto static
172.22.10.0/24 dev eth0 proto static scope link src 172.22.10.13 metric 1011
172.28.20.0/24 dev eth1 proto static scope link src 172.28.20.10 metric 1008
192.168.30.0/24 dev eth2 proto static scope link src 192.168.30.20 metric 1010
```

Notice that `eth0` is selected for both the default route and the HTTP server route.

Login to the application and try to access something in the Internet (e.g. 8.8.8.8)
and the HTTP server:

```shell
eve$ eve attach-app-console 3599588a-17d3-4d02-aae1-bcefe3706cfd.1.1/cons
app$ ping -c 3 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=252 time=17.6 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=252 time=19.2 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=252 time=18.2 ms

--- 8.8.8.8 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2004ms
rtt min/avg/max/mdev = 17.563/18.318/19.150/0.650 ms

app$ curl httpserver0.sdn/helloworld
Hello world from HTTP server
```

Check the external IP addresses that application will use (depending on the destination):

```shell
app$ curl http://169.254.169.254/eve/v1/network.json 2>/dev/null | jq
{
  "app-instance-uuid": "3599588a-17d3-4d02-aae1-bcefe3706cfd",
  "caller-ip": "10.50.0.2:48588",
  "device-name": "6845629e-500d-4b00-be66-801e75ba65b5",
  "device-uuid": "6845629e-500d-4b00-be66-801e75ba65b5",
  "enterprise-id": "",
  "enterprise-name": "",
  "external-ipv4": "172.22.10.13,172.28.20.10,192.168.30.20,10.40.40.30",
  "hostname": "3599588a-17d3-4d02-aae1-bcefe3706cfd",
  "project-name": "",
  "project-uuid": "00000000-0000-0000-0000-000000000000"
}
app$ curl http://169.254.169.254/eve/v1/external_ipv4 2>/dev/null
172.22.10.13
172.28.20.10
192.168.30.20
10.40.40.30
```

Try to ssh into the application over every port and confirm that only `eth3`
has port forwarding forbidden:

```shell
./eden sdn fwd eth0 2223 -- ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -o PasswordAuthentication=no -i ./dist/tests/eclient/image/cert/id_rsa root@FWD_IP -p FWD_PORT
...
Welcome to Alpine!
...
./eden sdn fwd eth1 2223 -- ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -o PasswordAuthentication=no -i ./dist/tests/eclient/image/cert/id_rsa root@FWD_IP -p FWD_PORT
...
Welcome to Alpine!
...
./eden sdn fwd eth2 2223 -- ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -o PasswordAuthentication=no -i ./dist/tests/eclient/image/cert/id_rsa root@FWD_IP -p FWD_PORT
...
Welcome to Alpine!
...
./eden sdn fwd eth3 2223 -- ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -o PasswordAuthentication=no -i ./dist/tests/eclient/image/cert/id_rsa root@FWD_IP -p FWD_PORT
...
Connection timed out during banner exchange
...
```

Next, simulate `eth0` losing the connectivity by changing the network model:

```shell
./eden sdn net-model get > net-model
jq '(.ports[] | select(.logicalLabel == "eveport0").adminUP) = false' net-model > net-model-eth0-down
./eden sdn net-model apply net-model-eth0-down
```

Eventually, default route is re-routed to use `eth3` while the HTTP server route
will use `eth2` (has lower cost than `eth3`):

```shell
# ssh over eth0 is not going to work, use console access:
./eden eve console
eve$ ip route show table 801
default via 10.40.40.1 dev eth3 proto static
unreachable default proto static metric 4294967295
10.40.40.0/24 dev eth3 proto static scope link src 10.40.40.30 metric 1009
10.50.0.0/24 dev bn1 proto static scope link src 10.50.0.1
10.88.88.0/24 via 192.168.30.1 dev eth2 proto static
172.22.10.0/24 dev eth0 proto static scope link src 172.22.10.13 metric 1011
172.28.20.0/24 dev eth1 proto static scope link src 172.28.20.10 metric 1008
192.168.30.0/24 dev eth2 proto static scope link src 192.168.30.20 metric 1010
```
