FROM golang:1.16-alpine as builder

WORKDIR /app
COPY mkconfig /app/mkconfig
COPY fioconv /app/fioconv
RUN go build -o fioconv ./fioconv/fioconv.go \
    && go build -o mkconfig ./mkconfig/mkconfig.go

FROM ubuntu:focal

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    fio=3.16-1 \
    git=1:2.25.1-1ubuntu3 \
    lshw=02.18.85-0.3ubuntu2 \
    sysstat=12.2.0-2

WORKDIR /
COPY run-test.sh README.md run-iostat.sh ./
COPY --from=builder /app/fioconv /app/mkconfig ./
RUN chmod a+x /run-iostat.sh /run-test.sh

VOLUME ["/data"]
ENTRYPOINT ["/bin/bash"]
CMD ["/run-test.sh"]
