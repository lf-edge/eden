FROM ubuntu:focal

ENV DEBIAN_FRONTEND=noninteractive
ENV PHORONIX_VERSION=9.8.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    build-essential=12.8ubuntu1 \
    autoconf=2.69-11.1 \
    apt-utils=2.0.2ubuntu0.1 \
    wget=1.20.3-1ubuntu1 \
    unzip=6.0-25ubuntu1 \
    libzip-dev=1.5.1-0ubuntu1 \
    git=1:2.25.1-1ubuntu3 \
    apt-file=3.2.2 \
    nginx=1.18.0-0ubuntu1 \
    mesa-utils=8.4.0-1build1 \
    && wget http://phoronix-test-suite.com/releases/repo/pts.debian/files/phoronix-test-suite_${PHORONIX_VERSION}_all.deb \
    && apt-get install -y --no-install-recommends ./phoronix-test-suite_${PHORONIX_VERSION}_all.deb \
    && rm -f phoronix-test-suite_${PHORONIX_VERSION}_all.deb
WORKDIR /
COPY entrypoint.sh /entrypoint.sh
RUN chmod a+x /entrypoint.sh
COPY defs/ /var/lib/phoronix-test-suite/test-suites/local/
EXPOSE 80
VOLUME ["/data"]
ENTRYPOINT ["/bin/bash"]
CMD ["/entrypoint.sh"]
