#!/bin/bash

# shellcheck source=/dev/null
source ~/.eden/activate.sh

echo eden config get --key eve.name
NAME=$(eden config get --key eve.name || exit)
echo "EVE name: $NAME"

if [ "$NAME" != default ]
then
    CLIENT="$NAME-client"
    SERVER="$NAME-server"
else
    CLIENT=client
    SERVER=server
fi

eden eve stop --config "$CLIENT"
eden eve stop --config "$SERVER"
eden clean --config "$CLIENT"
eden clean --config "$SERVER"
eden config delete "$CLIENT"
eden config delete "$SERVER"

eden eve start
