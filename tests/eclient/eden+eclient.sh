#!/bin/sh

cd ../..

EDEN=./eden

if $EDEN config get --key eve.hostfwd | grep -v 2223
then
  echo Adding 2223 port for eclient\'s SSH to EDEN config
  PORTS=$($EDEN config get --key eve.hostfwd | sed 's/map\[\(.*\)\]/\1/; s/^/2223:2223 /')
  $EDEN config set default --key eve.hostfwd --value "$PORTS"
fi

grep 'hostfwd = "tcp::2223-:2223"' ~/.eden/default-qemu.conf && exit

echo Adding 2223 port for eclient\'s SSH to EDEN QEMU config
ed ~/.eden/default-qemu.conf <<END
/hostfwd = "tcp::2222-:22"
a
  hostfwd = "tcp::2223-:2223"
.
wq
END

echo $EDEN stop
$EDEN stop
echo $EDEN start
$EDEN start
