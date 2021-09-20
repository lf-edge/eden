FROM lfedge/eve-alpine:6.6.0 AS build
ENV BUILD_PKGS go git openssh-keygen
RUN eve-alpine-deploy.sh

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN mkdir -p /src
WORKDIR /src
COPY pkg .

ARG GOOS=linux

RUN go build -ldflags "-s -w" -o /out/local_manager main.go

FROM ubuntu:focal

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install --no-install-recommends -y \
  busybox \
  lshw \
  curl \
  iproute2 \
  iputils-ping \
  mariadb-client \
  net-tools \
  netcat \
  openssh-server \
  jq \
  setserial \
  avahi-daemon \
  nginx-light \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN mkdir /var/run/sshd && \
    echo 'root:adam&eve' | chpasswd && \
    sed -i 's/#*PermitRootLogin prohibit-password/PermitRootLogin yes/g' /etc/ssh/sshd_config && \
    sed -i 's@session\s*required\s*pam_loginuid.so@session optional pam_loginuid.so@g' /etc/pam.d/sshd

ENV NOTVISIBLE="in users profile"
RUN echo "export VISIBLE=now" >> /etc/profile

COPY cert/id_rsa* /root/.ssh/
COPY cert/id_rsa.pub /root/.ssh/authorized_keys

RUN mkdir -p /mnt && \
    touch /mnt/profile && \
    chown root:root /root/.ssh/ && \
    chmod 600 /root/.ssh/id_rsa*
COPY portmap_test.sh /root/portmap_test.sh
COPY get_eve_ip.sh /root/get_eve_ip.sh
COPY entrypoint.sh /entrypoint.sh
COPY avahi.http.service /etc/avahi/services/http.service
COPY reboot /sbin/reboot
COPY halt /sbin/halt
COPY --from=build /out/local_manager /root/

EXPOSE 22
EXPOSE 80
CMD ["/entrypoint.sh"]
