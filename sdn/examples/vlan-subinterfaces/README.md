# SDN Example with VLAN sub-interfaces

VLANs enable the segmentation of a physical network into multiple logical networks,
allowing for better traffic control, security, and resource optimization.
On EVE, the use of VLANs helps isolate the management traffic from application traffic
or even to split applications and their traffic into different logical networks.
This allows the external networks to give preferential treatment and apply different
policies as per their requirements.

VLAN configurations supported by EVE:

1. VLAN filtering for switch network instances
2. VLAN sub-interfaces over physical NICs used for management traffic or for Local NIs
3. VLAN sub-interfaces over LAGs used for management traffic or for Local NIs

In this example, we focus on the second use-case, where VLANs are used to separate management
traffic from the application traffic routed via Local network instances.

Network topology diagram:

```text
            +-----+  +----------------+
            | EVE |--| VLAN 10 (mgmt) |-----
            +-----+  +----------------+    |
                                           |
+------+  +--------------+  +---------+  +------+
| app1 |--| NI1 (local)  |--| VLAN 20 |--| eth0 |
+------+  +--------------+  +---------+  +------+
                                           |
+------+  +--------------+                 |
| app2 |--| NI2 (local)  |---<untagged>-----
+------+  +--------------+
```

Deploy example with:

```shell
make clean && make build-tests
./eden config add default
./eden config set default --key sdn.disable --value false
./eden setup --eve-bootstrap-file $(pwd)/sdn/examples/vlan-subinterfaces/device-config.json
./eden start --sdn-network-model $(pwd)/sdn/examples/vlan-subinterfaces/network-model.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/vlan-subinterfaces/device-config.json
```

Note that VLAN IP subnets are `172.22.<VLAN-ID>.0/24`. EVE will therefore use IP address from
the subnet `172.22.10.0/24` to access the controller. Network traffic from `app1` will be NATed
to an IP address from `172.22.20.0/24` before it leaves the edge node.
`app2` will be using the underlying `eth0` interface instead of a VLAN sub-interface to access
the untagged portion of the network with subnet `192.168.77.0/24`.

Once deployed, check DHCP-assigned IPs:

```shell
./eden eve ssh
$ ifconfig vlan10
vlan10    Link encap:Ethernet  HWaddr 02:FE:22:1A:87:00
          inet addr:172.22.10.13  Bcast:172.22.10.255  Mask:255.255.255.0
          ...

$ ifconfig vlan20
vlan20    Link encap:Ethernet  HWaddr 02:FE:22:1A:87:00
          inet addr:172.22.20.13  Bcast:172.22.20.255  Mask:255.255.255.0

$ ifconfig eth0
eth0      Link encap:Ethernet  HWaddr 02:FE:22:1A:87:00
          inet addr:192.168.77.13  Bcast:192.168.77.255  Mask:255.255.255.0
          ...
```

Check that `app1` can access HTTP server deployed for VLAN 20 (`httpserver-20.sdn`),
but not HTTP server deployed for VLAN 10 (`httpserver-10.sdn`) or for the untagged
network (`httpserver-untagged.sdn`):

```shell
./eden eve ssh
CONSOLE="$(eve list-app-consoles | grep cee082fd-3a43-4599-bbd3-8216ffa8652d | grep CONTAINER | awk '{print $4}')"
eve attach-app-console "$CONSOLE"

app1$ curl httpserver-20.sdn/helloworld
Hello world from HTTP server for VLAN 20
app1$ curl httpserver-10.sdn/helloworld
curl: (7) Failed to connect to httpserver-10.sdn port 80 after 44 ms: Couldn't connect to server
app1$ curl httpserver-untagged.sdn/helloworld
curl: (7) Failed to connect to httpserver-untagged.sdn port 80 after 48 ms: Couldn't connect to server
```

Check that `app2` can access HTTP server deployed for the untagged network (`httpserver-untagged.sdn`),
but not HTTP server deployed for VLAN 10 (`httpserver-10.sdn`) or for VLAN 20 (`httpserver-20.sdn`):

```shell
./eden eve ssh
CONSOLE="$(eve list-app-consoles | grep 45ff198d-b295-4ff2-bf69-76977af809fd | grep CONTAINER | awk '{print $4}')"
eve attach-app-console "$CONSOLE"

app2$ curl httpserver-untagged.sdn/helloworld
Hello world from HTTP server for untagged network
app1$ curl httpserver-10.sdn/helloworld
curl: (7) Failed to connect to httpserver-10.sdn port 80 after 47 ms: Couldn't connect to server
app1$ curl httpserver-20.sdn/helloworld
curl: (7) Failed to connect to httpserver-20.sdn port 80 after 47 ms: Couldn't connect to server
```