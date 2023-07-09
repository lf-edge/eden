# SDN Example with emulated poor network connectivity

Eden-SDN Network Model allows to configure traffic control individually for every network port.
Included is traffic shaping, i.e. limiting traffic to meet but not exceed a configured rate,
and emulating network impairments, such as packet delay, loss, corruption, reordering, etc.
This can be used to simulate poor network connectivity and observe how EVE is able to deal
with such challenging conditions.

In this example, traffic control parameters are set for the single and only network interface.
The intention is to model rather poor network connection with a low bandwidth and a high
percentage of packet loss or corruption. For the purposes of the showcase, we set every
available traffic control attribute to a specific non-default value.

Run the example with:

```shell
make clean && make build-tests
./eden config add default
./eden config set default --key sdn.disable --value false
./eden setup
./eden start --sdn-network-model $(pwd)/sdn/examples/poor-network/network-model.json
./eden eve onboard
./eden controller edge-node set-config --file $(pwd)/sdn/examples/poor-network/device-config.json
```
