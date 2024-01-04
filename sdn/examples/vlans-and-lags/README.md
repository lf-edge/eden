# SDN Example with VLANs and LAGs

VLANs enable the segmentation of a physical network into multiple logical networks,
allowing for better traffic control, security, and resource optimization.
On EVE, the use of VLANs helps isolate the management traffic from application traffic
or even to split applications and their traffic into different logical networks.
This allows the external networks to give preferential treatment and apply different
policies as per their requirements.

On the other hand, Link Aggregation Groups, commonly known as bonds or LAGs, aggregate
multiple physical links between network devices into a single logical link.
This not only increases the available bandwidth but also provides redundancy
and load balancing. LAGs ensure a more resilient and reliable network infrastructure
by distributing traffic across multiple links, thereby avoiding bottlenecks and improving
overall network performance. Moreover, LAGs enhance fault tolerance as they continue
to operate even if some individual links fail.

EVE supports:

- VLAN filtering for switch network instances
- VLAN sub-interfaces used as uplinks for management traffic or for local network instances
- LAG (bond) interfaces aggregating multiple physical interfaces and used as uplinks
- VLAN sub-interfaces over LAGs used as uplinks

Here we combine all these cases into one example:

- device has 3 physical interfaces all connected to the same switch
- `eth0` and `eth1` are aggregated by Round-Robin LAG
- Both `eth0+eth1` and `eth2` are trunks for VLANs 10, 20, 30
- VLAN 10 is used for EVE management (controller is reachable)
- VLAN 20 is used by `app1`
- VLAN 30 is used by 2 apps: `app2` and `app3`
- These VLANs are isolated from each other (router does not have routes from one VLAN to another)
- `app1` is connected to VLAN 20 indirectly through local network instance which uses
  VLAN sub-interface as uplink
- `app2` is connected to VLAN 30 directly through switch network instance with VLAN
  sub-interface as uplink
- `app3` is also connected to VLAN 30 directly, but in this case through switch network
  instance with `eth2` as uplink (trunk) and with traffic filtered down to VLAN 30
  by switch NI bridge
- dedicated HTTP "hello-world" servers are deployed for VLAN 20 and VLAN 30,
  accessible exclusively through their respective VLAN networks
  (HTTP servers are deployed in different segments but router is configured accordingly)

Network topology diagram:

```text
            +-----+  +----------------+
            | EVE |--| VLAN 10 (mgmt) |-----
            +-----+  +----------------+    |          +------+      +---------+
                                           |      ----| eth0 |......| p0 --+  |
+------+  +--------------+  +---------+  +-----+  |   +------+      |      |  |
| app1 |--| NI1 (local)  |--| VLAN 20 |--| LAG |--|                 |    LAG  |
+------+  +--------------+  +---------+  +-----+  |   +------+      |      |  |  VLAN20  +---------------+
                                           |      ----| eth1 |......| p1 --+  |..........| httpserver-20 |
+------+  +--------------+  +---------+    |          +------+      |         |          +---------------+
| app2 |--| NI2 (switch) |--| VLAN 30 |-----                        |         |
+------+  +--------------+  +---------+                             | SWITCH  |  VLAN30  +---------------+
                                                                    |         |..........| httpserver-30 |
+------+                    +--------------+          +------+      |         |          +---------------+
| app3 |--<access VLAN 30>--| NI3 (switch) |----------| eth2 |......| p2      |
+------+                    +--------------+          +------+      |         |
                                                                    +---------+
```

Deploy example with:

```shell
make clean && make build-tests
./eden config add default
./eden config set default --key sdn.disable --value false
./eden setup --eve-bootstrap-file $(pwd)/sdn/examples/vlans-and-lags/device-config.json
./eden start --sdn-network-model $(pwd)/sdn/examples/vlans-and-lags/network-model.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/vlans-and-lags/device-config.json
```

Note that VLAN IP subnets are `172.22.<VLAN-ID>.0/24`. Apps `app2` and `app3` will have IPs
from the subnet `172.22.30.0/24`. `app1` is connected to VLAN 20 indirectly and will use IP
from the local NI subnet (`10.50.0.0/24`).

