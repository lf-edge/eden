FROM ubuntu:focal

ENV DEBIAN_FRONTEND=noninteractive
ENV PHORONIX_VERSION=9.8.0

RUN apt-get update \
    && apt-get -y install build-essential autoconf apt-utils wget unzip libzip-dev git-core apt-file nginx \
    && cd /tmp \
    && wget http://phoronix-test-suite.com/releases/repo/pts.debian/files/phoronix-test-suite_${PHORONIX_VERSION}_all.deb \
    && apt install -y ./phoronix-test-suite_${PHORONIX_VERSION}_all.deb \
    && rm -f phoronix-test-suite_${PHORONIX_VERSION}_all.deb \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /
ADD entrypoint.sh /entrypoint.sh
RUN chmod a+x /entrypoint.sh
EXPOSE 80
ENTRYPOINT ["/bin/bash"]
CMD ["/entrypoint.sh"]
