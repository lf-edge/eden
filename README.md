# Eden

Eden is the simpliest way to setup & test [EVE](https://github.com/lf-edge/eve) and [Adam](https://github.com/lf-edge/adam).

Eden is inspired by Kubernetes workflows and CLI

Eden contains  series of integration tests implemented in Golang. Tests are structured as normal Golang tests by using ```_test.go``` nomenclature and be available for test runs using standard go test framework.

## Install Prerequisites

Install requirements from [eve](https://github.com/lf-edge/eve#install-dependencies)

Also, you need to install `telnet`.

You need to be able to run docker commands and able to access virtualization accelerators (KVM on Linux or machyve on Mac OS X)

## Quickstart
```
git clone https://github.com/lf-edge/eden.git
cd eden
make bin
eden setup
eden start
eden status
eden test
```

To find out what is running and where
```
eden status
```

To create a new config named new1 and change the system to a new config 

```
EXPORT EDITOR=vim
eden config add new1
eden config edit new1
eden config set new1 
eden stop
eden clean
eden setup
eden start
```

To get the config & update eve image stored in dist/amd64/installer/rootfs.img  after eden has started

``` 
 eden controller -m adam:// edge-node get-config 
 eden controller -m adam:// edge-node eveimage-update dist/amd64/installer/rootfs.img
``` 

## Remote access to eve
Main way: 
```
eden eve ssh
```

You can also use telnet connection for Qemu console:
``` 
eden eve console
``` 

## Run applications on EVE

Notice: if you are on QEMU there is a limited number of exposed ports. Add some if you want to expose more.
```
hostfwd: 
         {{ .DefaultSSHPort }}: 22 
         5912: 5901 
         5911: 5900 
         8027: 8027 
         8028: 8028
``` 

Deploy nginx server from dockerhub. Expose port 80 of the container to port 8028 of eve. 
``` 
eden pod deploy -p 8028:80 docker://nginx
``` 

Deploy a VM from Openstack. Initialize root user with password - 'passw0rd'  Expose port 22 of the VM (ssh) to port 8027 of eve for ssh:
``` 
eden pod deploy -p 8027:22 http://cdimage.debian.org/cdimage/openstack/current/debian-10.4.3-20200610-openstack-amd64.qcow2 -v debug --metadata='#cloud-config\npassword: passw0rd\nchpasswd: { expire: False }\nssh_pwauth: True\n'
``` 

List running applications and their ip/ports
```
eden pod ps
```

## Help

You can get more information about `make` actions by running `make help`.

More information can be found at the
[wiki](https://wiki.lfedge.org/display/EVE/EDEN) or in the #eve-help channel in
[slack](slack.lfedge.org/).

## Utilites

Eden is controlled by a single command named (in a secret code) `eden`. It has multple sub-commands and options.
Run `eden help` to see sub-commands.

To build `eden`:

```
make build
```

You can build it for different computer architectures and operating systems by passing `OS` and `ARCH` options.
The default, however, is for the architecture and OS on which you are building it.

```
make build OS=linux
make build ARCH=arm64
```

The generated command is place in `./dist/bin/eden-<arch>-<os>`, for example `eden-darwin-amd64` or `eden-linux-arm64`.
To ease your life, a symlink is placed in the local directory named `eden` for your current architecture and OS.

The current sub-commands are:
   * `config` -- work with different configurations of environment; 
   * `status` -- get status of all components;
   * `setup` --  get all components that are specified in the config and ensure they are ready for startup;
   * `start` -- start all components
   * `stop` -- stop all components;
   * `test` -- run tests; 
   * `info` -- displays Info records, accepts regular expression as a filter;
   * `log` -- displays Log records, accepts regular expression as a filter;
   * `metric` -- displays Metric records, accepts regular expression as a filter ;
   * `eve` -- sub-commands for interact with EVE.
   * `controller` -- sub-commands to update EVE.
   * `pod` -- work with applications running on EVE (containers and VMs)
