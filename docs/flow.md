# Flow

What is the flow of launching an entire setup?

1. Make sure you have `eden`, as well as any prerequisites, like `docker` and `qemu`. See the main [README](../README.md).
1. `eden setup` - this does the following:
   * reads the configuration from your context and validates it
   * generates the certificates for adam and for eve
   * generates the config directory for eve, which includes the above certificates, as well as a `server` file pointing at the soon-to-be-started adam
   * gets a live eve image, which either is downloaded from docker hub or built
1. `eden start` - this does the following:
   * start redis in docker
   * start adam in docker
   * start eserver in docker
   * start eve in qemu
1. `eden eve onboard` - this does the following:
   * waits for eve to generate its device certificate
   * loads the device certificate into adam
   * waits for eve to onboard successfully to adam

## Modifying the Flow

While the above flow controls everything, you can use only certain parts of it. Common use cases are:

* Using a different live eve image, e.g. building a custom eve image, but launching and controlling it via eden
* Running onboarding manually

### Manual Onboarding

To onboard manually, simply skip the `eden eve onboard` step. The eve device already is configured to generate its device certificate
and attempt to communicate with the controller in `/var/config/server`, i.e. the adam device. You simply skip the `eden eve onboard` step,
and communicate directly with adam.

`adam` can be controlled using the `adam admin` command. If you run `eden status`, it will tell you exactly where it is reachable. If you
do not have the `adam` command installed, you can do so via the docker container:

```sh
$ docker exec -it eden_adam sh
# adam admin
```

### Different EVE Image

TBD

