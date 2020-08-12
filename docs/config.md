# Eden Configuration

`eden` itself is an orchestrator. It manages several components together:

* adam
* eve
* redis as a backing store for adam
* eserver to serve up files to eve

Each of these can have their own config, and each can have multipe instances, running in their own ways.

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

### Example

To create a new config named `new1` and change the system to a new config

```
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

```sh
eden controller -m adam:// edge-node get-config
```

The above uses the adam mode to get the config for the edge-node, i.e. eve device.

To update the running eve os base image to another one, for example one stored in `dist/amd64/installer/rootfs.img`:

```
 eden controller -m adam:// edge-node eveimage-update dist/amd64/installer/rootfs.img
``
