#!/bin/sh

printf "${url}">/var/www/html/user-data.html
printf "">/var/www/html/received-data.html
printf "">/var/www/html/ifconfig.html

/usr/bin/supervisord -c /etc/supervisord.conf