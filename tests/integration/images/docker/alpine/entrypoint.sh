#!/bin/bash
echo $url>/usr/share/nginx/html/user-data.html
echo ''>/usr/share/nginx/html/received-data.html
/usr/bin/supervisord -c /etc/supervisord.conf