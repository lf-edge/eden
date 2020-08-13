FROM debian:buster-slim
RUN apt-get update && apt-get install -y supervisor curl dhcpcd5 nginx net-tools && rm -rf /var/lib/apt/lists/*

COPY supervisord.conf /etc/supervisord.conf
COPY entrypoint.sh /entrypoint.sh
COPY dhcpcd.conf /dhcpcd.conf
RUN chmod a+x /entrypoint.sh

EXPOSE 80

STOPSIGNAL SIGTERM

CMD ["/bin/sh", "entrypoint.sh"]