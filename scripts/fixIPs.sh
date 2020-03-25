#!/bin/bash
usage() {
  echo "Usage: $0 Makefile"
  exit
}

if [ "$#" -ne 1 ]; then
  usage
fi

Makefile=$1

subnet1_prefix=""
subnet2_prefix=""
for ((i = 0; i <= 254; i++)); do
  ifconfig | grep -F -q "192.168.$i."
  if [[ $? -ne 0 ]]; then
    if [ -z $subnet1_prefix ]; then
      subnet1_prefix="192\.168\.$i"
      continue
    fi
    if [ -z $subnet2_prefix ]; then
      subnet2_prefix="192\.168\.$i"
      continue
    fi
    break
  fi
done

sed -i "s/eth0,net=192\.168\.1\.0\/24,dhcpstart=192\.168\.1\.10/eth0,net=$subnet1_prefix\.0\/24,dhcpstart=$subnet1_prefix\.10/g" $Makefile
sed -i "s/eth1,net=192\.168\.2\.0\/24,dhcpstart=192\.168\.2\.10/eth1,net=$subnet2_prefix\.0\/24,dhcpstart=$subnet2_prefix\.10/g" $Makefile
