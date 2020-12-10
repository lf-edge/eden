FROM golang:1.15-alpine as build
RUN apk --no-cache add make==4.3-r0 qemu-img=5.0.0-r2
ENV CGO_ENABLED=0
WORKDIR /eden
COPY . /eden
RUN make DO_DOCKER=0 build-tests
RUN mkdir /out
RUN cp -rf /eden/eden /eden/tests /eden/dist /eden/docs /eden/README.md /out

FROM alpine:3.12
WORKDIR /
COPY --from=build /out /