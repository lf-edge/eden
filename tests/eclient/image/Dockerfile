FROM ubuntu:focal-20210119

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install --no-install-recommends -y \
  curl=7.68.0-1ubuntu2.4 \
  iproute2=5.5.0-1ubuntu1 \
  iputils-ping=3:20190709-3 \
  mariadb-client=1:10.3.22-1ubuntu1 \
  net-tools=1.60+git20180626.aebd88e-1ubuntu1 \
  netcat=1.206-1ubuntu1 \
  openssh-server=1:8.2p1-4ubuntu0.1 \
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
COPY cert/id_rsa* /root/.ssh/
COPY cert/id_rsa.pub /root/.ssh/authorized_keys
RUN chown root:root /root/.ssh/
RUN chmod 600 /root/.ssh/id_rsa*
COPY entrypoint.sh /entrypoint.sh

EXPOSE 22
CMD ["/entrypoint.sh"]
