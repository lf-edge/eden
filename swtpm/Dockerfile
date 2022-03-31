FROM alpine AS build

RUN apk add git autoconf automake libtool gcc g++ openssl-dev build-base make socat gawk libtasn1-dev gnutls gnutls-utils gnutls-dev expect python3 libseccomp-dev json-glib-dev

RUN git clone --branch=v0.9.1 --single-branch --depth=1 https://github.com/stefanberger/libtpms.git
RUN cd libtpms                                  &&\
    ./autogen.sh --prefix=/usr --libdir=/usr/lib --with-tpm2 --with-openssl &&\
    make -j4                                    &&\
    make install


RUN git clone --branch=v0.7.0 --single-branch --depth=1 https://github.com/stefanberger/swtpm.git
RUN cd swtpm                                                &&\
    ./autogen.sh --prefix=/usr --libdir=/usr/lib --with-openssl --with-tss-user=root --with-tss-group=root &&\
    make -j4                                                &&\
    make install

FROM alpine
ENTRYPOINT ["/usr/bin/swtpm"]

RUN apk add --no-cache libseccomp openssl

COPY --from=build /usr /usr
