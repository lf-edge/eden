#!/bin/bash
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
EDEN_DIR=$SCRIPT_DIR/../../

function eden_get_ipxe_cfg_url() {
  if ! [ -f "$EDEN_DIR"/dist/default-images/eve/tftp/ipxe.efi.cfg ]; then
    exit 1
  fi

  set_url_str=$(< "$EDEN_DIR"/dist/default-images/eve/tftp/ipxe.efi.cfg grep "set url")
  echo "${set_url_str/"set url "/""}ipxe.efi.cfg"
}

ipxe_cfg_url=$(eden_get_ipxe_cfg_url)
if [ -z "$ipxe_cfg_url" ]; then
  fail "First, configure eden for network boot"
fi

"$SCRIPT_DIR"/create.sh "$@" -os custom_ipxe -ipxe "$ipxe_cfg_url"