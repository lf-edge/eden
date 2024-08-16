# SDN Example with Switch Network Instance using multiple ports

Switch network instance with multiple ports allows to:

* bridge multiple switches and add redundant links. STP is used to avoid bridge loops.
* connect end-devices into the same L2 segment as applications running on the edge node.

User is able to use shared-label to select multiple network ports for a Switch Network Instance.
EVE automatically runs the Spanning Tree Protocol (STP) for bridge with multiple ports to avoid
bridge loops and broadcast storms. User is able to enable BPDU guard on ports that are supposed
to connect end-devices and therefore are not expected to participate in the STP algorithm.

With VLAN-enabled Switch NI, the user is able to use a physical network port attached
to Switch NI as a VLAN access port (by default all switch NI ports are configured as trunks).

## Example 1: Switch NI with redundant links

In this example, we connect application into a single switch instance with 2 ports
connected into the same L2 segment. Spanning Tree Protocol will pick one of these redundant
links for forwarding while it will block the other one. If the active link goes down,
STP will re-converge. The blocked port will transition from the Blocking state to
the Forwarding state, effectively allowing traffic to flow through the previously
redundant link.

Here is a diagram depicting the network topology:

```text
                          +-----------------+
                          | eth0 (EVE mgmt) |----------------------------
                          |     (DHCP)      |                           |
                          +-----------------+                           |
                                                                        |
+-----+   +----------+    +-------------------+                         |
| app |-->| Switch NI |-->| eth1 (app-shared) |      +--------+      +--------+   +------------+
+-----+   +----------+    | (No IP, L2-only)  |------| switch |------| router |---| httpserver |
              |           +-------------------+      +--------+      +--------+   +------------+
              |                                           |
              |           +-------------------+           |
              ----------->| eth2 (app-shared) |------------
                          | (No IP, L2-only)  |
                          +-------------------+
```

Run the example with:

```shell
make clean && make build-tests
./eden config add default
./eden config set default --key sdn.disable --value false
./eden setup --eve-bootstrap-file $(pwd)/sdn/examples/switch-ni-multiple-ports/redundant-links/device-config.json
./eden start --sdn-network-model $(pwd)/sdn/examples/switch-ni-multiple-ports/redundant-links/network-model.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/switch-ni-multiple-ports/redundant-links/device-config.json
```

Once deployed, check that the bridge is created, runs STP and contains both eth1 and eth2:

```shell
./eden eve ssh
eve$ brctl showstp bn1
bn1
 bridge id           8000.00163e060001
 designated root     8000.00163e060001
 root port              0               path cost              0
 max age               20.00            bridge max age        20.00
 hello time             2.00            bridge hello time      2.00
 forward delay         15.00            bridge forward delay  15.00
 ageing time          300.00
 hello timer            1.62            tcn timer              0.00
 topology change timer  0.00            gc timer              41.52
 flags


nbu1x1 (3)
 port id            8003                state          forwarding
 designated root    8000.00163e060001   path cost          100
 designated bridge  8000.00163e060001   message age timer    0.00
 designated port    8003                forwrd delay timer   0.00
 designated cost       0                hold timer           0.62
 flags

eth1 (2)
 port id            8002                state          forwarding
 designated root    8000.00163e060001   path cost          100
 designated bridge  8000.00163e060001   message age timer    0.00
 designated port    8002                forward delay timer  0.00
 designated cost       0                hold timer           0.62
 flags

eth2 (1)
 port id            8001                 state          forwarding
 designated root    8000.00163e060001    path cost          100
 designated bridge  8000.00163e060001    message age timer    0.00
 designated port    8001                 forward delay timer  0.00
 designated cost       0                 hold timer           0.62
 flags

```

In this case, `bridge id` and `designated root` show the same values, meaning
that this is a STP root bridge and all ports are thus in the Forwarding state.
To avoid bridge loops, the opposite bridge, running inside Eden-SDN must therefore
block one of the ports:

