# Eden

Eden is the simplest way to setup & test [EVE](https://github.com/lf-edge/eve)
and [Adam](https://github.com/lf-edge/adam).

Eden is inspired by Kubernetes workflows and CLI

All components of Eden -- the main executable file `eden`
and test binaries can be used as stand-alone applications
without the need to install and configure the development
environment on your computer. Generally, you will only need
a text editor to configure the system and create test scenarios
and scripts using `eden` and pre-compiled test files. To create
more complex tests, you can use standard Go testing machinery.

Eden contains series of integration tests implemented in Golang.
Tests are structured as normal Golang tests by using `_test.go`
nomenclature and be available for test runs using standard go test framework.

## Install Prerequisites

Install requirements from [eve](https://github.com/lf-edge/eve#install-dependencies).
Eden requires QEMU version 4.x+.

Also, you need to install `telnet` and `squashfs-tools` (`squashfs` for Mac OS X).

If you want to use Eden with RPI on Linux, you also need to install
`binfmt-support` and `qemu-user-static`.

You need to be able to run docker commands and able to access virtualization accelerators
(KVM on Linux or machyve on Mac OS X).
For Linux, you need to add current user into docker and kvm groups:

```console
sudo usermod -aG docker $USER
newgrp docker
sudo usermod -aG kvm $USER
newgrp kvm
```

If you want to be able to launch VMs or containers-in-VMs on Mac
You need to use Parallels. ([Parallels Manual](./docs/parallels.md))

## Quickstart

```console
git clone https://github.com/lf-edge/eden.git
cd eden
make clean
make build-tests
./eden config add default
./eden setup
source ~/.eden/activate.sh
./eden start
./eden eve onboard
./eden status
EDEN_TEST=small ./eden test tests/workflow -v debug
```

Note: Don't forget to call clean if you want to try the  installation again.
Call either `make clean` or

```console
eden stop
eden clean --current-context=false
```

Note for cloud and VM users: eden and eve use virtualization.
To run in VM-based environments, see [the cloud document](./docs/cloud.md).

To find out what is running and where:

```console
eden status
```

## Eden's shell settings

For more ease of use of Eden, you can use the automatically generated setup files for your shell:

* for BASH -- `source ~/.eden/activate.sh`
* for TCSH -- `source ~/.eden/activate.csh`

These settings add the Eden's binaries directory to the PATH environment variable and add the "EDEN\_<current\_config>\_" label to the command prompt.

In setup files defined some functions (BASH) and aliases (TCSH) for work with configs:

* eden+config <config\_name> -- add new config for Eden
* eden-config <config\_name> -- remove config from Eden and switch to 'default'
* eden_config <config\_name> -- switch Eden to config and change prompt

To deactivate this settings call `eden_deactivate` function.

## Eden Config

Eden's config is controlled via a yaml file, overriddable using command-line options.
In most cases, the defaults will work just fine for you.
For more informaton, see [docs/config.md](./docs/config.md).

You can run multiple instances of EVE with different configurations:

```console
make build-tests
./eden config add default
./eden config add t1
./eden config set t1 --key eve.hostfwd --value '{"2223":"22"}'
./eden config set t1 --key eve.telnet-port --value 7778
./eden setup -v debug # default config
./eden setup --config t1 -v debug
./eden start -v debug # start all services and EVE with default config
./eden eve start --config t1 -v debug # start second EVE with t1 config
```

## Remote access to eve

The main way to get a shell, especially once the device is fully registered to
its `adam` controller, is via ssh:

```console
eden eve ssh
```

You can run a certain command remotely by passing it as argument:

```console
eden eve ssh "ls -la"
```

If you need access to the actual device console, run:

```console
eden eve console
```

## Run applications on EVE

Notice: if you are on QEMU there is a limited number of exposed ports.
Add some if you want to expose more.

```console
hostfwd:
         {{ .DefaultSSHPort }}: 22
         5912: 5901
         5911: 5900
         8027: 8027
         8028: 8028
```

### Deploy Applications

You can deploy images from different sources. In addition, eden deploys
its own http server and docker registry.
Each source has its default format, but you can override it with
the `--format` flag (`container`,`qcow2` or `raw`). The defaults are:

* `docker://` - OCI image
* `http://` - VM qcow2
* `https://` - VM qcow2 image
* `file://` - VM qcow2 image

#### Docker Image

Deploy nginx server from dockerhub. Expose port 80 of the container
to port 8028 of eve.

```console
eden pod deploy -p 8028:80 docker://nginx
```

#### Docker Image with volume

If Docker image contains `Volume` annotation inside, Eden will add volumes for every of it.
You can modify behavior with `--volume-type` flag:

* choose type of volume (`qcow2`, `raw` or `oci`)
* skip this action with `none`

#### Docker Image from Local Registry

eden starts a local registry image, running on the localhost at port `5000`
(configurable). You can start a docker image from that registry by
passing it the `--registry=local` option. The default for `--registry`
is the hostname given by the registry. For example,
`docker://docker.io/library/nginx` defaults to `docker.io`.
But if you pass it the `--registry=local` option,
it will try to get it from the local, eden-managed registry.

```console
eden pod deploy --registry=local docker://nginx
```

If the image is not available in the local registry, it will fail.
You can load it up (see the next section).

#### Loading Image into Local Registry

You can load the local registry with images.

```console
eden registry load library/nginx
```

eden will do the following:

1. Check if the image is in your local docker image cache, i.e. `docker image ls`.
If it is, load it into the local registry and done.
1. If it is not, try to pull it from the remote registry via `docker pull`.
Once that is done, it will load it into the local registry.

#### VM Image

Deploy a VM for Openstack. Initialize `ubuntu` user with password `passw0rd`.
Expose port 22 of the VM (ssh) to port 8027 of eve for ssh:

```console
eden pod deploy -p 8027:22 https://cloud-images.ubuntu.com/releases/groovy/release-20210108/ubuntu-20.10-server-cloudimg-amd64.img -v debug --metadata='#cloud-config\npassword: passw0rd\nchpasswd: { expire: False }\nssh_pwauth: True\n'
```

Deploy a VM from a local file. This will cause the local file
to be uploaded to the eden-deployed `eserver`, which, in turn,
will deploy it to eve:

```console
eden pod deploy file:///path/to/some.img
```

#### VM Image from Docker Registry

Deploy a VM that is in a docker image, whether in OCI Artifacts format,
or wrapped in a container. All formats from
[edge-containers](https://github.com/lf-edge/edge-containers) are supported.

```console
eden pod deploy docker://some/image:container-tag --format=qcow2
```

### List Applications

List running applications, their names, ip/ports

```console
eden pod ps
```

#### Edit forwarded ports of Applications

To modify port forward you can run `eden pod modify <app name> -p <new port forward>` command.

For example for `laughing_maxwell` app name and forwarding of 8028<->80 TCP port you can run:

```console
eden pod modify laughing_maxwell -p 8028:80
```

### View Logs of Applications

To see logs of app you should get name (`app_name` in the example below)
of it and run:

```console
eden pod logs app_name
```

You can choose information to display by providing `--fields` flag
from the list (or define multiple fields separated by the comma):

* `log` - to view log objects
* `info` - to view info objects
* `metric` - to view metric objects
* `app` - to view console output

You can show only N last lines by setting of `--tail` flag.

### Delete Application

To delete the app you should get name (`app_name` in the example below)
of it and run:

```console
eden pod delete app_name
```

The command will also delete volumes of app. If you want to save volumes,
please use `--with-volumes=false` flag.

## Different EVE Builds

eden runs EVE using a `live.img` file with an embedded
config partition, which eve configures. In a normal run,
eden takes care of getting and installing an eve OS disk image for you.
However, you can configure eden to use a different image.
This is particularly useful for eden's primary use case: testing EVE.

To work with custom EVE images, see [docs/eve-images.md](./docs/eve-images.md).

## Tests

To run tests make sure you called `make build-tests`.

The easy way to run tests is to call `eden test <test folder>`

For example -- run reboot test:

`eden test tests/reboot/`

Some tests may accept parameters - run Log/Metrics/Info test
in debug mode with timeout of 600 seconds and requiring 3 messages of each type

`eden test tests/lim/ -v debug -a '-timewait 600 -number 3'`

You can find more detailed information about `eden test`
in [tests/README.md](tests/README.md) and [tests/escript/README.md](tests/escript/README.md)

## Google Cloud support

Eden is enough to deploy Eve on Google Cloud. We are going to make
it with one command, but now the process is:

* Make an image
* Upload it to GCP and run
* start eden (if not started yet)
* Onboard eve (use `eden eve onboard`)

Step 1 : Make an image. You need to specify IP of Adam,
by defalut Adam is a container inside machine with Eden,
so put there an IP that is accessible from gcp

```console
make build
eden config add default --devmodel GCP
eden config set default --key adam.eve-ip --value <IP of Adam/Eden>
eden setup
```

Step 2 : Upload image to gcp and run it. You will need
a google service key json.
[creating-managing-service-account-keys](https://cloud.google.com/iam/docs/creating-managing-service-account-keys)

```console
eden utils gcp image -k <PATH TO SERVICE KEY FILE> -p <PROJECT ON GCP> --image-name <NAME OF IMAGE ON GCP> upload <PATH TO EVE IMAGE>
eden utils gcp vm -k <PATH TO SERVICE KEY FILE> -p <PROJECT ON GCP> --image-name <NAME OF IMAGE ON GCP> --vm-name=<NAME OF VM ON GCP> run
```

`eden utils gcp` also supports:

* --bucket-name  for images
* --machine-type for vm

Step 3 : Configure the firewall and make sure ADAM is exposed in the network
Note that the firewall may be active on GCP. Allow connections (create rules)

```console
BWD=$(./eden utils gcp vm get-ip --vm-name eve-eden-one -k <google json key path>)
./eden utils gcp firewall -k <google json key path>  --source-range $BWD --name <firewall_rule_name>
```

ADAM should be publicly available from GCP machine.

Step 4 : Start eden and onboard Eve

```console
eden start
eden eve onboard
```

Note. You can see logs from gcp console:

```console
eden utils gcp vm -k <google json key path> -p <PROJECT ON GCP> --vm-name=<NAME OF VM ON GCP> log
```

## Raspberry Pi 4 support

Eden is the only thing you need to work with Raspberry and deploy containers there:

Step 0: If you already have EVE on your SD and want to try the new version, please format SD card
with zeroes (at least first 700 MB).

Step 1: Install EVE on Raspberry instead of any other OS.

Prepare Raspberry image

```console
git clone https://github.com/lf-edge/eden.git
cd eden
make build
eden config add default --devmodel RPi4
eden setup
eden start
```

Then you will have an .img that can be transferred to SD card:
[installing-images](https://www.raspberrypi.org/documentation/installation/installing-images/).

For example for MacOS:

```console
diskutil list
diskutil unmountDisk /dev/diskN
sudo dd bs=1m if=path_of_your_image.img of=/dev/rdiskN; sync
sudo diskutil eject /dev/rdiskN
```

Put in SD card into Raspberry and power it on

Step 2: Connect to  Raspberry and run some app.

```console
eden eve onboard
eden pod deploy -p 8028:80 docker://nginx
```

After these lines you will have nginx available on public EVE IP at port 8028

Use

```console
eden status
eden pod ps
```

to get the status of the deployment

You can try to boot the Windows 10 ARM64 image on RPi
(you need not less than 32GB SD card):

```console
eden pod deploy docker://itmoeve/eci-windows:2004-compressed-arm64 --vnc-display=1 --memory=2GB --cpus=2
```

You also can use RDP by adding the port forwarding (`-p 3389:3389`)
to the command above.

With running status in `eden pod ps` you can try to connect via VNC
on the public EVE IP at port 5901 with credentials `IEUser:Passw0rd!`.

### Obtain information about EVE on SD card

To get information from SD card about previously flashed instance of EVE you should:

Step 0 (for MacOS): Install required packets after installing of [brew](https://brew.sh/):

```console
brew cask install osxfuse
brew install ext4fuse squashfuse
```

Step 1: List partitions.

For MacOS `diskutil list`

For Linux `lsblk`

You should find your SD and 5 partitions on it.

Step 2: Mount partitions of SD card on your PC

For MacOS (`diskN` is SD card from step 1)

```console
sudo umount /dev/diskN*
sudo squashfuse /dev/diskNs2 ~/tmp/rootfs -o allow_other
sudo ext4fuse /dev/diskNs9 ~/tmp/persist -o allow_other
```

For Linux (`sdN` is SD card from step 1)

```console
mkdir -p ~/tmp/rootfs ~/tmp/persist
sudo umount /dev/sdN*
sudo mount /dev/sdN2 ~/tmp/rootfs
sudo mount /dev/sdN9 ~/tmp/persist
```

Step 3: Extract files and save them:

* syslog.txt contains logs of EVE: `sudo cp ~/tmp/persist/rsyslog/syslog.txt ~/syslog.txt`
* eve-release contains version of EVE: `sudo cp ~/tmp/rootfs/etc/eve-release ~/eve-release`

Step 4: Umount and eject SD

For MacOS

```console
sudo umount /dev/diskN*
sudo diskutil eject /dev/diskN
```

For Linux

```console
sudo umount /dev/sdN*
sudo eject /dev/sdN
```

## General image

You can use Eden to run EVE on your device. Note, that you must be in the same network.

### Step 1: Make an image

```console
make build
eden config add default --devmodel general
eden setup
```

You can set the architecture of EVE image with `--arch` flag (amd64 or arm64).
You will see `EVE image ready` and full path to the image.

### Step 2: deploy the image to your device

You should copy the image to you device. For example, you can use `dd` command from any of bootable USB live image.

### Step 3: onboard

```console
eden start
eden eve onboard
```

You will see log of onboarding the EVE into Adam controller. When this is done, you can launch applications.

## Help

You can get more information about `make` actions by running `make help`.

More information can be found at the
[wiki](https://wiki.lfedge.org/display/EVE/EDEN) or in the #eve-help channel in
[slack](https://slack.lfedge.org/).

## Utilites

Eden is controlled by a single command named (in a secret code) `eden`.
It has multple sub-commands and options.
Run `eden help` to see sub-commands.

To build `eden`:

```console
make build
```

To build `eden` and tests inside eden
It's better to call `eden config add` first, so the build command
can build tests for the desired architecture

```console
make build-tests
```

You can build it for different computer architectures and
operating systems by passing `OS` and `ARCH` options.
The default, however, is for the architecture and OS on
which you are building it.

```console
make build OS=linux
make build ARCH=arm64
```

The generated command is place in `./dist/bin/eden-<arch>-<os>`,
for example `eden-darwin-amd64` or `eden-linux-arm64`.
To ease your life, a symlink is placed in the local directory named
`eden` for your current architecture and OS.

The current sub-commands are:

* `config` -- work with different configurations of environment;
* `status` -- get status of all components;
* `setup` --  get all components that are specified in the config and ensure they are ready for startup;
* `start` -- start all components;
* `stop` -- stop all components;
* `clean` -- cleanup EDEN artifacts (images and certificates) for current context.
If you want to clean artifacts of all contexts, you should to run `clean --current-context=false`;
* `test` -- run tests;
* `info` -- displays Info records, accepts regular expression as a filter;
* `log` -- displays Log records, accepts regular expression as a filter;
* `metric` -- displays Metric records, accepts regular expression as a filter;
* `eve` -- sub-commands for interact with EVE;
* `controller` -- sub-commands to update EVE;
* `pod` -- work with applications running on EVE (containers and VMs);
* `network` -- sub-commands to work with networks running on EVE;
* `registry` -- sub-commands to work with [registry](docs/registry.md).

## Eden EVE commands

* `eden eve onboard` - onboard EVE that is the current config
* `eden eve reset` - put EVE to the initial state (reset to config) removing all changes made by commands or tests

## Eden utils commands

* `eden utils certs` - generate certificates for Adam and EVE
* `eden utils download eve` - download EVE live image from docker hub
* `eden utils download eve-rootfs` - download EVE rootfs image from docker hub
* `eden utils sd` - get information about EVE from provided SD card
* `eden utils gcp` - sub-commands to work with Google Cloud Platform
* `eden utils export` - sub-command to save certs and configs into tar.gz
* `eden utils import` - sub-commands to load certs and configs into tar.gz
