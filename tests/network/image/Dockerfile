FROM debian:buster-slim
RUN apt-get update && \
  apt-get install -y --no-install-recommends \
  supervisor=3.3.5-1 \
  curl=7.64.0-4+deb10u1 \
  dhcpcd5=7.1.0-2 \
  nginx=1.14.2-2+deb10u3 \
  net-tools=1.60+git20180626.aebd88e-1 && \
  rm -rf /var/lib/apt/lists/*

COPY supervisord.conf /etc/supervisord.conf
COPY entrypoint.sh /entrypoint.sh
COPY dhcpcd.conf /dhcpcd.conf
RUN chmod a+x /entrypoint.sh

EXPOSE 80

ENTRYPOINT ["/entrypoint.sh"]