```shell
./eden sdn ssh
sdn$ brctl showstp br-bridge1
br-bridge1
 bridge id           8000.02fdb51b8701
 designated root     8000.00163e060001
 root port              2                path cost              4
 max age               20.00             bridge max age        20.00
 hello time             2.00             bridge hello time      2.00
 forward delay         15.00             bridge forward delay  15.00
 ageing time          300.00
 hello timer            0.00             tcn timer              0.00
 topology change timer  0.00             gc timer              54.73
 flags


net-br-out-yCqC (3)
 port id            8003                 state            forwarding
 designated root    8000.00163e060001    path cost              2
 designated bridge  8000.02fdb51b8701    message age timer      0.00
 designated port    8003                 forward delay timer    0.00
 designated cost       4                 hold timer             0.70
 flags

eth1 (1)
 port id            8001                 state            blocking
 designated root    8000.00163e060001    path cost            4
 designated bridge  8000.00163e060001    message age timer   19.88
 designated port    8002                 forward delay timer  0.00
 designated cost       0                 hold timer           0.00
 flags

eth2 (2)
 port id            8002                 state            forwarding
 designated root    8000.00163e060001    path cost              4
 designated bridge  8000.00163e060001    message age timer     19.88
 designated port    8001                 forward delay timer    0.00
 designated cost       0                 hold timer             0.00
 flags
```

Notice that `eth1` is in the blocking state.

Login to the application and try to access something in the Internet (e.g. 8.8.8.8)
and the HTTP server:

```shell
./eden eve ssh
eve$ eve attach-app-console 4d88a7c5-64fc-43ee-a58a-f5944bc7872c.1.1/cons
app$ ping -c 3 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=252 time=17.6 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=252 time=19.2 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=252 time=18.2 ms

--- 8.8.8.8 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2004ms
rtt min/avg/max/mdev = 17.563/18.318/19.150/0.650 ms

app$ curl httpserver.sdn/helloworld
Hello world from HTTP server
```

Next, simulate forwarding port (`eth2` in this case) losing the connectivity
by changing the network model:

```shell
./eden sdn net-model get > net-model
jq '(.ports[] | select(.logicalLabel == "eveport2").adminUP) = false' net-model > net-model-eth2-down
./eden sdn net-model apply net-model-eth2-down
```

Then check the STP link states of the non-root bridge (in this case inside SDN).
The previously Blocked port will transition through Listening, Learning and eventually
will reach the Forwarding state:

```shell
./eden sdn ssh
sdn$ brctl showstp br-bridge1
...
 flags           TOPOLOGY_CHANGE
...
eth1 (1)
 port id            8001                 state           forwarding
 designated root    8000.00163e060001    path cost             4
 designated bridge  8000.00163e060001    message age timer    19.42
 designated port    8002                 forward delay timer   0.00
 designated cost       0                 hold timer            0.00
 flags
```

You may retry the connectivity tests from the application to confirm a successful failover.

## Example 2: Switch NI with VLAN access ports

In this example, we create switch NI instance for VLANs 100 and 200. One application
is deployed inside each VLAN. Eden-SDN provides DHCP and IP gateway services for both VLANs
via trunk port.
Additionally, we add one physical access port for each VLAN and used them to connect
external HTTP servers directly into the segments.

Here is a diagram depicting the network topology:

```text
                                                 +-----------------+
                                                 | eth0 (EVE mgmt) |---------------------
                                                 |     (DHCP)      |                    |
                                                 +-----------------+                    |
                      +-----------+                                                     |
                      |           |                                                     |
                      |           |              +-------------------+      +------------------------+
                      |           |---<trunk>----| eth1 (app-shared) |------|         router         |
                      |           |              | (No IP, L2-only)  |      | (DHCP server per VLAN) |
                      | Switch NI |              +-------------------+      +------------------------+
                      |           |
+------+              |           |              +-------------------+      +-------------+
| app1 |--<VLAN 100>--|           |--<VLAN 100>--| eth2 (app-shared) |------| httpserver1 |
+------+              |           |              | (No IP, L2-only)  |      +-------------+
                      |           |              +-------------------+
+------+              |           |
| app2 |--<VLAN 200>--|           |              +-------------------+      +-------------+
+------+              |           |--<VLAN 200>--| eth3 (app-shared) |------| httpserver2 |
                      |           |              | (No IP, L2-only)  |      +-------------+
                      +-----------+              +-------------------+
```

```shell
make clean && make build-tests
./eden config add default
./eden config set default --key sdn.disable --value false
./eden setup --eve-bootstrap-file $(pwd)/sdn/examples/switch-ni-multiple-ports/access-ports/device-config.json
./eden start --sdn-network-model $(pwd)/sdn/examples/switch-ni-multiple-ports/access-ports/network-model.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/switch-ni-multiple-ports/access-ports/device-config.json
```

Once deployed, check that the bridge is created and has access VLANs set correctly for eth2, eth3
and applications VIFs:

