FROM golang:1.16-alpine AS build

WORKDIR /go/src/local_manager
COPY pkg .

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/root/local_manager main.go

COPY files /out/
COPY cert/id_rsa* /out/root/.ssh/
COPY cert/id_rsa.pub /out/root/.ssh/authorized_keys

FROM alpine:3.14.6

# use patched version of dhcping 1.2-r2+
# hadolint ignore=DL3018
RUN apk add --no-cache lshw \
    curl \
    nginx \
    iproute2 \
    mysql-client \
    netcat-openbsd \
    net-tools \
    openssh \
    jq \
    setserial \
    avahi \
    lsblk && \
    apk --no-cache --repository https://dl-cdn.alpinelinux.org/alpine/edge/community add -U --upgrade dhcping

COPY --from=build /out /

SHELL ["/bin/ash", "-eo", "pipefail", "-c"]
RUN mkdir -p /mnt && \
    touch /mnt/profile && \
    chown root:root /root/.ssh/ && \
    chmod go-w /root && \
    chmod 600 /root/.ssh/id_rsa* && \
    mkdir /var/run/sshd && \
    mkdir -p /var/www/html && \
    echo 'root:adam&eve' | chpasswd && \
    sed -i 's/#*PermitRootLogin prohibit-password/PermitRootLogin yes/g' /etc/ssh/sshd_config && \
    sed -i 's/#*PubkeyAuthentication yes/PubkeyAuthentication yes/g' /etc/ssh/sshd_config && \
    sed -i 's/#*PasswordAuthentication yes/PasswordAuthentication yes/g' /etc/ssh/sshd_config && \
    sed -i 's/#enable-dbus=yes/enable-dbus=no/g' /etc/avahi/avahi-daemon.conf

EXPOSE 22
EXPOSE 80
CMD ["/entrypoint.sh"]
