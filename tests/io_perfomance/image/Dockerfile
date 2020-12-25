FROM golang:1.15 as builder

WORKDIR /app
COPY mkconfig.go /app
COPY fioconv.go /app
RUN go build fioconv.go
RUN go build mkconfig.go

FROM ubuntu:focal

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    fio=3.16-1 \
    git=1:2.25.1-1ubuntu3 \
    lshw=02.18.85-0.3ubuntu2 \
    sysstat=12.2.0-2

WORKDIR /
COPY run-test.sh /run-test.sh
COPY README.md /README.md
COPY run-iostat.sh /run-iostat.sh
COPY --from=builder /app/fioconv .
COPY --from=builder /app/mkconfig .
RUN chmod a+x /run-iostat.sh
RUN chmod a+x /run-test.sh

VOLUME ["/data"]
ENTRYPOINT ["/bin/bash"]
CMD ["/run-test.sh"]


