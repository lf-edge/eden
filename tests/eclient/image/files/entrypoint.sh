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

# eclient can be also used as a router app
IP_FORWARDING="$(sysctl -n net.ipv4.ip_forward)"
if [ "$IP_FORWARDING" -eq 1 ]; then
    # With bare containers (eve.accel=false), IP forwarding is always enabled.
    echo "IP forwarding is already enabled"
else
    echo "Enabling IP forwarding..."
    sysctl -w net.ipv4.ip_forward=1 && echo "IP forwarding is now enabled"
fi

nginx

/usr/sbin/sshd -h /root/.ssh/id_rsa

avahi-daemon -D

# For app_logs test.
echo "Started eclient"

# Running shell as the entrypoint allows to enter the container using
# `eve attach-app-console <console-id>/cons` and have interactive session.
# This is useful when a deployed container cannot be accessed via ssh over the network.
while true; do /bin/sh; done
