# Applications

Application, or workload, management on a running EVE device is provided via the
`eden pod` commands. This document describes how to use those commands, and the
options and considerations for deploying applications on an EVE device.

## Command Summary

### Deploy Applications

In order to deploy application you can use `eden pod deploy` command:

```console
Deploy app in pod.

Usage:
  eden pod deploy (docker|http(s)|file|directory)://(<TAG|PATH>[:<VERSION>] | <URL for qcow2 image> | <path to qcow2 image>) [flags]

Flags:
      --acl strings           Allow access only to defined hosts/ips/subnets
                              You can set acl for particular network in format '<network_name:acl>'
                              To remove acls you can set empty line '<network_name>:'
      --adapters strings      adapters to assign to the application instance
      --cpus uint32           cpu number for app (default 1)
      --direct                Use direct download for image instead of eserver (default true)
      --disk-size string      disk size (empty or 0 - same as in image) (default "0 B")
      --disks strings         Additional disks to use. You can write it in notation <link> or <mount point>:<link>. Deprecated. Please use volumes instead.
      --format string         format for image, one of 'container','qcow2','raw','qcow','vmdk','vhdx'; if not provided, defaults to container image for docker and oci transports, qcow2 for file and http/s transports
  -h, --help                  help for deploy
      --memory string         memory for app (default "1.0 GB")
      --metadata string       metadata for pod
      --mount stringArray     Additional volumes to use. You can write it in notation src=<link>,dst=<mount point>.
  -n, --name string           name for pod
      --networks strings      Networks to connect to app (ports will be mapped to first network). May have <name:[MAC address]> notation.
      --no-hyper              Run pod without hypervisor
      --only-host             Allow access only to host and external networks
      --openstack-metadata    Use OpenStack metadata for VM
  -p, --publish strings       Ports to publish in format EXTERNAL_PORT:INTERNAL_PORT
      --registry string       Select registry to use for containers (remote/local) (default "remote")
      --sftp                  Force use of sftp to load http/file image from eserver
      --vnc-display int       display number for VNC pod
      --vnc-password string   VNC password (empty - no password)
      --vnc-for-shim-vm       Enables VNC for a shim VM
      --volume-size string    volume size (default "200 MiB")
      --volume-type string    volume type for empty volumes (qcow2, raw, qcow, vmdk, vhdx or oci); set it to none to not use volumes (default "qcow2")

Global Flags:
      --config string      Name of config (default "default")
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic (default "info")
```

### List Deployed Applications

List running applications, their names, ip/ports

```console
eden pod ps
```

### View Application Logs

To view the logs of an application:

```console
eden pod logs <app_name>
```

You can get the `<app_name>` from the application list.

You can choose which information to display by providing `--fields` flag,
and listing one or more of the following fields, separated by comma:

* `log` - to view log objects
* `info` - to view info objects
* `metric` - to view metric objects
* `netstat` - to view network packages counts
* `app` - to view console output

For example:

```console
eden pod logs app1 --fields log,info,app
```

You can limit output to only the last N lines with the `--tail <N>` flag.

### Delete Application

To delete an application:

```console
eden pod delete <app_name>
```

You can get the `<app_name>` from the application list.

**Warning:** The command will also delete volumes of app. If you want to save
the volumes, pass the `--with-volumes=false` flag.

#### Mount Volumes

You can pass additional volumes to application with `--mount` flag. For example:

```bash
eden pod deploy docker://lfedge/eden-eclient:83cfe07 -p 8027:22 --mount=src=docker://nginx:1.20.0,dst=/tst --volume=src=./tests,dst=/dir
```

The command above will deploy eclient image with rootfs of `nginx` mounted to `/tst` and local directory `./tests` mounted to `/dir`.
Note: if directory contains `Dockerfile` the command will use it to build image instead of just copying of all files.

### Modify Existing Applications

In order to modify existing application you can use `eden pod modify` command:

```console
Modify pod

Usage:
  eden pod modify <app> [flags]

Flags:
      --acl strings        Allow access only to defined hosts/ips/subnets
                           You can set acl for particular network in format '<network_name:acl>'
                           To remove acls you can set empty line '<network_name>:'
  -h, --help               help for modify
      --networks strings   Networks to connect to app (ports will be mapped to first network)
      --only-host          Allow access only to host and external networks
  -p, --publish strings    Ports to publish in format EXTERNAL_PORT:INTERNAL_PORT

Global Flags:
      --config string      Name of config (default "default")
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic (default "info")
```

