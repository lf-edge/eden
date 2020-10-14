#!/bin/sh

printf "${url}">/var/www/html/user-data.html
printf "">/var/www/html/received-data.html
printf "">/var/www/html/ifconfig.html

exec /usr/bin/supervisord -c /etc/supervisord.conf
