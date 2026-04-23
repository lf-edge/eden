# syntax=docker/dockerfile:1
FROM ubuntu:20.04

ARG FLEDGEVERSION="3.1.0"

ENV DEBIAN_FRONTEND=noninteractive \
    FLEDGE_ROOT=/usr/local/fledge

SHELL ["/bin/bash", "-eo", "pipefail", "-c"]

# Install dependencies
RUN apt-get update && apt-get install --no-install-recommends -y curl
RUN apt-get install --no-install-recommends -y \
        sudo git ca-certificates curl gnupg lsb-release \
        libssl-dev avahi-daemon \
        rsyslog nginx postgresql wget jq procps \
        libmodbus-dev libmbedtls-dev libjwt-dev \
        libboost-filesystem-dev libboost-program-options-dev
RUN apt-get remove -y nodejs npm && \
    apt-get autoremove -y && \
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install --no-install-recommends -y nodejs && \
    npm install -g yarn@1.22.22

# Build and install fledge core.
# requirements.sh installs all C++ and Python build dependencies for the core
# (cmake, g++, boost, libpq-dev, sqlite3 custom build, krb5, etc.).
ADD https://github.com/fledge-iot/fledge.git#v${FLEDGEVERSION} /tmp/fledge
WORKDIR /tmp/fledge
RUN ./requirements.sh && \
    env -u FLEDGE_ROOT make -j "$(nproc)" && \
    env -u FLEDGE_ROOT make install && \
    cp -r cmake_build /usr/local/fledge/ && \
    cp -r C /usr/local/fledge/ && \
    rm -rf /tmp/fledge /tmp/sqlite3-pkg

# Build the Free OPC-UA library as a static library.
# fledge-south-opcua's requirements.sh references this Dianomic/Kapsch fork.
# Static build avoids shipping .so files at runtime.
ADD https://github.com/dianomic/freeopcua.git#Kapsch /tmp/freeopcua
WORKDIR /tmp/freeopcua
RUN sed -i 's/SHARED/STATIC/g' CMakeLists.txt && \
    cmake -S . -B build && \
    cmake --build build -j "$(nproc)"

ENV FREEOPCUA=/tmp/freeopcua

# Build and install fledge-south-modbus-c
# (the deb was named fledge-south-modbus; the GitHub repo is fledge-south-modbus-c)
ADD https://github.com/fledge-iot/fledge-south-modbus-c.git#v${FLEDGEVERSION} /tmp/fledge-south-modbus-c
RUN cmake -S /tmp/fledge-south-modbus-c -B /tmp/fledge-south-modbus-c/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-south-modbus-c/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-south-modbus-c

# Build and install fledge-south-FlirAX8 (depends on modbus-c)
# (capitalization matters: FlirAX8, not flirax8)
ADD https://github.com/fledge-iot/fledge-south-FlirAX8.git#v${FLEDGEVERSION} /tmp/fledge-south-FlirAX8
RUN cmake -S /tmp/fledge-south-FlirAX8 -B /tmp/fledge-south-FlirAX8/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-south-FlirAX8/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-south-FlirAX8

# Build and install fledge-south-opcua (uses FREEOPCUA env set above)
ADD https://github.com/fledge-iot/fledge-south-opcua.git#v${FLEDGEVERSION} /tmp/fledge-south-opcua
RUN cmake -S /tmp/fledge-south-opcua -B /tmp/fledge-south-opcua/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-south-opcua/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-south-opcua /tmp/freeopcua

# Build and install fledge-filter-asset
# NOTE: fledge-filter-flirvalidity has no public GitHub repo and is omitted.
ADD https://github.com/fledge-iot/fledge-filter-asset.git#v${FLEDGEVERSION} /tmp/fledge-filter-asset
RUN cmake -S /tmp/fledge-filter-asset -B /tmp/fledge-filter-asset/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-filter-asset/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-filter-asset

# Build and install fledge-service-notification.
# Must be built before rule/notify plugins so its headers are available.
ADD https://github.com/fledge-iot/fledge-service-notification.git#v${FLEDGEVERSION} /tmp/fledge-service-notification
RUN cmake -S /tmp/fledge-service-notification -B /tmp/fledge-service-notification/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-service-notification/build -j "$(nproc)" --target install && \
    cp -r /tmp/fledge-service-notification/C/services/notification/include \
        /usr/local/fledge/notification-service-include && \
    rm -rf /tmp/fledge-service-notification

ENV NOTIFICATION_SERVICE_INCLUDE_DIRS=/usr/local/fledge/notification-service-include

