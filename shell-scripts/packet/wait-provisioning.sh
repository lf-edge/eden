#!/bin/bash
device_id=$1
timeout=${2:-1800}

fail() { echo "ERROR: packet: $*" 1>&2; exit 1; }

help_text=$(cat << __EOT__
Usage: wait-provisioning.sh <packet server id> <timeout>

A script awaiting provisioning to complete.
__EOT__
)

function packet-cli() {
   if [ -e "$HOME"/go/bin/packet-cli ]; then
     "$HOME"/go/bin/packet-cli "$@"
   else
      "$GOPATH"/bin/packet-cli "$@"
   fi
}

function packet_cli_wait_provisioning() {
    time=${1:-0}
    packet_server_info=$(packet-cli -j device get -i "$device_id")
    packet_state=$(echo "$packet_server_info" | jq -r '.state')
    if echo "$packet_state" | grep -q "null" || [ -z "$packet_state" ] || echo "$packet_state" | grep -q "provisioning"; then
        if [ "$time" -gt "$timeout" ]; then
            exit 1
        fi
        sleep 10
        packet_cli_wait_provisioning $((time + 14))
    fi
}

if [ -z "$device_id" ]; then
  fail "$help_text"
fi

packet_cli_wait_provisioning
