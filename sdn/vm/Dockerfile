FROM lfedge/eve-alpine:12.1.0 AS build

ENV BUILD_PKGS="git gcc go make wget libc-dev linux-headers"
ENV PKGS="bash iptables ip6tables iproute2 dhcpcd ipset curl radvd ethtool jq tcpdump \
         strace openssh-client openssh-server vim ca-certificates"
RUN eve-alpine-deploy.sh

ARG DEV=n
RUN if [ ${DEV} = "y" ]; then \
    CGO_ENABLED=0 go get -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@v1.8.3 && \
    cp /root/go/bin/dlv /out/bin; \
fi

ARG DNSMASQ_VERSION=2.86
RUN mkdir -p /dnsmasq
WORKDIR /dnsmasq
RUN wget https://thekelleys.org.uk/dnsmasq/dnsmasq-${DNSMASQ_VERSION}.tar.gz && \
    tar xvzf dnsmasq-${DNSMASQ_VERSION}.tar.gz && \
    cd dnsmasq-${DNSMASQ_VERSION} && \
    make && make install DESTDIR=/out

#TODO: uncomment once SDN-Agent supports NTP server endpoints
#ARG NTP_VERSION=4.2
#ARG NTP_VERSION_PATCH=8p15
#RUN mkdir -p /ntp
#WORKDIR /ntp
#RUN wget https://archive.ntp.org/ntp4/ntp-${NTP_VERSION}/ntp-${NTP_VERSION}.${NTP_VERSION_PATCH}.tar.gz && \
#    tar xvzf ntp-${NTP_VERSION}.${NTP_VERSION_PATCH}.tar.gz && \
#    cd ntp-${NTP_VERSION}.${NTP_VERSION_PATCH} && \
#    ./configure --prefix=/out && \
#    make && make install

WORKDIR /out
RUN mkdir -p ./etc/ssh ./root/.ssh && chmod 0700 ./root/.ssh
COPY cert/ssh/id_rsa* ./root/.ssh/
COPY cert/ssh/id_rsa.pub ./root/.ssh/authorized_keys
RUN chmod 600 ./root/.ssh/id_rsa ./root/.ssh/authorized_keys
RUN cp /etc/passwd /etc/group ./etc/

ADD ./  /sdn/
WORKDIR /sdn
RUN cp scripts/*sh /out/bin/

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN echo "Downloading dependecies" && go mod download && \
    echo "Running go vet" && go vet ./... && \
    echo "Running go fmt" && ERR=$(gofmt -e -l -s $(find . -name \*.go | grep -v /vendor/)) && \
       if [ -n "$ERR" ] ; then echo "go fmt Failed - ERR: "$ERR ; exit 1 ; fi && \
    go build -ldflags "-s -w" -o /out/bin ./cmd/sdnagent/... && \
    go build -ldflags "-s -w" -o /out/bin ./cmd/dns64proxy/... && \
    go build -ldflags "-s -w" -o /out/bin ./cmd/httpsrv/... && \
    go build -ldflags "-s -w" -o /out/bin ./cmd/goproxy/... && \
    go build -ldflags "-s -w" -o /out/bin ./cmd/netbootsrv/... && \
    go build -ldflags "-s -w" -o /out/bin ./cmd/conntrack/...

FROM scratch
COPY --from=build /out/ /

RUN sed -i 's/#*PubkeyAuthentication yes/PubkeyAuthentication yes/g' /etc/ssh/sshd_config && \
    sed -i 's/#*PasswordAuthentication yes/PasswordAuthentication no/g' /etc/ssh/sshd_config && \
    sed -i 's/AllowTcpForwarding no/AllowTcpForwarding yes/g' /etc/ssh/sshd_config

RUN chown -R root:root /root/.ssh/

SHELL ["/bin/sh", "-c"]
WORKDIR /
CMD ["init.sh"]
