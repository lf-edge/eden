#!/bin/sh

MAC="$1"

find_ip() {
  local NETNS="$1"
  local PREFIX=""
  if [ -n "$NETNS" ]; then
    PREFIX="ip netns exec $NETNS "
  fi
  $PREFIX arp -an | while read ARP_ENTRY; do
      local ENTRY_IP="$(echo "$ARP_ENTRY" | cut -d ' ' -f 2 | tr -d '()')"
      local ENTRY_MAC="$(echo "$ARP_ENTRY" | cut -d ' ' -f 4)"
      if [ "$MAC" = "$ENTRY_MAC" ]; then
        echo "$ENTRY_IP"
        return
      fi
  done
}

find_ip_all_ns() {
  ip netns list | while read NETNS; do
    NETNS="$(echo "$NETNS" | cut -d ' ' -f 1)"
    IP="$(find_ip "$NETNS")"
    if [ -n "$IP" ]; then
      echo "$IP"
      return
    fi
  done
}

# Main network namespace.
IP="$(find_ip)"
if [ -n "$IP" ]; then
  echo "$IP"
  exit 0
fi

# Other (named) network namespaces.
IP="$(find_ip_all_ns)"
if [ -n "$IP" ]; then
  echo "$IP"
  exit 0
fi

echo "Failed to find ARP entry for MAC=$MAC" >&2
exit 1