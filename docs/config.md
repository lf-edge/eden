# Eden Configuration

`eden` itself is an orchestrator. It manages several components together:

* adam
* eve
* redis as a backing store for adam
* eserver to serve up files to eve

Each of these can have their own config, and each can have multiple instances, running in their own ways.

eden maintains a configuration for running, that combines the setup of adam, eve devices, redis and the eserver. Each such
combination is called a "context" and is given a unique name.

Contexts are stored in a context file, by default `~/.eden/contexts/<name>.yml`. In addition, the currently active context
is indicated in `~/.eden/context.yml`. If you do not change the settings, eden will name the sole running context `default`,
stored in `~/.eden/contexts/default.yml`.

## Modifying, Adding or Removing Contexts

You can change settings by doing one of the following:

* editing a context file directly
* using the command `eden config edit <context>`

Similarly, you can remove a context by deleting its file, or using `eden config delete <context>`, and you can add a new one
by adding the file or using `eden config add <context>`.

You also can set the current context by editing `~/.eden/context.yml` or running `eden config set <context>`, and you can
list all known contexts with `eden config list`.

Every context you add creates the new instance of EVE with dedicated certificates
according to generated context file inside `~/.eden/contexts/` directory.
You can modify settings before running `eden setup`. Only one EVE instance can be run locally (in qemu). You need to stop it before starting another one.

Please see [Test configuring](../tests/README.md#Test configuring) section for details about tests config options with switching context.

### Example

To create a new config named `new1` and change the system to a new config

```console
export EDITOR=vim     # sets your default editor
eden stop             # stop anything else running
eden clean            # remove any old artifacts
eden config add new1  # creates a new context named "new1"
eden config edit new1 # edit the context "new1"
eden config set new1  # set the context "new1" as the current context
eden setup            # generate config and certificates based on context "new1"
eden start            # start everything up
```

To get the current config in json format:

```console
eden controller -m adam:// edge-node get-config
```

The above uses the adam mode to get the config for the edge-node, i.e. eve device.

To update the running eve os base image to another one, for example one stored in `dist/amd64/installer/rootfs.img`:

```console
eden controller -m adam:// edge-node eveimage-update dist/amd64/installer/rootfs.img
```

The last argument is the path tp the new rootfs image. It can be one of:

* relative file path, like the example above, `dist/amd64/installer/rootfs.img`
* absolute file path, e.g. `/home/user1/eve/dist/amd64/installer/rootfs.img`
* file URL, e.g. `file:///home/user1/eve/dist/amd64/installer/rootfs.img`
* http/s URL, e.g. `https://some.server.com/path/to/rootfs.img`
* OCI registry, e.g. `docker.io/lfedge/eve:6.7.8`

Note that when providing the OCI registry option, you can select to get the image from the local, eden-launched registry with
the `--registry=local` argument, e.g.

```console
eden controller -m adam:// edge-node eveimage-update --registry=local oci://docker.io/lfedge/eve:6.7.8
```

To set config property of EVE from [list](https://github.com/lf-edge/eve/blob/master/docs/CONFIG-PROPERTIES.md) you
can use the following command:

```console
eden controller -m adam:// edge-node update --config timer.config.interval=5
```

## Modifying of EVE config

You can obtain the current config of EVE with command `eden controller edge-node get-config --file=<file>`.
It produces current configuration in json format obtained from Adam in the provided file.
You can make modifications in this file (please do not forget to increment id.version field) and send it back with
`eden controller edge-node set-config --file=<file>`. You can also omit `file` in commands and use stdin and stdout
of them.