# Build and install notification rule plugins
ADD https://github.com/fledge-iot/fledge-rule-average.git#v${FLEDGEVERSION} /tmp/fledge-rule-average
RUN cmake -S /tmp/fledge-rule-average -B /tmp/fledge-rule-average/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-rule-average/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-rule-average

ADD https://github.com/fledge-iot/fledge-rule-outofbound.git#v${FLEDGEVERSION} /tmp/fledge-rule-outofbound
RUN cmake -S /tmp/fledge-rule-outofbound -B /tmp/fledge-rule-outofbound/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-rule-outofbound/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-rule-outofbound

ADD https://github.com/fledge-iot/fledge-rule-simple-expression.git#v${FLEDGEVERSION} /tmp/fledge-rule-simple-expression
RUN cmake -S /tmp/fledge-rule-simple-expression -B /tmp/fledge-rule-simple-expression/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-rule-simple-expression/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-rule-simple-expression

# Build and install notification delivery plugins
ADD https://github.com/fledge-iot/fledge-notify-asset.git#v${FLEDGEVERSION} /tmp/fledge-notify-asset
RUN cmake -S /tmp/fledge-notify-asset -B /tmp/fledge-notify-asset/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-notify-asset/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-notify-asset

ADD https://github.com/fledge-iot/fledge-notify-email.git#v${FLEDGEVERSION} /tmp/fledge-notify-email
RUN cmake -S /tmp/fledge-notify-email -B /tmp/fledge-notify-email/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-notify-email/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-notify-email

# Build and install north plugins
# (the deb was named fledge-north-httpc; the GitHub repo is fledge-north-http-c)
ADD https://github.com/eclipse/paho.mqtt.c.git#v1.3.16 /paho.mqtt
WORKDIR /paho.mqtt
RUN make PAHO_WITH_SSL=1 && \
    make install && \
    ldconfig

ADD https://github.com/fledge-iot/fledge-north-gcp.git#v2.1.0 /tmp/fledge-north-gcp
RUN cmake -S /tmp/fledge-north-gcp -B /tmp/fledge-north-gcp/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-north-gcp/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-north-gcp

ADD https://github.com/fledge-iot/fledge-north-http-c.git#v${FLEDGEVERSION} /tmp/fledge-north-http-c
RUN cmake -S /tmp/fledge-north-http-c -B /tmp/fledge-north-http-c/build \
          -DFLEDGE_INSTALL="${FLEDGE_ROOT}" && \
    cmake --build /tmp/fledge-north-http-c/build -j "$(nproc)" --target install && \
    rm -rf /tmp/fledge-north-http-c

# Install Python south plugins by copying into the fledge plugin tree
ADD https://github.com/fledge-iot/fledge-south-sinusoid.git#v${FLEDGEVERSION} /tmp/fledge-south-sinusoid
RUN cp -r /tmp/fledge-south-sinusoid/python/fledge/plugins/south/sinusoid \
        "${FLEDGE_ROOT}/python/fledge/plugins/south/" && \
    rm -rf /tmp/fledge-south-sinusoid

# fledge-south-http replaces the discontinued fledge-south-http-south deb package
ADD https://github.com/fledge-iot/fledge-south-http.git#v${FLEDGEVERSION} /tmp/fledge-south-http
RUN cp -r /tmp/fledge-south-http/python/fledge/plugins/south/http_south \
        "${FLEDGE_ROOT}/python/fledge/plugins/south/" && \
    rm -rf /tmp/fledge-south-http

# Build and deploy fledge-gui
ADD https://github.com/fledge-iot/fledge-gui.git#v${FLEDGEVERSION} /tmp/fledge-gui
WORKDIR /tmp/fledge-gui
RUN yarn install && \
    ./build --clean-start && \
    mkdir -p /var/www/html && \
    cp -r dist/* /var/www/html/ && \
    cp nginx.conf /etc/nginx/sites-available/fledge-gui && \
    ln -sf /etc/nginx/sites-available/fledge-gui /etc/nginx/sites-enabled/fledge-gui && \
    yarn cache clean && \
    rm -f /etc/nginx/sites-enabled/default && \
    rm -rf /tmp/fledge-gui /root/.cache /root/.yarn

WORKDIR /
RUN echo "service rsyslog start" > /start.sh && \
    echo "/etc/init.d/nginx start" >> /start.sh && \
    echo "${FLEDGE_ROOT}/bin/fledge start" >> /start.sh && \
    echo "tail -f /dev/null" >> /start.sh && \
    chmod +x /start.sh

ENV FLEDGE_ROOT=/usr/local/fledge

# Fledge API, fledge alternate port, nginx (GUI), and fledge GUI port
EXPOSE 8081 1995 80 6683

CMD ["bash", "/start.sh"]
