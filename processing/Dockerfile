FROM lfedge/eve-alpine:745ae9066273c73b0fd879c4ba4ff626a8392d04 AS build
ENV BUILD_PKGS=git
ENV PKGS="perl gawk git"
RUN eve-alpine-deploy.sh

WORKDIR /out
RUN git clone --single-branch https://github.com/brendangregg/FlameGraph FlameGraph
COPY entrypoint.sh ./

FROM scratch
COPY --from=build /out/ /
WORKDIR /FlameGraph
ENTRYPOINT ["/entrypoint.sh"]
