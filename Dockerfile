FROM node:22-bookworm-slim AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -o /out/sslplatform ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates nginx openssl curl \
    && rm -rf /var/lib/apt/lists/*
COPY deploy/nginx.conf /etc/nginx/nginx.conf
COPY deploy/entrypoint.sh /entrypoint.sh
COPY --from=build /out/sslplatform /usr/local/bin/sslplatform
RUN chmod +x /entrypoint.sh
EXPOSE 8080 80
VOLUME ["/data", "/usr/local/nginx/conf/certs", "/var/www/acme"]
ENV SSL_DATABASE_PATH=/data/sslplatform.db \
    SSL_BOOTSTRAP_NGINX=true \
    SSL_SERVER_ADDR=:8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -fsS -X POST http://127.0.0.1:8080/api/v1/system/info -H 'Content-Type: application/json' -d '{}' || exit 1
ENTRYPOINT ["/entrypoint.sh"]
