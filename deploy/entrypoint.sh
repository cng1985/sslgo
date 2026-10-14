#!/bin/sh
set -eu
mkdir -p /var/www/acme /etc/nginx/ssl /etc/nginx/conf.d /data /var/log/nginx
if [ ! -f /etc/nginx/ssl/placeholder.crt ]; then
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout /etc/nginx/ssl/placeholder.key \
    -out /etc/nginx/ssl/placeholder.crt \
    -days 1 -subj "/CN=localhost" >/dev/null 2>&1 || true
fi
nginx
exec /usr/local/bin/sslplatform
