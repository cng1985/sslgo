#!/bin/sh
set -eu
mkdir -p /var/www/acme /usr/local/nginx/conf/certs /etc/nginx/conf.d /data /var/log/nginx
nginx
exec /usr/local/bin/sslplatform
