#!/bin/sh

cd ../..

EDEN=./eden

if $EDEN config get --key eve.hostfwd | grep -v 2223
then
  echo Adding 2223 port for eclient\'s SSH to EDEN config
  PORTS=$($EDEN config get default --key eve.hostfwd | sed 's/map\[\(.*\)\]/{"\1 2223:2223"}/;s/ /","/g;s/:/":"/g')
  $EDEN config set default --key eve.hostfwd --value "$PORTS"
  $EDEN config get default --key eve.hostfwd
  echo $EDEN stop
  $EDEN stop
  echo $EDEN start
  $EDEN start
fi
