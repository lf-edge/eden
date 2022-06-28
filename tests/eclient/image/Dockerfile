FROM golang:1.16-alpine AS build

# hadolint ignore=DL3018
RUN apk add --no-cache \
    musl-dev \
    automake \
    make \
    gcc \
    patch

COPY patches /patches

WORKDIR /go/src/local_manager
COPY pkg .

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/root/local_manager main.go

ARG DHCPING_VERSION=1.2
WORKDIR /
RUN wget -q "https://www.mavetju.org/download/dhcping-${DHCPING_VERSION}.tar.gz" && \
    tar -xvf dhcping-${DHCPING_VERSION}.tar.gz
WORKDIR /dhcping-${DHCPING_VERSION}
RUN set -e && for patch in "/patches/dhcping-${DHCPING_VERSION}"/*.patch; do \
        echo "Applying $patch"; \
        patch -p1 < "$patch"; \
    done
RUN case $(uname -m) in \
    x86_64) \
        ./configure --prefix=/usr; \
        ;; \
    aarch64) \
        ./configure --build=arm64-unknown-linux-gnu --prefix=/usr; \
        ;; \
    esac  && \
    make && make DESTDIR=/out install

COPY files /out/
COPY cert/id_rsa* /out/root/.ssh/
COPY cert/id_rsa.pub /out/root/.ssh/authorized_keys

FROM alpine:3.14.6

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
    lsblk

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
