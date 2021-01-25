#!/bin/bash

#./eden config add
#./eden setup
# shellcheck source=/dev/null
source ~/.eden/activate.sh

#eden start
#eden eve onboard
eden eve stop

echo eden config get --key eden.root
DIR=$(eden config get --key eden.root || exit)/..
echo "Eden directory: $DIR"
echo eden config get --key eve.name
NAME=$(eden config get --key eve.name || exit)
echo "EVE name: $NAME"
echo eden config get --key eve.accel
ACCEL=$(eden config get --key eve.accel || exit)
echo "EVE accel: $ACCEL"

pushd "$DIR" || exit

if [ "$NAME" != default ]
then
    CLIENT="$NAME-client"
    SERVER="$NAME-server"
else
    CLIENT=client
    SERVER=server
fi

echo eden config add "$CLIENT" -v debug
eden config add "$CLIENT" -v debug
echo eden config set "$CLIENT" --config "$CLIENT" --key eve.accel --value "$ACCEL" -v debug
eden config set "$CLIENT" --config "$CLIENT" --key eve.accel --value "$ACCEL" -v debug
echo eden config get "$CLIENT" --config "$CLIENT" --key eve.accel -v debug
eden config get "$CLIENT" --config "$CLIENT" --key eve.accel -v debug
echo eden config set "$CLIENT" --config "$CLIENT" --key eve.hostfwd --value '{"2232":"2232"}' -v debug
eden config set "$CLIENT" --config "$CLIENT" --key eve.hostfwd --value '{"2232":"2232"}' -v debug
echo eden config get "$CLIENT" --config "$CLIENT" --key eve.hostfwd
eden config get "$CLIENT" --config "$CLIENT" --key eve.hostfwd
echo eden config set "$CLIENT" --config "$CLIENT" --key eve.telnet-port --value 7778 -v debug
eden config set "$CLIENT" --config "$CLIENT" --key eve.telnet-port --value 7778 -v debug
echo eden config get "$CLIENT" --config "$CLIENT" --key eve.telnet-port
eden config get "$CLIENT" --config "$CLIENT" --key eve.telnet-port

echo eden config add "$SERVER" -v debug
eden config add "$SERVER" -v debug
echo eden config set "$SERVER" --config "$SERVER" --key eve.accel --value "$ACCEL" -v debug
eden config set "$SERVER" --config "$SERVER" --key eve.accel --value "$ACCEL" -v debug
echo eden config get "$SERVER" --config "$SERVER" --key eve.accel -v debug
eden config get "$SERVER" --config "$SERVER" --key eve.accel -v debug
echo eden config set "$SERVER" --config "$SERVER" --key eve.hostfwd --value '{"2233":"2233","1234":"1234"}' -v debug
eden config set "$SERVER" --config "$SERVER" --key eve.hostfwd --value '{"2233":"2233","1234":"1234"}' -v debug
echo eden config get "$SERVER" --config "$SERVER" --key eve.hostfwd
eden config get "$SERVER" --config "$SERVER" --key eve.hostfwd
echo eden config set "$SERVER" --config "$SERVER" --key eve.telnet-port --value 7779 -v debug
eden config set "$SERVER" --config "$SERVER" --key eve.telnet-port --value 7779 -v debug
echo eden config get "$SERVER" --config "$SERVER" --key eve.telnet-port
eden config get "$SERVER" --config "$SERVER" --key eve.telnet-port

popd || exit
