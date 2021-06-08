# Eclient image v 0.8

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

FROM ubuntu:focal-20210119

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install --no-install-recommends -y \
  busybox=1:1.30.1-4ubuntu6 \
  lshw=02.18.85-0.3ubuntu2 \
  curl=7.68.0-1ubuntu2.5 \
  iproute2=5.5.0-1ubuntu1 \
  iputils-ping=3:20190709-3 \
  mariadb-client=1:10.3.22-1ubuntu1 \
  net-tools=1.60+git20180626.aebd88e-1ubuntu1 \
  netcat=1.206-1ubuntu1 \
  openssh-server=1:8.2p1-4ubuntu0.2 \
  jq=1.6-1ubuntu0.20.04.1 \
  setserial=2.17-52 \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN mkdir /var/run/sshd
RUN echo 'root:adam&eve' | chpasswd
RUN sed -i 's/#*PermitRootLogin prohibit-password/PermitRootLogin yes/g' /etc/ssh/sshd_config

# SSH login fix. Otherwise user is kicked off after login
RUN sed -i 's@session\s*required\s*pam_loginuid.so@session optional pam_loginuid.so@g' /etc/pam.d/sshd

ENV NOTVISIBLE="in users profile"
RUN echo "export VISIBLE=now" >> /etc/profile

RUN mkdir -p /root/.ssh/
RUN mkdir -p /mnt
RUN touch /mnt/profile
COPY cert/id_rsa* /root/.ssh/
COPY cert/id_rsa.pub /root/.ssh/authorized_keys
RUN chown root:root /root/.ssh/
RUN chmod 600 /root/.ssh/id_rsa*
COPY portmap_test.sh /root/portmap_test.sh
COPY get_eve_ip.sh /root/get_eve_ip.sh
COPY entrypoint.sh /entrypoint.sh
COPY reboot /sbin/
COPY halt /sbin
COPY --from=build /out/local_manager /root/

EXPOSE 22
CMD ["/entrypoint.sh"]
