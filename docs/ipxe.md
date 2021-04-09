# IPXE boot

You can ipxe to boot and install EVE onto your hardware. The booting process is
described [here](https://github.com/lf-edge/eve/blob/master/docs/BOOTING.md).

To make an image supporting EDEN, you need to have IP, which is resolvable by your hardware
(actually, you need to forward port of eden-http-server which one is `8888/tcp` by default
and Adam which one is `3333/tcp` by default).

Next, do the following (where `IP` below is the ip address of EDEN for access from EVE):

```bash
eden config add default --devmodel=general
eden config set default --key adam.eve-ip --value IP
eden setup --netboot=true
```

You will see in the output something like
`Please use /home/user/eden/dist/default-images/eve/tftp/ipxe.efi.cfg to boot your EVE via ipxe`.
You should add this file into your tftp server and point yours
dhcp option [67 Bootfile-Name](https://tools.ietf.org/html/rfc2132#section-9.5) onto it.

You can also use `ipxe.efi.cfg` uploaded into eserver, link to which will also be inside
output of setup command (something like
`ipxe.efi.cfg uploaded to eserver (http://IP:8888/eserver/ipxe.efi.cfg).`).

You can start your device and wait for installation process of EVE. Next, you can run
`eden start` and `eden eve onboard` as usual.
