#! /bin/sh

EVE_IP=$(curl -s "http://169.254.169.254/eve/v1/network.json" | jq -r '."external-ipv4"')
echo "$EVE_IP"
