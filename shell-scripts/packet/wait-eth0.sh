#!/bin/bash
device_id=$1
repeat_count=$2

fail() { echo "ERROR: packet: $*" 1>&2; exit 1; }

help_text=$(cat << __EOT__
Usage: wait-eth0.sh <packet server id> <repeat count>

Attempts to get the ip of first internet adapter, with the specified number of attempts.
There is a 10 second break between attempts to get ip.

Returns the ip of th packet server to the stdout or 0.0.0.0 on fail.
__EOT__
)

function packet-cli() {
   if [ -e "$HOME"/go/bin/packet-cli ]; then
     "$HOME"/go/bin/packet-cli "$@"
   else
      "$GOPATH"/bin/packet-cli "$@"
   fi
}

function packet_cli_get_ip() {
    counter_ip=${1:-0}
    packet_server_info=$(packet-cli -j device get -i "$device_id")
    packet_ip=$(echo "$packet_server_info" | jq -r '.ip_addresses[0]."address"?')
    if echo "$packet_ip" | grep -q "null" || [ -z "$packet_ip" ]; then
        if [ "$counter_ip" -gt "$repeat_count" ]; then
            echo "0.0.0.0"
            exit 1
        fi
        sleep 10
        packet_cli_get_ip $((counter_ip + 1))
    else
        echo "$packet_ip"
    fi
}

if [ -z "$device_id" ]; then
  fail "$help_text"
fi

packet_cli_get_ip
