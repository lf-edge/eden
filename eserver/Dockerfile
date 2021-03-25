FROM lfedge/eve-alpine:6.2.0 as build
ENV BUILD_PKGS git go openssh-keygen
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
ARG GOARCH=amd64

RUN go build -ldflags "-s -w" -o /eserver/bin/eserver main.go

FROM alpine:3.13

COPY --from=build /eserver/bin/eserver /bin/
COPY --from=build /root/.ssh/ /root/.ssh/
WORKDIR /eserver
ENTRYPOINT ["/bin/eserver"]
