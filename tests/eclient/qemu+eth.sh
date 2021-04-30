#!/bin/sh

test -n "$EDEN_CONFIG" || EDEN_CONFIG=default

EDEN=eden
DIR=$(dirname "$0")
PATH=$DIR:$DIR/../../bin:$PATH

dist=$($EDEN config get "$EDEN_CONFIG" --key eden.root)

dd if=/dev/zero of="$dist/stick.raw" bs=1K count=1

cat >> ~/.eden/"$EDEN_CONFIG"-qemu.conf <<END


[netdev "hostnet"]
  type = "user"

[device "net"]
  driver = "virtio-net-pci"
  netdev = "hostnet"
  bus = "pcie.0"
  addr = "19.0"
END

cat <<END
To activate the changes in the config, you need to restart EVE:
  $EDEN eve stop
  $EDEN eve start
END
