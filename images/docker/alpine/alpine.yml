FROM nginx:1.17.9-alpine
RUN apk add --no-cache supervisor curl dhcpcd
COPY supervisord.conf /etc/supervisord.conf
WORKDIR /
COPY entrypoint.sh /entrypoint.sh
COPY dhcpcd.conf /dhcpcd.conf
RUN chmod a+x /entrypoint.sh

EXPOSE 80 443

CMD ["/bin/sh", "entrypoint.sh"]