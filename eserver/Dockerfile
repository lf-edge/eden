FROM golang:1.12.5-alpine3.9 AS build

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN apk --no-cache add git=2.20.4-r0

RUN mkdir -p /eserver/src && mkdir -p /eserver/bin
WORKDIR /eserver/src
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . /eserver/src

ARG GOOS=linux
ARG GOARCH=amd64

RUN go build -o /eserver/bin/eserver main.go

FROM alpine:3.11

COPY --from=build /eserver/bin/eserver /bin/
WORKDIR /eserver
ENTRYPOINT ["/bin/eserver"]
