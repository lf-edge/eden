# EVE deployment in EDEN

## Qemu deployment

This is default deployment
Once you run `eden config add default` eden generates config with this
type of deployment. Also, it generates `<name>-qemu.conf` in `~/.eden/`
with qemu settings (where `<name>` is your context name).

This installation type requires qemu package installed. Eve runs
in qemu on the same machine. This is useful for debugging and getting
started, but not that useful in production.

Note that KVM requires that the virtual machine host's processor
has virtualization support.

Qemu instance may be started with vTPM device to fully support EVE-OS
security considerations. To use vTPM please ensure that
[swtpm](https://github.com/stefanberger/swtpm/wiki) package is built/installed
in you system and configure eden with
`eden config set default --key eve.tpm --value true`.

## GCP deployment

This deployment type is activated  by flag `--devmodel GCP`
like `make CONFIG='--devmodel GCP' run`
The line above prepares the image for GCP. Use `eden utils gcp`
to upload it to Google cloud

## Raspberry 4 deployment

This deployment type is activated  by flag `--devmodel RPi`
like `make CONFIG='--devmodel RPi' run
The line above makes an image for Raspberry. Use simple SD card burner
to burn the image and put the card into Raspberry and power it up.

Each deployment requires to run

```console
eden setup
eden start
eden eve onboard
```

## File to overwrite model settings

Default properties of devmodel may be overwritten with values provided in [files](../models/README.md).
To use it provide flag `--devmodel-file <file>`
like `make CONFIG='--devmodel-file <file>' run` or `eden config add --devmodel-file <file>`.
To change it on fly set config `eve.devmodelfile` and run `eden eve reset`.
