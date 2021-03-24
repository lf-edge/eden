FROM lfedge/eve-alpine:8a53b741d47f42752db30dd896c3467433bac152 as build
ENV BUILD_PKGS perl git gawk
RUN eve-alpine-deploy.sh

WORKDIR /out
RUN git clone --single-branch https://github.com/brendangregg/FlameGraph FlameGraph
COPY entrypoint.sh ./

FROM scratch
COPY --from=build /out/ /
WORKDIR /FlameGraph
ENTRYPOINT ["/entrypoint.sh"]
