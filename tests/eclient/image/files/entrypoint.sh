#!/bin/sh

#save envs to debug them
env >/run/envs

# configurable with envs
if [ -z "$AVAHI_HOST_NAME" ]; then AVAHI_HOST_NAME=ubuntu-http-server; fi
if [ -z "$AVAHI_DOMAIN_NAME" ]; then AVAHI_DOMAIN_NAME=local; fi

sed -i "s/^#host-name=.*/host-name=$AVAHI_HOST_NAME/" /etc/avahi/avahi-daemon.conf
sed -i "s/^host-name=.*/host-name=$AVAHI_HOST_NAME/" /etc/avahi/avahi-daemon.conf
sed -i "s/^#domain-name=.*/domain-name=$AVAHI_DOMAIN_NAME/" /etc/avahi/avahi-daemon.conf
sed -i "s/^domain-name=.*/domain-name=$AVAHI_DOMAIN_NAME/" /etc/avahi/avahi-daemon.conf
sed -i 's/^publish-workstation=no/publish-workstation=yes/' /etc/avahi/avahi-daemon.conf
sed -i 's/^use-ipv6=yes/use-ipv6=no/' /etc/avahi/avahi-daemon.conf

# we must re-generate them now on every boot
rm -rf /run/*
mkdir -p /run/sshd
mkdir -p /run/nginx

nginx

/usr/sbin/sshd -h /root/.ssh/id_rsa

avahi-daemon
