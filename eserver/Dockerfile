FROM lfedge/eve-alpine:12.1.0 AS build
ENV BUILD_PKGS go git openssh-keygen
RUN eve-alpine-deploy.sh

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN ssh-keygen -t rsa -q -P "" -f /root/.ssh/id_rsa

RUN mkdir -p /eserver/src && mkdir -p /eserver/bin
WORKDIR /eserver/src
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . /eserver/src

ARG GOOS=linux

RUN go build -ldflags "-s -w" -o /eserver/bin/eserver main.go

WORKDIR /out/root/.ssh
RUN mv /root/.ssh/* .
RUN mv /eserver/bin/eserver /out/bin/

FROM scratch

COPY --from=build /out/ /
WORKDIR /eserver
ENTRYPOINT ["/bin/eserver"]