```shell
./eden eve ssh
eve$ eve enter pillar
pillar$ bridge vlan show
port              vlan-id
keth0             1 PVID Egress Untagged
eth1              1 PVID Egress Untagged
                  100
                  200
eth2              1 Egress Untagged
                  100 PVID Egress Untagged
eth3              1 Egress Untagged
                  200 PVID Egress Untagged
eth0              1 PVID Egress Untagged
bn1               1 PVID Egress Untagged
nbu1x1            1 Egress Untagged
                  100 PVID Egress Untagged
nbu1x2            1 Egress Untagged
                  200 PVID Egress Untagged
```

Next, check that the bridge is VLAN-aware:

```shell
./eden eve ssh
eve$ cat /sys/class/net/bn1/bridge/vlan_filtering
1
```

Next, check that all port connecting endpoints (apps and HTTP servers) have BPDU guard enabled:

```shell
./eden eve ssh
eve$ cat /sys/class/net/bn1/brif/eth1/bpdu_guard
0
eve$ cat /sys/class/net/bn1/brif/eth2/bpdu_guard
1
eve$ cat /sys/class/net/bn1/brif/eth3/bpdu_guard
1
eve$ cat /sys/class/net/bn1/brif/nbu1x1/bpdu_guard
1
eve$ cat /sys/class/net/bn1/brif/nbu1x2/bpdu_guard
1
```

Login to the application `app1`, check that the IP address is from the range `10.203.10.0/24`
and try to access HTTP server in the same VLAN and something in the Internet (e.g. 8.8.8.8):

```shell
./eden eve ssh
eve$ eve attach-app-console cee082fd-3a43-4599-bbd3-8216ffa8652d.1.1/cons
app1$ ifconfig eth0
eth0      Link encap:Ethernet  HWaddr 02:16:3E:2B:F1:BE
          inet addr:10.203.10.129  Bcast:10.203.10.255  Mask:255.255.255.0
          UP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1
          RX packets:8 errors:0 dropped:0 overruns:0 frame:0
          TX packets:31 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000
          RX bytes:1328 (1.2 KiB)  TX bytes:5759 (5.6 KiB)

app1$ curl httpserver100.sdn/helloworld
Hello world from HTTP server for VLAN 100

# Cannot access HTTP server in another VLAN:
app1$ curl --max-time 5 httpserver200.sdn/helloworld
curl: (28) Connection timed out after 5002 milliseconds

app1$ ping -c 3 8.8.8.8
PING 8.8.8.8 (8.8.8.8): 56 data bytes
64 bytes from 8.8.8.8: seq=0 ttl=253 time=15.846 ms
64 bytes from 8.8.8.8: seq=1 ttl=253 time=17.075 ms
64 bytes from 8.8.8.8: seq=2 ttl=253 time=15.062 ms
--- 8.8.8.8 ping statistics ---
3 packets transmitted, 3 packets received, 0% packet loss
round-trip min/avg/max = 15.062/15.994/17.075 ms
```

Login to the application `app2`, check that the IP address is from the range `10.203.20.0/24`
and try to access HTTP server in the same VLAN and something in the Internet (e.g. 8.8.8.8):

```shell
./eden eve ssh
eve$ eve attach-app-console 5341bfb9-c828-4f98-807e-e9763d4dc316.1.2/cons
app2$ ifconfig eth0
eth0      Link encap:Ethernet  HWaddr 02:16:3E:8A:6C:7D
          inet addr:10.203.20.133  Bcast:10.203.20.255  Mask:255.255.255.0
          UP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1
          RX packets:8 errors:0 dropped:0 overruns:0 frame:0
          TX packets:31 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000
          RX bytes:1328 (1.2 KiB)  TX bytes:5759 (5.6 KiB)

app2$ curl httpserver200.sdn/helloworld
Hello world from HTTP server for VLAN 200

# Cannot access HTTP server in another VLAN:
app2$ curl --max-time 5 httpserver100.sdn/helloworld
curl: (28) Connection timed out after 5002 milliseconds

app2$ ping -c 3 8.8.8.8
PING 8.8.8.8 (8.8.8.8): 56 data bytes
64 bytes from 8.8.8.8: seq=0 ttl=253 time=16.407 ms
64 bytes from 8.8.8.8: seq=1 ttl=253 time=18.493 ms
64 bytes from 8.8.8.8: seq=2 ttl=253 time=14.792 ms
--- 8.8.8.8 ping statistics ---
3 packets transmitted, 3 packets received, 0% packet loss
round-trip min/avg/max = 14.792/16.564/18.493 ms
```
