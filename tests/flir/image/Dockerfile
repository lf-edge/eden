FROM ubuntu:18.04

ARG ARCHITECTURE="x86_64"
ARG FLEDGEVERSION="1.8.2"
ARG OPERATINGSYSTEM="ubuntu1804"
ARG FLEDGELINK="http://archives.fledge-iot.org/${FLEDGEVERSION}/${OPERATINGSYSTEM}/${ARCHITECTURE}/"

RUN apt-get update && \
    apt-get install --no-install-recommends -y \
    rsyslog=8.32.0-1ubuntu4 \
    curl=7.58.0-2ubuntu3.10 \
    wget=1.19.4-1ubuntu2.2 \
    jq=1.5+dfsg-2 \
    procps=2:3.3.12-3ubuntu1.2 \
    python3-wheel=0.30.0-0.2 \
    automake=1:1.15.1-3ubuntu2 \
    make=4.1-9.1ubuntu1 && \
    wget ${FLEDGELINK}/fledge-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-gui-${FLEDGEVERSION}.deb && \
    wget ${FLEDGELINK}/fledge-south-modbus-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-south-flirax8-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-south-sinusoid-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-south-opcua-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-south-http-south-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-filter-flirvalidity-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-filter-asset-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-service-notification-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-rule-average-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-rule-outofbound-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-rule-simple-expression-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-notify-asset-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-notify-email-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-north-gcp-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    wget ${FLEDGELINK}/fledge-north-httpc-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    echo '==============================================' && \
    dpkg-deb -R fledge-${FLEDGEVERSION}-${ARCHITECTURE}.deb fledge-${FLEDGEVERSION}-${ARCHITECTURE} && \
    sed -i 's/systemctl/echo/g' ./fledge-${FLEDGEVERSION}-${ARCHITECTURE}/DEBIAN/postinst && \
    dpkg-deb -b fledge-${FLEDGEVERSION}-${ARCHITECTURE} fledge-${FLEDGEVERSION}-${ARCHITECTURE}.deb && \
    dpkg-deb -R fledge-gui-${FLEDGEVERSION}.deb fledge-gui-${FLEDGEVERSION} && \
    sed -i 's/service/echo/g' ./fledge-gui-${FLEDGEVERSION}/DEBIAN/preinst && \
    sed -i 's/service/echo/g' ./fledge-gui-${FLEDGEVERSION}/DEBIAN/postinst && \
    sed -i 's/grep/echo/g' ./fledge-gui-${FLEDGEVERSION}/DEBIAN/postinst && \
    sed -i 's/service/echo/g' ./fledge-gui-${FLEDGEVERSION}/DEBIAN/postrm && \
    dpkg-deb -b fledge-gui-${FLEDGEVERSION} fledge-gui-${FLEDGEVERSION}.deb && \
    echo '==============================================' && \
    DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y ./*.deb && \
    rm -rf ./*deb

RUN echo "service rsyslog start" > start.sh && \
    echo "/etc/init.d/nginx start" >> start.sh && \
    echo "/usr/local/fledge/bin/fledge start" >> start.sh && \
    echo "tail -f /dev/null" >> start.sh && \
    chmod +x start.sh

ENV FLEDGE_ROOT=/usr/local/fledge

# Fledge API and Fledge GUI ports
EXPOSE 8081 1995 80 6683


CMD ["bash", "./start.sh"]

