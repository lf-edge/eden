#!/bin/bash
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

while true; do
   case "$1" in
      -l*) #shellcheck disable=SC2039
          location="${1/-l/}"
          if [ -z "$location" ]; then
             location="$2"
             shift
          fi
          shift
          ;;
      -c*) #shellcheck disable=SC2039
          server_conf="${1/-c/}"
          if [ -z "$server_conf" ]; then
             server_conf="$2"
             shift
          fi
          shift
          ;;
       -p*) #shellcheck disable=SC2039
          project="${1/-p/}"
          if [ -z "$project" ]; then
             project="$2"
             shift
          fi
          shift
          ;;
       -ns*) #shellcheck disable=SC2039
          name_suffix="${1/-ns/}"
          if [ -z "$name_suffix" ]; then
             name_suffix="$2"
             shift
          fi
          shift
          ;;
       -os*) #shellcheck disable=SC2039
          os="${1/-os/}"
          if [ -z "$os" ]; then
             os="$2"
             shift
          fi
          shift
          ;;
      -ipxe*) #shellcheck disable=SC2039
          ipxe_cfg_url="${1/-ipxe/}"
          if [ -z "$ipxe_cfg_url" ]; then
             ipxe_cfg_url="$2"
             shift
          fi
          shift
          ;;
       *) break
          ;;
   esac
done

function fail() { echo "ERROR: packet: $*" 1>&2; echo "00000000-0000-0000-0000-000000000000"; exit 1; }

help_text=$(cat << __EOT__
Usage: create.sh -l <location> -c <server configuration> -p <packet project id> -os <packet os> [OPTIONS]

OPTIONS:
   -ns <string>
      Name suffix for packet server.
   -ipxe <string>
      Url to ipxe cfg. Setup only if os param - custom_ipxe.

Create packet server with name eden-<name suffix>-<location>-<conf>. Returns the id of the created packet server to the stdout or 00000000-0000-0000-0000-000000000000
as server id on create fail.
------------------------------------------------------------------------------------------------------
__EOT__
)

function packet-cli() {
   if [ -e "$HOME"/go/bin/packet-cli ]; then
     "$HOME"/go/bin/packet-cli "$@"
   else
      "$GOPATH"/bin/packet-cli "$@"
   fi
}

function packet_cli_create_device() {
  counter_create=${1:-0}
  packet_id=$(packet-cli -j device create -f "$location" \
        -H eden-"$name_suffix"-"$location"-"$server_conf" \
        -P "$server_conf" -p "$project" \
        -o "$os" "$ipxe" | \
        jq -r '.["id"]?')
  if echo "$packet_id" | grep -q "null" || [ -z "$packet_id" ]; then
    if [ "$counter_create" -gt "10" ]; then
      fail "packet-cli thrown an error while creating"
    fi
    sleep 10
    packet_cli_create_device $((counter_create + 1))
  else
    echo "$packet_id"
  fi
}

if [ -z "$location" ] || [ -z "$server_conf" ] || [ -z "$project" ] || [ -z "$os" ]; then
  fail "$help_text"
fi

if [ "$PACKET_TOKEN" = "" ]; then
  fail "PACKET_TOKEN is empty, please set token"
fi;

ipxe=""
if [ -n "$ipxe_cfg_url" ]; then
   if ! [ "$os" = "custom_ipxe" ]; then
      fail "You can't setup -ipxe with $os"
   fi
   ipxe="-i $ipxe_cfg_url"
fi

"$SCRIPT_DIR"/tools/cli-prepare.sh
packet_cli_create_device
