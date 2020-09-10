# Eden

Eden is the simplest way to setup & test [EVE](https://github.com/lf-edge/eve) and [Adam](https://github.com/lf-edge/adam).

Eden is inspired by Kubernetes workflows and CLI

All components of Eden -- the main executable file `eden` and test binaries can be used as stand-alone applications without the need to install and configure the development environment on your computer. Generally, you will only need a text editor to configure the system and create test scenarios and scripts using `eden` and pre-compiled test files. To create more complex tests, you can use standard Go testing machinery.

Eden contains series of integration tests implemented in Golang. Tests are structured as normal Golang tests by using `_test.go` nomenclature and be available for test runs using standard go test framework.

## Install Prerequisites

Install requirements from [eve](https://github.com/lf-edge/eve#install-dependencies)

Also, you need to install `telnet` and `squashfs-tools` (`squashfs` for Mac OS X).

You need to be able to run docker commands and able to access virtualization accelerators (KVM on Linux or machyve on Mac OS X)

## Quickstart

```
git clone https://github.com/lf-edge/eden.git
cd eden
make build
eden setup
eden start
eden status
eden test
```

Note for cloud and VM users: eden and eve use virtualization. To run in VM-based environments, see [the cloud document](./docs/cloud.md).

To find out what is running and where:

```
eden status
```

## Eden Config

Eden's config is controlled via a yaml file, overriddable using command-line options. In most cases, the defaults will work just fine for you.
For more informaton, see [docs/config.md](./docs/config.md).

## Remote access to eve

The main way to get a shell, especially once the device is fully registered to its `adam` controller, is via ssh:

```
eden eve ssh
```

If you need access to the actual device console, run:

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

### Deploy Applications

You can deploy images from different sources. In addition, eden deploys its own http server and docker registry.
Each source has its default format, but you can override it with the `--format` flag. The defaults are:

* `docker://` - OCI image
* `http://` - VM qcow2
* `https://` - VM qcow2 image
* `file://` - VM qcow2 image

#### Docker Image

Deploy nginx server from dockerhub. Expose port 80 of the container to port 8028 of eve.

```
eden pod deploy -p 8028:80 docker://nginx
```

#### Docker Image from Local Registry

eden starts a local registry image, running on the localhost at port `5000` (configurable). You can start a docker image
from that registry by passing it the `--registry=local` option. The default for `--registry` is the hostname given
by the registry. For example, `docker://docker.io/library/nginx` defaults to `docker.io`. But if you pass it
the `--registry=local` option, it will try to get it from the local, eden-managed registry.

```console
eden pod deploy --registry=local docker://nginx
```

If the image is not available in the local registry, it will fail. You either can load it up (see the next section),
or ask it to load and deploy:

```console
eden pod deploy --registry=local --load=true docker://nginx
```

#### Loading Local Registry

You can load the local registry with images.

```console
eden registry load docker://docker.io/library/nginx
```

eden will do the following:

1. Check if the image is in your local docker image cache, i.e. `docker image ls`. If it is, load it into the local registry and done.
1. If it is not, try to pull it from the remote registry via `docker pull`. Once that is done, it will load it into the local registry.

#### VM Image

Deploy a VM from Openstack. Initialize root user with password - 'passw0rd'  Expose port 22 of the VM (ssh) to port 8027 of eve for ssh:

```
eden pod deploy -p 8027:22 http://cdimage.debian.org/cdimage/openstack/current/debian-10.4.3-20200610-openstack-amd64.qcow2 -v debug --metadata='#cloud-config\npassword: passw0rd\nchpasswd: { expire: False }\nssh_pwauth: True\n'
```

Deploy a VM from a local file. This will cause the local file to be uploaded to the eden-deployed `eserver`, which, in turn, will deploy it to eve:

```
eden pod deploy file:///path/to/some.img
```

#### VM Image from Docker Registry

Deploy a VM that is in a docker image, whether in OCI Artifacts format, or wrapped in a container. All formats from
[edge-containers](https://github.com/lf-edge/edge-containers) are supported.

```
eden pod deploy docker://some/image:container-tag --format=qcow2
```

### List Applications

List running applications and their ip/ports

```
eden pod ps
```

## Different EVE Builds

eden runs EVE using a `live.img` file with an embedded config partition, which eve configures. In a normal
run, eden takes care of getting and installing an eve OS disk image for you. However, you can configure eden
to use a different image. This is particularly useful for eden's primary use case: testing EVE.

To work with custom EVE images, see [docs/eve-images.md](./docs/eve-images.md).

## Tests

To run tests make sure you called `make build`.

The easy way to run tests is to call `eden test <test folder>`

For example -- run reboot test:

`eden test tests/reboot/`

Some tests may accept parameters - run Log/Metrics/Info test in debug mode with  timeout of 600 seconds  and requiring  3  messages of each type  

`eden test tests/lim/ -v debug -a '-timewait 600 -number 3'`

You can find more detailed information about `eden test` in [tests/README.md](tests/README.md) and [tests/escript/README.md](tests/escript/README.md)

## Google Cloud support
Eden is enough to deploy Eve on Google Cloud. We are going to make it one command, but now the process is: 
* Make an image 
* Upload it to GCP and run 
* start eden (if not started yet) 
* Onboard eve (use `eden eve onboard`)

Step 1 : Make an image. You need to specify IP of Adam, by defalut Adam is a container inside machine with Eden, so put there an IP that is accessible from gcp
```
make CONFIG='--devmodel GCP' build
./eden config set default --key adam.eve-ip --value <IP of Adam/Eden>  
./eden setup
```
Step 2 : Upload image to gcp and run it. You will need a google service key json. https://cloud.google.com/iam/docs/creating-managing-service-account-keys 
```
./eden utils gcp image -k  <PATH TO SERVICE KEY FILE>  upload <PATH TO EVE IMAGE>
./eden utils gcp vm -k <PATH TO SERVICE KEY FILE> 
```
vm is named eden-gcp-test
`eden utils gcp` supports vm-name image-name and bucket-name  parameters

Step 3 : Start eden and onboard Eve
```
./eden start
./eden eve onboard
```


## Raspberry Pi 4 support

Eden is the only thing you need to work with Raspberry and deploy containers there:

Step 1: Install EVE on Raspberry instead of any other OS.

Prepare Raspberry image

```
git clone https://github.com/lf-edge/eden.git
cd eden
eden config add default --devmodel RPi4
eden setup
eden start
```

Then you will have an .img that can be transfered to SD card.
https://www.raspberrypi.org/documentation/installation/installing-images/

For example for MacOS:

```
diskutil list
diskutil unmountDisk /dev/diskN
sudo dd bs=1m if=path_of_your_image.img of=/dev/rdiskN; sync
sudo diskutil eject /dev/rdiskN
```

Put in SD card into Raspberry and power it on

Step 2: Connect to  Raspberry and run some app.

```
eden eve onboard
eden pod deploy -p 8028:80 docker://nginx
```

After these lines you will have nginx available on public EVE IP at port 8028

Use
```
eden status
eden pod ps
```
to get the status of the deployment

### Obtain information about EVE on SD card

To get information from SD card about previously flashed instance of EVE you should:

Step 0 (for MacOS): Install required packets after installing of [brew](https://brew.sh/):

```
brew cask install osxfuse
brew install ext4fuse squashfuse
```

Step 1: List partitions.

For MacOS `diskutil list`

For Linux `lsblk`

You should find your SD and 5 partitions on it.

Step 2: Mount partitions of SD card on your PC

For MacOS (`diskN` is SD card from step 1)

```
sudo umount /dev/diskN*
sudo squashfuse /dev/diskNs2 ~/tmp/rootfs -o allow_other
sudo ext4fuse /dev/diskNs9 ~/tmp/persist -o allow_other
```

For Linux (`sdN` is SD card from step 1)

```
mkdir -p ~/tmp/rootfs ~/tmp/persist
sudo umount /dev/sdN*
sudo mount /dev/sdN2 ~/tmp/rootfs
sudo mount /dev/sdN9 ~/tmp/persist
```

Step 3: Extract files and save them:
- syslog.txt contains logs of EVE: `sudo cp ~/tmp/persist/rsyslog/syslog.txt ~/syslog.txt`
- eve-release contains version of EVE: `sudo cp ~/tmp/rootfs/etc/eve-release ~/eve-release`

Step 4: Umount and eject SD

For MacOS

```
sudo umount /dev/diskN*
sudo diskutil eject /dev/diskN
```
For Linux
```
sudo umount /dev/sdN*
sudo eject /dev/sdN
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
make bin
```

To build `eden` and tests inside eden
It's better to call `eden config add` first, so the build command can build tests for the desired architecture

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
   * `network` -- sub-commands to work with networks running on EVE

## Eden EVE commands

    `eden eve onboard` - onboard EVE that is the current config
    `eden eve reset` - put EVE to the initial state (reset to config) removing all changes made by commands or tests

## Eden utils commands

    `eden utils certs` - generate certificates for Adam and EVE
    `eden utils download eve` - download EVE live image from docker hub
    `eden utils download eve-rootfs` - download EVE rootfs image from docker hub
    `eden utils sd` - get information about EVE from provided SD card
    `eden utils gcp` - sub-commands to work with Google Cloud Platform
   
