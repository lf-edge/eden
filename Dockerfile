FROM lfedge/eve-alpine:6.6.0 as build
ENV BUILD_PKGS go make qemu-img
RUN eve-alpine-deploy.sh

ARG OS=linux
ENV CGO_ENABLED=0
WORKDIR /eden
COPY . /eden
RUN make DO_DOCKER=0 OS=${OS} build-tests
RUN cp -rf /eden/eden /eden/tests /eden/dist /eden/docs /eden/README.md /out

FROM scratch
WORKDIR /
COPY --from=build /out/ /
