#!/bin/sh
# shellcheck disable=SC3043

MAC="$1"

find_ipv4() {
  local NETNS="$1"
  local PREFIX=""
  if [ -n "$NETNS" ]; then
    PREFIX="ip netns exec $NETNS"
  fi

  $PREFIX arp -an | while read -r ARP_ENTRY; do
    local ENTRY_IP
    ENTRY_IP="$(echo "$ARP_ENTRY" | awk '{print $2}' | tr -d '()')"
    local ENTRY_MAC
    ENTRY_MAC="$(echo "$ARP_ENTRY" | awk '{print $4}')"
    if [ "$MAC" = "$ENTRY_MAC" ]; then
      echo "$ENTRY_IP"
      return
    fi
  done
}

find_ipv6() {
  local NETNS="$1"
  local PREFIX=""
  if [ -n "$NETNS" ]; then
    PREFIX="ip netns exec $NETNS"
  fi

  $PREFIX ip -6 neigh show | while read -r ND_ENTRY; do
    local ENTRY_IP
    ENTRY_IP="$(echo "$ND_ENTRY" | awk '{print $1}')"
    if [ "${ENTRY_IP#fe80:}" != "$ENTRY_IP" ]; then
      # Skip link-local address.
      continue
    fi
    local ENTRY_MAC
    ENTRY_MAC="$(echo "$ND_ENTRY" | awk '{for(i=1;i<=NF;i++) if($i=="lladdr") print $(i+1)}')"
    if [ "$MAC" = "$ENTRY_MAC" ]; then
      echo "$ENTRY_IP"
      return
    fi
  done
}

find_ipv4_all_ns() {
  ip netns list | while read -r NETNS; do
    NETNS="$(echo "$NETNS" | cut -d ' ' -f 1)"
    IP="$(find_ipv4 "$NETNS")"
    if [ -n "$IP" ]; then
      echo "$IP"
      return
    fi
  done
}

find_ipv6_all_ns() {
  ip netns list | while read -r NETNS; do
    NETNS="$(echo "$NETNS" | cut -d ' ' -f 1)"
    IP="$(find_ipv6 "$NETNS")"
    if [ -n "$IP" ]; then
      echo "$IP"
      return
    fi
  done
}

# Prefer IPv4 over IPv6.
# - ARP table lookup in the main network namespace.
IP="$(find_ipv4)"
if [ -n "$IP" ]; then
  echo "$IP"
  exit 0
fi
# - ARP table lookup in every other (named) network namespaces.
IP="$(find_ipv4_all_ns)"
if [ -n "$IP" ]; then
  echo "$IP"
  exit 0
fi

# Next try to get IPv6 address.
# - Neighbor Discovery table lookup in the main network namespace.
IP="$(find_ipv6)"
if [ -n "$IP" ]; then
  echo "$IP"
  exit 0
fi
# - Neighbor Discovery table lookup in every other (named) network namespaces.
IP="$(find_ipv6_all_ns)"
if [ -n "$IP" ]; then
  echo "$IP"
  exit 0
fi

echo "Failed to get IP address for MAC=$MAC" >&2
exit 1