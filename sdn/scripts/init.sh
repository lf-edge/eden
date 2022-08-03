#!/bin/sh

{
  echo "Starting SSH daemon..."
  /usr/sbin/sshd -h /root/.ssh/id_rsa

  echo "Starting Eden SDN mgmt agent..."
  while true; do
    sdnagent -debug
    echo "Restarting Eden SDN mgmt Agent!!!"
  done

} > /run/sdn.log 2>&1