Check management interface IP:

```shell
./eden eve ssh
ifconfig  vlan10
vlan10    Link encap:Ethernet  HWaddr 02:FE:22:1A:87:00
          inet addr:172.22.10.13  Bcast:172.22.10.255  Mask:255.255.255.0
```

Check app IPs:

```shell
./eden pod ps
NAME	IMAGE				UUID					INTERNAL	EXTERNAL	MEMORY		STATE(ADAM)	LAST_STATE(EVE)
app1	lfedge/eden-eclient:8a279cd	cee082fd-3a43-4599-bbd3-8216ffa8652d	10.50.0.2	-		177 MB/709 MB	IN_CONFIG	RUNNING
app2	lfedge/eden-eclient:8a279cd	45ff198d-b295-4ff2-bf69-76977af809fd	172.22.30.11	-		154 MB/732 MB	IN_CONFIG	RUNNING
app3	lfedge/eden-eclient:8a279cd	0c569673-988d-4d32-874c-2b09de12e0fc	172.22.30.12	-		155 MB/731 MB	IN_CONFIG	RUNNING
```

Check that `app1` can access HTTP server deployed for VLAN 20 (`10.20.20.70`),
but not HTTP server deployed for VLAN 30 (`10.30.30.70`):

```shell
./eden eve ssh
CONSOLE="$(eve list-app-consoles | grep cee082fd-3a43-4599-bbd3-8216ffa8652d | grep CONTAINER | awk '{print $4}')"
eve attach-app-console "$CONSOLE"

app1$ curl 10.20.20.70/helloworld
Hello world from HTTP server for VLAN 20
app1$ curl 10.30.30.70/helloworld
curl: (7) Failed to connect to 10.30.30.70 port 80 after 1 ms: Couldn't connect to server
```

Check that `app2` and `app3` can access each other:

```shell
./eden eve ssh
CONSOLE="$(eve list-app-consoles | grep 45ff198d-b295-4ff2-bf69-76977af809fd | grep CONTAINER | awk '{print $4}')"
eve attach-app-console "$CONSOLE"

app2$ ping -c 3 172.22.30.12
PING 172.22.30.12 (172.22.30.12): 56 data bytes
64 bytes from 172.22.30.12: seq=0 ttl=64 time=45.818 ms
64 bytes from 172.22.30.12: seq=1 ttl=64 time=5.318 ms
64 bytes from 172.22.30.12: seq=2 ttl=64 time=4.895 ms

--- 172.22.30.12 ping statistics ---
3 packets transmitted, 3 packets received, 0% packet loss
round-trip min/avg/max = 4.895/18.677/45.818 ms
```

Check that `app2` can access HTTP server deployed for VLAN 30 (`10.30.30.70`),
but not HTTP server deployed for VLAN 20 (`10.20.20.70`):

```shell
./eden eve ssh
CONSOLE="$(eve list-app-consoles | grep 45ff198d-b295-4ff2-bf69-76977af809fd | grep CONTAINER | awk '{print $4}')"
eve attach-app-console "$CONSOLE"

app2$ curl 10.20.20.70/helloworld
curl: (7) Failed to connect to 10.20.20.70 port 80 after 3 ms: Couldn't connect to server
app2$ curl 10.30.30.70/helloworld
Hello world from HTTP server for VLAN 30
```

Check that `app3` can access HTTP server deployed for VLAN 30 (`10.30.30.70`),
but not HTTP server deployed for VLAN 20 (`10.20.20.70`):

```shell
./eden eve ssh
CONSOLE="$(eve list-app-consoles | grep 0c569673-988d-4d32-874c-2b09de12e0fc | grep CONTAINER | awk '{print $4}')"
eve attach-app-console "$CONSOLE"

app3$ curl 10.20.20.70/helloworld
curl: (7) Failed to connect to 10.20.20.70 port 80 after 3 ms: Couldn't connect to server
app3$ curl 10.30.30.70/helloworld
Hello world from HTTP server for VLAN 30
```
