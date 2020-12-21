#!/bin/sh

if [ $# -eq 0 ]
then
  echo Usage: "$0" port1:port2...
  exit
fi

EDEN=eden
which $EDEN || EDEN=../../eden
CFG=$($EDEN config get)

OLD=$($EDEN config get "$CFG" --key eve.hostfwd)
NEW=$OLD

for port in "$@"
do
  echo "$port" | grep '[0-9]\+:[0-9]\+' || continue
  port=$(echo "$port" | sed 's/^/"/;s/:/":"/g;s/$/"/')
  shift
  if echo "$OLD" | grep "$port"
  then
    echo Removing "$port" port redirection to EDEN config
    NEW=$(echo "$NEW" | sed "s/{\(.*\)$port\(.*\)}/{\1,\2}/; s/\(,\)\1/\1/g; s/[, ]*}/}/; s/{[, ]\+/{/; s/\([^, ]\),[, ]\+\([^, ]\)/\1,\2/")
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
