# Applications

## Deployment

In order to deploy application you can use `eden pod deploy` command:

```console
Deploy app in pod.

Usage:
eden pod deploy (docker|http(s)|file)://(<TAG>[:<VERSION>] | <URL for qcow2 image> | <path to qcow2 image>) [flags]

Flags:
--acl strings           Allow access only to defined hosts/ips/subnets
--adapters strings      adapters to assign to the application instance
--cpus uint32           cpu number for app (default 1)
--direct                Use direct download for image instead of eserver (default true)
--disk-size string      disk size (empty or 0 - same as in image) (default "0 B")
--disks strings         Additional disks to use
--format string         format for image, one of 'container','qcow2','raw','qcow','vmdk','vhdx'; if not provided, defaults to container image for docker and oci transports, qcow2 for file and http/s transports
-h, --help                  help for deploy
--memory string         memory for app (default "1.0 GB")
--metadata string       metadata for pod
-n, --name string           name for pod
--networks strings      Networks to connect to app (ports will be mapped to first network)
--no-hyper              Run pod without hypervisor
--only-host             Allow access only to host and external networks
-p, --publish strings       Ports to publish in format EXTERNAL_PORT:INTERNAL_PORT
--registry string       Select registry to use for containers (remote/local) (default "remote")
--sftp                  Force use of sftp to load http/file image from eserver
--vnc-display uint32    display number for VNC pod (0 - no VNC)
--vnc-password string   VNC password (empty - no password)
--volume-type string    volume type for empty volumes (qcow2, raw, qcow, vmdk, vhdx or oci); set it to none to not use volumes (default "qcow2")

Global Flags:
--config string      Name of config (default "default")
-v, --verbosity string   Log level (debug, info, warn, error, fatal, panic (default "info")
```

## Modification

In order to modify existing application you can use `eden pod modify` command:

```console
Modify pod

Usage:
eden pod modify <app> [flags]

Flags:
--acl strings        Allow access only to defined hosts/ips/subnets
-h, --help               help for modify
--networks strings   Networks to connect to app (ports will be mapped to first network)
--only-host          Allow access only to host and external networks
-p, --publish strings    Ports to publish in format EXTERNAL_PORT:INTERNAL_PORT

Global Flags:
--config string      Name of config (default "default")
-v, --verbosity string   Log level (debug, info, warn, error, fatal, panic (default "info")
```
