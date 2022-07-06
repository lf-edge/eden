#!/bin/sh

{
  # TODO: do this from the agent?
  echo "Configuring basic connectivity"
  echo 'nameserver 8.8.8.8' > /etc/resolv.conf
  ip link set dev eth0 up
  dhcpcd -w -j /run/dhcpcd.log eth0
  echo 1 > /proc/sys/net/ipv4/ip_forward

  echo "Starting SSH daemon..."
  /usr/sbin/sshd -h /root/.ssh/id_rsa

  # Manually configure Proxy to debug VW issue...

  # LAN
  ip link set dev eth1 up
  ip link set dev eth2 up
  ip link add name evebr type bridge
  ip link set evebr up
  ip link set eth1 master evebr
  ip link set eth2 master evebr
  ip addr add 192.168.86.1/24 dev evebr

  # DHCP + DNS
  cat << EOF > /run/dnsmasq-evebr.conf
except-interface=lo
bind-interfaces
no-hosts
bogus-priv
stop-dns-rebind
rebind-localhost-ok
neg-ttl=10
dhcp-ttl=600
log-dhcp
log-queries
log-facility=/run/dnsmasq.log
dhcp-leasefile=/run/dnsmasq.leases
server=8.8.8.8@eth0
no-resolv
pid-file=/run/dnsmasq.pid
interface=evebr
listen-address=192.168.86.1
dhcp-option=option:dns-server,192.168.86.1
dhcp-option=option:netmask,255.255.255.0
dhcp-option=option:router,192.168.86.1
dhcp-option=option:classless-static-route,192.168.86.1/32,0.0.0.0,0.0.0.0/0,192.168.86.1,192.168.86.0/24,192.168.86.1
dhcp-range=192.168.86.17,192.168.86.30,60m
EOF
  dnsmasq -C /run/dnsmasq-evebr.conf

  # Proxy
  echo "10.10.10.102 mydomain.adam" >> /etc/hosts
  mkdir -p /var/run/netns
  ip netns add proxyns
  ip netns exec proxyns ip link set dev lo up
  ip link add proxy-out type veth peer name proxy-in
  ip link set proxy-in netns proxyns
  ip -n proxyns link set proxy-in up
  ip -n proxyns addr add 192.168.120.1/30 dev proxy-in
  ip link set proxy-out up
  ip addr add 192.168.120.2/30 dev proxy-out
  ip netns exec proxyns ip route add default via 192.168.120.2 dev proxy-in
  ip netns exec proxyns goproxy -v &

  # Firewall & NAT
  iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
  iptables -t filter -A FORWARD -s 192.168.86.16/28 -d 192.168.120.1 -j ACCEPT
  iptables -t filter -A FORWARD -s 192.168.86.16/28 -j DROP

  echo "Starting Eden SDN mgmt agent..."
  while true; do
    sdn-agent -debug
    echo "Restarting Eden SDN mgmt Agent!!!"
  done


 #TODO: get rid of lastresort
 #TODO: find out why we cannot load proxy cert via override
 #TODO: find out why lastresort is not disabled (probably due to: https://github.com/lf-edge/eve/pull/2620/files)
} > /run/sdn.log 2>&1