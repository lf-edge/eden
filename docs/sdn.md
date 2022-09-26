# Eden-SDN

Eden-SDN is a component of the Eden framework providing programmable networking between
EVE VM (only supported with virtual device) and the controller (adam or zedcloud).

Eden-SDN is running as another VM and every EVE network interface is actually a pipe
(inter-VM interface pair) going into this VM. Inside the Eden-SDN VM there is a management
agent that receives (from eden CLI) a declarative description of the desired networking
(interface count, topology, IP settings, firewall rules, etc.) and configures the (Linux)
network stack and some open-source user-space tools accordingly.

Eden-SDN enables to easily apply and test various device connectivity scenarios.
For example, one may test device onboarding (and other stages of life cycle) with static
IP configuration (as opposed to DHCP-enabled networks, which is what eden configures by default)
or with network proxies terminating TLS traffic, or with a management interface being
assigned a VLAN ID, etc.

Please see additional info [here](../sdn/README.md).
