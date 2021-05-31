FROM lfedge/eve-alpine:6.6.0 as build
ENV BUILD_PKGS git
ENV PKGS perl gawk git
RUN eve-alpine-deploy.sh

WORKDIR /out
RUN git clone --single-branch https://github.com/brendangregg/FlameGraph FlameGraph
COPY entrypoint.sh ./

FROM scratch
COPY --from=build /out/ /
WORKDIR /FlameGraph
ENTRYPOINT ["/entrypoint.sh"]
