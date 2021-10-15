#! /bin/sh

echo ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -i /root/.ssh/id_rsa root@"$1" -p "$2" grep Ubuntu /etc/issue
ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -i /root/.ssh/id_rsa root@"$1" -p "$2" grep Ubuntu /etc/issue
