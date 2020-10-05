# Registry

Eden deploys official docker registry [image](https://hub.docker.com/_/registry) during setup. You can use the registry to deploy applications to the EVE bypassing the public hub.
This also makes it possible to reuse images for different EVEs.

### General image
To add the general image to the register, you need to run the command:
```
eden registry load <image name>
```
where `<image name>` is `nginx` for example. In this example EDEN will try to use local docker image cache in the first attempt and if it fails, pull from remote to local, and then load.

### Edge container image
To build and add the [edge-container](https://github.com/lf-edge/edge-containers) image you can use the following commands:
```
eden pod publish <image name> --local --root <path to the root disk>:<format of root disk>
```
where `<path to the root disk>` is full path to the file with disk you want to publish as root, `<format of root disk>` is, for example `qcow2`.

### Using an image to launch an application
To use loaded image you just need to add `--registry=local` flag to `pod deploy` command:
```
eden pod deploy docker://nginx --registry=local
``` 

