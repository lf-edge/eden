# SDN Example with IPv6 networks

This is a basic example of EVE connecting to the Adam controller over an IPv6 network.
The setup is still a work in progress and will be extended to include local network
instances with IPv6 connectivity. However, additional development is needed on the EVE
side to support this functionality.

Run the example with:

```shell
make clean && make build-tests
./eden config add default
./eden config set default --key eden.enable-ipv6 --value true
./eden config set default --key sdn.disable --value false
./eden config set default --key sdn.enable-ipv6 --value true
./eden setup
./eden start --sdn-network-model $(pwd)/sdn/examples/ipv6-network/network-model.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/ipv6-network/device-config.json
```
