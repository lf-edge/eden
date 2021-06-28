FROM debian:buster-slim
RUN apt-get update && \
  apt-get install -y --no-install-recommends \
  supervisor=3.3.5-1 \
  curl=7.64.0-4+deb10u2 \
  dhcpcd5=7.1.0-2 \
  nginx=1.14.2-2+deb10u4 \
  net-tools=1.60+git20180626.aebd88e-1 \
  dnsmasq=2.80-1+deb10u1 \
  iproute2=4.20.0-2+deb10u1 && \
  rm -rf /var/lib/apt/lists/*

RUN mkdir /app/
COPY supervisord.conf /etc/supervisord.conf
COPY entrypoint.sh /app/entrypoint.sh
COPY dhcpcd.conf /etc/dhcpcd.conf
RUN chmod a+x /app/entrypoint.sh

EXPOSE 80

ENTRYPOINT ["/app/entrypoint.sh"]