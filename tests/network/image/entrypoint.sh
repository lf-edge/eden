#!/bin/sh

# shellcheck disable=SC2154
printf "%s" "${url}">/var/www/html/user-data.html
printf "">/var/www/html/received-data.html
printf "">/var/www/html/ifconfig.html

exec /usr/bin/supervisord -c /etc/supervisord.conf
