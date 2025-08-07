#!/bin/sh

# This script runs collect-info.sh on EVE VM and downloads produced tarball
# using only serial console. This is especially useful when networking
# on the virtualized EVE is not working and therefore collect-info-ssh.sh
# is unable to do the same via SSH tunnel.

# Use output filename without colon, otherwise Github action "upload-artifact" complains.
OUTPUT="eve-info.tar"

# 20 seconds should be enough for collect-info.sh to prepare tarball with debug info
# if run locally on a solid machine. However, on Github runners, it can take up to 2 minutes
# to complete (which is what we set from Github actions).
WAIT_TIME="${1:-20}"

# Get the port number used for the EVE console.
CONSOLE_PORT="$(./eden config get --key eve.telnet-port)"
if ! echo "$CONSOLE_PORT" | grep -qE '^[0-9]+$'; then
    echo "Failed to get EVE console port"
    exit 1
fi

# Check if EVE is running and is listening on the console port.
if ! netstat -nl | grep -qE ".*:${CONSOLE_PORT}.*LISTEN"; then
    echo "EVE is not running or listening on the console port"
    exit 1
fi

# Switch to debug container where collect-info.sh is installed.
for i in $(seq 3); do
  {
    echo "eve verbose off"; echo "eve enter debug"; sleep 3;
    echo "which collect-info.sh"; sleep 3
  } | telnet 127.1 "${CONSOLE_PORT}" | tee telnet.stdout
  grep -q "/usr/bin/collect-info.sh" telnet.stdout && break
  sleep 60
done

for i in $(seq 3); do
  {
    echo "/usr/bin/collect-info.sh";
    sleep $((WAIT_TIME+60*(i-1)))
  } | telnet 127.1 "${CONSOLE_PORT}" | tee telnet.stdout
  grep -q "EVE info is collected" telnet.stdout && break
done

# We stream the tarball content to stdout, so we can read it
# and decode it in one go.
# The tarball is base64 encoded to avoid issues with binary data.
# The streaming can take a while, so we retry a few times.
# If that still doesn't help we need to consider a different approach.
for i in $(seq 3); do
  {
    echo "tar -C /persist -cf - eve-info | base64 -w 0 > /persist/eve-info.base64";
    echo "echo \>\>\>\$(cat /persist/eve-info.base64)\<\<\<";
    sleep $((WAIT_TIME+60*(i-1)))
  } | telnet 127.1 "${CONSOLE_PORT}" | sed -n "s/>>>\(.*\)<<</\1/p" | base64 -id > "${OUTPUT}"
  [ -s "${OUTPUT}" ] && break
  echo "Failed to receive eve-info tarball, retrying..."
done

if [ ! -s "${OUTPUT}" ]; then
  echo "Failed to receive eve-info"
  exit 1
fi

FILESIZE="$(stat -c%s "$OUTPUT")"
echo "Received ${OUTPUT} with size ${FILESIZE}"