### Manage Volumes

To see volumes you can run `eden volume ls` to output the list like below:

```console
NAME                    UUID                                    REF                     IMAGE                           TYPE            SIZE    MAX_SIZE        MOUNT   STATE(ADAM)     LAST_STATE(EVE)
eclient-mount_0_m_0     1784916f-b0dc-4d94-b29e-e954741c9d8a    app: eclient-mount      lfedge/eden-eclient:83cfe07     CONTAINER       9.4 kB  -               /       IN_CONFIG       DELIVERED
eclient-mount_1_m_0     0b5fda69-680f-4780-8439-ed8e1104a15f    app: eclient-mount      library/nginx:1.20.0            CONTAINER       7.8 kB  -               /tst    IN_CONFIG       DELIVERED
```

If you want to detach the volume from app you can run `eden volume detach <volume name>`. Where `<volume name>`
is the volume from list.
To attach volume you can run `attach <volume name> <app name> [mount point]`. Where `<volume name>`
is the volume from list, `<app name>` - name of application you want to attach the volume, `[mount point]` - the
mount point of volume attached to the app (may be omitted).

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

## Application Deployment Details

EVE can load and run application images from different sources. In addition,
eden deploys its own http server and docker registry, to simplify presenting
local files and container images to a running EVE device.

Each source has its default format, but you can override it with
the `--format` flag (`container`,`qcow2` or `raw`). The defaults are:

* `docker://` - OCI image from OCI registry
* `http://` - VM qcow2 from http endpoint
* `https://` - VM qcow2 image from https endpoint
* `file://` - VM qcow2 image from local file; eden loads it into eserver and then serves it over http

You can also pass additional disks for your app with `--disks` flag.

### Docker Image

Deploy nginx server from dockerhub. Expose port 80 of the container
to port 8028 of eve.

```console
eden pod deploy -p 8028:80 docker://nginx
```

### Docker Image with volume

If Docker image contains `Volume` annotation inside, Eden will add volumes for every mention of volume.
You can modify behavior with `--volume-type` flag:

* choose type of volume (`qcow2`, `raw` or `oci`)
* skip this action with `none`

### Docker Image from Local Registry

eden starts a local registry image, running on the localhost at port `5050`
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

### Loading Image into Local Registry

You can load the local registry with images.

```console
eden registry load library/nginx
```

eden will do the following:

1. Check if the image is in your local docker image cache, i.e. `docker image ls`.
If it is, load it into the local registry and done.
2. If it is not there, try to pull it from the remote registry via `docker pull`.
Once that is done, it will load it into the local registry.

### VM Image with SSH access

Deploy a VM with Ubuntu 20.10 . Initialize `ubuntu` user with password `passw0rd`.
Expose port 22 of the VM (ssh) to port 8027 of eve for ssh:

```console
eden pod deploy -p 8027:22 https://cloud-images.ubuntu.com/releases/groovy/release-20210108/ubuntu-20.10-server-cloudimg-amd64.img -v debug --metadata='#cloud-config\npassword: passw0rd\nchpasswd: { expire: False }\nssh_pwauth: True\n'
```

You will be able to ssh into the image via EVE-IP and port 8027 and do whatever you'd like to
`ssh ubuntu@EVE-IP -p 8027`

Deploy a VM from a local file. This will cause the local file
to be uploaded to the eden-deployed `eserver`, which, in turn,
will deploy it to eve:

```console
eden pod deploy file:///path/to/some.img
```

### VM Image from Docker Registry

Deploy a VM that is in a docker image, whether in OCI Artifacts format,
or wrapped in a container. All formats from
[edge-containers](https://github.com/lf-edge/edge-containers) are supported.

```console
eden pod deploy docker://some/image:container-tag --format=qcow2
```

### Deal with multiple network interfaces. Expose the pod on a specific network

Eve is listening on all interfaces connected. Docker/VM can only be exposed on one. By default it's the first interface (eth0). If you want to expose on the selected interface you need to set up a network and then use this network upon the deploy.

Here eth1 is added to network n2 and then a pod is exposed on this network.

```console
eden network create 10.11.13.0/24 -n n2 --uplink eth1
eden pod deploy -p 8028:80 --networks n2 docker://nginx
```

### Edit forwarded ports of Applications

To modify port forward you can run `eden pod modify <app name> -p <new port forward>` command.

For example for `laughing_maxwell` app name and forwarding of 8028<->80 TCP port you can run:

```console
eden pod modify laughing_maxwell -p 8028:80
```
