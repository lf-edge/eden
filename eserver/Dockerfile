FROM lfedge/eve-alpine:6.6.0 AS build
ENV BUILD_PKGS go git openssh-keygen
RUN eve-alpine-deploy.sh

# FIXME bump eve-alpine to alpine 3.14
# hadolint ignore=DL3018
RUN apk --no-cache --repository https://dl-cdn.alpinelinux.org/alpine/v3.14/community add -U --upgrade go && go version

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
