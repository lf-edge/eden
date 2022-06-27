#!/bin/sh

{
  # TODO: do this from the agent?
  echo "Configuring basic connectivity"
  echo 'nameserver 8.8.8.8' > /etc/resolv.conf
  ip link set dev eth0 up
  dhcpcd -w -j /run/dhcpcd.log eth0

  echo "Starting SSH daemon..."
  /usr/sbin/sshd -h /root/.ssh/id_rsa

  echo "Starting Eden SDN mgmt agent..."
  while true; do
    sdn-agent
    echo "Restarting Eden SDN mgmt Agent!!!"
  done

} > /run/sdn.log 2>&1