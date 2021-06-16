# Applications

## Deployment

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
      --vnc-display uint32    display number for VNC pod (0 - no VNC)
      --vnc-password string   VNC password (empty - no password)
      --volume-size string    volume size (default "200 MiB")
      --volume-type string    volume type for empty volumes (qcow2, raw, qcow, vmdk, vhdx or oci); set it to none to not use volumes (default "qcow2")

Global Flags:
      --config string      Name of config (default "default")
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic (default "info")
```

### Mount

You can pass additional volumes to application with `--mount` flag. For example:

```bash
eden pod deploy docker://itmoeve/eclient:0.7 -p 8027:22 --mount=src=docker://nginx:1.20.0,dst=/tst --volume=src=./tests,dst=/dir
```

The command above will deploy eclient image with rootfs of `nginx` mounted to `/tst` and local directory `./tests` mounted to `/dir`.
Note: if directory contains `Dockerfile` the command will use it to build image instead of just copying of all files.

## Modification

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

## Volume management

To see volumes you can run `eden volume ls` to output the list like below:

```console
NAME                    UUID                                    REF                     IMAGE                   TYPE            SIZE    MAX_SIZE        MOUNT   STATE(ADAM)     LAST_STATE(EVE)
eclient-mount_0_m_0     1784916f-b0dc-4d94-b29e-e954741c9d8a    app: eclient-mount      itmoeve/eclient:0.7     CONTAINER       9.4 kB  -               /       IN_CONFIG       DELIVERED
eclient-mount_1_m_0     0b5fda69-680f-4780-8439-ed8e1104a15f    app: eclient-mount      library/nginx:1.20.0    CONTAINER       7.8 kB  -               /tst    IN_CONFIG       DELIVERED
```

If you want to detach the volume from app you can run `eden volume detach <volume name>`. Where `<volume name>`
is the volume from list.
To attach volume you can run `attach <volume name> <app name> [mount point]`. Where `<volume name>`
is the volume from list, `<app name>` - name of application you want to attach the volume, `[mount point]` - the
mount point of volume attached to the app (may be omitted).
