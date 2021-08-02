FROM ubuntu:focal

ENV DEBIAN_FRONTEND=noninteractive
ENV PHORONIX_VERSION=9.8.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    build-essential \
    autoconf \
    apt-utils \
    wget \
    unzip \
    libzip-dev \
    git \
    apt-file \
    nginx \
    mesa-utils \
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
