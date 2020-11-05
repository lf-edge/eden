#!/bin/sh

if [ $# -eq 0 ]
then
  echo Usage: "$0" port1:port2...
  exit
fi

cd ../..

EDEN=./eden

OLD=$($EDEN config get --key eve.hostfwd)
NEW=$OLD

for port in "$@"
do
  echo "$port" | grep '[0-9]\+:[0-9]\+' || continue
  port=$(echo "$port" | sed 's/^/"/;s/:/":"/g;s/$/"/')
  shift
  if echo "$OLD" | grep -v "$port"
  then
    echo Adding "$port" port redirection to EDEN config
    NEW=$(echo "$NEW" | sed "s/{\(.*\)}/{\1,$port}/")
  fi
done

if [ "$OLD" != "$NEW" ]
then
  echo $EDEN config set default --key eve.hostfwd --value \'"$NEW"\'
  $EDEN config set default --key eve.hostfwd --value "$NEW"
  $EDEN config get default --key eve.hostfwd
  echo $EDEN eve stop
  $EDEN eve stop
  sleep 5
  echo $EDEN eve start
  $EDEN eve start
fi
