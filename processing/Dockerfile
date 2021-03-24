FROM alpine:3.12

RUN apk add --no-cache perl=5.30.3-r0 git=2.26.3-r0 gawk=5.1.0-r0

RUN git clone --single-branch https://github.com/brendangregg/FlameGraph /FlameGraph

COPY entrypoint.sh /

WORKDIR /FlameGraph

ENTRYPOINT ["/entrypoint.sh"]
