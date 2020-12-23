#!/bin/sh

if [ $# -eq 0 ]
then
  echo Usage: "$0" port1:port2...
  exit
fi

EDEN=eden
DIR=$(dirname "$0")
PATH=$DIR:$DIR/../../bin:$PATH
CFG=$($EDEN config get)

OLD=$($EDEN config get "$CFG" --key eve.hostfwd)
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
  echo $EDEN config set "$CFG" --key eve.hostfwd --value \'"$NEW"\'
  $EDEN config set "$CFG" --key eve.hostfwd --value "$NEW"
  echo $EDEN config get "$CFG" --key eve.hostfwd
  $EDEN config get "$CFG" --key eve.hostfwd
  echo $EDEN eve stop
  $EDEN eve stop
  sleep 5
  echo $EDEN eve start
  $EDEN eve start
fi
