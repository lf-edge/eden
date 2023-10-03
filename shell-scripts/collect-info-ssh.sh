#/bin/sh

# Use output filename without colon, otherwise Github action "upload-artifact" complains.
OUTPUT="eve-info.tar.gz"

ssh() {
  ./eden sdn fwd eth0 22 --\
    ssh -o StrictHostKeyChecking=no -p FWD_PORT -i ./dist/default-certs/id_rsa root@FWD_IP "$@"
}

scp() {
  ./eden sdn fwd eth0 22 --\
    scp -o StrictHostKeyChecking=no -P FWD_PORT -i ./dist/default-certs/id_rsa root@FWD_IP:$1 $2
}

if ./eden eve status | grep -q "no onboarded EVE"; then
  echo "Cannot get eve-info via SSH from non-onboarded EVE VM"
  exit 1
fi

# Give EVE 5 minutes at most to enable ssh access.
# This delay is typically needed if tests failed early.
for i in $(seq 60); do
  ./eden eve ssh : && break || sleep 5
done

ssh collect-info.sh | tee ssh.stdout
if [ $? -ne 0 ]; then
  echo "Failed to run collect-info.sh script"
  exit 1
fi

TGZNAME="$(cat ssh.stdout  | sed -n "s/EVE info is collected '\(.*\)'/\1/p")"
if [ -z "${TGZNAME}" ]; then
  echo "Failed to parse eve-info tarball filename"
  exit 1
fi

scp "${TGZNAME}" ${OUTPUT}
if [ $? -ne 0 ]; then
  echo "Failed to receive eve-info"
  exit 1
fi

FILESIZE="$(stat -c%s "$OUTPUT")"
echo "Received ${OUTPUT} with size ${FILESIZE}"