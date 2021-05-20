#!/bin/bash

function packet_cli_get() {
    if ! [ -e "$HOME"/go/bin/packet-cli ] && ! [ -e "$GOPATH"/bin/packet-cli ]; then
        GO111MODULE=on go get github.com/packethost/packet-cli
    fi
}

packet_cli_get
