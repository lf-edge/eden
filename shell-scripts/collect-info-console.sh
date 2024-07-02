#!/bin/sh

# This script runs collect-info.sh on EVE VM and downloads produced tarball
# using only serial console. This is especially useful when networking
# on the virtualized EVE is not working and therefore collect-info-ssh.sh
# is unable to do the same via SSH tunnel.

# Use output filename without colon, otherwise Github action "upload-artifact" complains.
OUTPUT="eve-info.tar.gz"

# 20 seconds should be enough for collect-info.sh to prepare tarball with debug info
# if run locally on a solid machine. However, on Github runners, it can take up to 2 minutes
# to complete (which is what we set from Github actions).
WAIT_TIME="${1:-20}"

# Switch to debug container where collect-info.sh is installed.
for i in $(seq 3); do
  {
    echo "eve verbose off"; echo "eve enter debug"; sleep 3;
    echo "which collect-info.sh"; sleep 3
  } | telnet 127.1 17777 | tee telnet.stdout
  grep -q "/usr/bin/collect-info.sh" telnet.stdout && break
  sleep 60
done

for i in $(seq 3); do
  {
    echo "rm -f /persist/eve-info*"; echo "/usr/bin/collect-info.sh";
    sleep $((WAIT_TIME+60*(i-1)))
  } | telnet 127.1 17777 | tee telnet.stdout
  TGZNAME="$(sed -n "s/EVE info is collected '\(.*\)'/\1/p" telnet.stdout)"
  [ -n "${TGZNAME}" ] && break
done

if [ -z "${TGZNAME}" ]; then
  echo "Failed to run collect-info.sh script"
  exit 1
fi

for i in $(seq 3); do
  {
    # Filename does not fit on one console line, we have to use asterisk.
    echo "echo \>\>\>\$(base64 -w 0 /persist/eve-info*)\<\<\<";
    # This is fairly quick even on Github runners - around 10 seconds, but depends
    # on the tarball size.
    sleep $((20+60*(i-1)))
  } | telnet 127.1 17777 | sed -n "s/>>>\(.*\)<<</\1/p" | base64 -id > "${OUTPUT}"
  [ -s "${OUTPUT}" ] && break
  echo "Failed to receive eve-info tarball, retrying..."
done

if [ ! -s "${OUTPUT}" ]; then
  echo "Failed to receive eve-info"
  exit 1
fi

FILESIZE="$(stat -c%s "$OUTPUT")"
echo "Received ${OUTPUT} with size ${FILESIZE}"
