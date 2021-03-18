#! /bin/sh

EVE_IP=$(curl -s "http://169.254.169.254/eve/v1/network.json" | jq -r '."external-ipv4"')
echo ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -i /root/.ssh/id_rsa root@"$EVE_IP" -p "$1" grep Ubuntu /etc/issue
ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -i /root/.ssh/id_rsa root@"$EVE_IP" -p "$1" grep Ubuntu /etc/issue
