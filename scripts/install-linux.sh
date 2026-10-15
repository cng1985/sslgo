#!/bin/sh
# 把证枢 SSL 平台安装为 systemd 服务。
# 用法：
#   sudo ./scripts/install-linux.sh
#   sudo ./scripts/install-linux.sh --bin ./dist/sslplatform-linux-amd64
#   sudo ./scripts/install-linux.sh --build
#   sudo ./scripts/install-linux.sh uninstall
#   sudo ./scripts/install-linux.sh uninstall --purge
#   ./scripts/install-linux.sh status
set -eu

SERVICE_NAME=sslplatform
BIN_DEST=/usr/local/bin/sslplatform
CONFIG_DIR=/etc/sslplatform
ENV_FILE=/etc/sslplatform/sslplatform.env
CONFIG_FILE=/etc/sslplatform/config.yaml
DATA_DIR=/var/lib/sslplatform
UNIT_FILE=/etc/systemd/system/sslplatform.service
CRED_FILE=/etc/sslplatform/initial-admin.txt
WEBROOT=/var/www/acme
CERT_DIR=/usr/local/nginx/conf/certs

ACTION=install
BIN_SRC=
DO_BUILD=0
PURGE=0
ADDR=:8080

usage() {
  cat <<'EOF'
用法: install-linux.sh [install|uninstall|status] [选项]

  install              安装或升级二进制、配置和 systemd 服务（默认）
  uninstall            停止服务并删除程序与单元文件，保留配置和数据库
  status               查看服务状态

选项:
  --bin PATH           指定要安装的二进制文件
  --build              在源码目录用本机 Go 编译当前平台
  --addr ADDR          首次安装时的监听地址，默认 :8080
  --purge              卸载时同时删除 /etc/sslplatform 和 /var/lib/sslplatform
  -h, --help           显示此说明

未指定 --bin 时，按以下顺序查找：
  脚本同目录的 sslplatform-linux-<arch> 或 sslplatform
  仓库 dist/sslplatform-linux-<arch>
  仓库 bin/sslplatform
EOF
}

die() {
  echo "错误: $*" >&2
  exit 1
}

info() {
  echo "$*"
}

script_dir() {
  (CDPATH='' cd -- "$(dirname "$0")" && pwd)
}

repo_root() {
  base=$1
  if [ -f "$base/../go.mod" ]; then
    (CDPATH='' cd -- "$base/.." && pwd)
    return
  fi
  printf '%s\n' "$base"
}

candidate_names() {
  case "$(uname -m)" in
    x86_64|amd64) printf '%s\n' sslplatform-linux-amd64 ;;
    aarch64|arm64) printf '%s\n' sslplatform-linux-arm64 ;;
    armv7l)
      printf '%s\n' sslplatform-linux-armv7
      printf '%s\n' sslplatform-linux-arm
      ;;
    armv6l)
      printf '%s\n' sslplatform-linux-armv6
      printf '%s\n' sslplatform-linux-arm
      ;;
    i386|i686) printf '%s\n' sslplatform-linux-386 ;;
    *) die "不支持的 CPU 架构: $(uname -m)" ;;
  esac
}

rand_hex() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32 | tr -d '\n'
    return
  fi
  od -An -N32 -tx1 /dev/urandom | tr -d ' \n'
}

listen_url() {
  addr=$1
  case $addr in
    http://*|https://*)
      printf '%s\n' "$addr"
      return
      ;;
    :*)
      printf 'http://127.0.0.1%s\n' "$addr"
      return
      ;;
    0.0.0.0:*)
      printf 'http://127.0.0.1:%s\n' "${addr#0.0.0.0:}"
      return
      ;;
    '[::]:'*)
      printf 'http://127.0.0.1:%s\n' "${addr##*:}"
      return
      ;;
  esac
  printf 'http://%s\n' "$addr"
}

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    die "请使用 root 执行，例如: sudo $0 $*"
  fi
}

require_systemd() {
  if ! command -v systemctl >/dev/null 2>&1; then
    die "未找到 systemctl，本脚本只安装 systemd 服务"
  fi
  if [ ! -d /run/systemd/system ]; then
    die "当前系统没有运行 systemd"
  fi
}

find_binary() {
  root=$1
  here=$2
  if [ -n "$BIN_SRC" ]; then
    [ -f "$BIN_SRC" ] || die "找不到二进制: $BIN_SRC"
    printf '%s\n' "$BIN_SRC"
    return 0
  fi
  names=$(candidate_names)
  # 文件名不含空格，按空白拆开即可。
  # shellcheck disable=SC2086
  for name in $names; do
    for dir in "$here" "$root/dist" "$(pwd)"; do
      if [ -f "$dir/$name" ]; then
        printf '%s\n' "$dir/$name"
        return 0
      fi
    done
  done
  for dir in "$here" "$root/bin" "$(pwd)"; do
    if [ -f "$dir/sslplatform" ]; then
      printf '%s\n' "$dir/sslplatform"
      return 0
    fi
  done
  return 1
}

build_binary() {
  root=$1
  out=$2
  command -v go >/dev/null 2>&1 || die "未找到 go，无法 --build"
  [ -f "$root/go.mod" ] || die "未找到源码目录，无法 --build"
  [ -f "$root/internal/web/dist/index.html" ] || die "缺少前端产物 internal/web/dist，请先在 web 目录执行 npm install && npm run build"
  info "正在编译 $out"
  (cd "$root" && CGO_ENABLED=0 go build -trimpath -o "$out" ./cmd/server)
}

write_config() {
  if [ -f "$CONFIG_FILE" ]; then
    info "保留已有配置 $CONFIG_FILE"
    return
  fi
  cat > "$CONFIG_FILE" <<EOF
server:
  addr: "$ADDR"
  mode: release
database:
  path: $DATA_DIR/sslplatform.db
auth:
  jwtSecret: change-me-please-32chars!!
  tokenTTL: 24h
acme:
  directoryURL: https://acme-staging-v02.api.letsencrypt.org/directory
  timeout: 120s
EOF
  chmod 0644 "$CONFIG_FILE"
  info "已写入 $CONFIG_FILE"
}

write_env() {
  if [ -f "$ENV_FILE" ]; then
    info "保留已有环境文件 $ENV_FILE"
    return 1
  fi
  jwt=$(rand_hex)
  password=$(rand_hex)
  cat > "$ENV_FILE" <<EOF
# 证枢 SSL 平台。修改后执行 systemctl restart sslplatform
SSL_CONFIG=$CONFIG_FILE
SSL_DATABASE_PATH=$DATA_DIR/sslplatform.db
SSL_SERVER_ADDR=$ADDR
SSL_SERVER_MODE=release
SSL_JWT_SECRET=$jwt
SSL_ADMIN_USER=admin
SSL_ADMIN_PASSWORD=$password
SSL_ACME_DIRECTORY=https://acme-staging-v02.api.letsencrypt.org/directory
EOF
  chmod 0600 "$ENV_FILE"
  url=$(listen_url "$ADDR")
  cat > "$CRED_FILE" <<EOF
用户名: admin
密码: $password
控制台: $url
请登录后立即修改密码，然后删除本文件。
EOF
  chmod 0600 "$CRED_FILE"
  info "已生成管理员凭据 $CRED_FILE"
  return 0
}

write_unit() {
  cat > "$UNIT_FILE" <<EOF
[Unit]
Description=证枢 SSL 平台
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
# 以 root 运行，以便覆盖 Nginx 证书并执行 nginx / systemctl。
WorkingDirectory=$DATA_DIR
EnvironmentFile=$ENV_FILE
ExecStart=$BIN_DEST
Restart=on-failure
RestartSec=3
TimeoutStopSec=20
KillMode=mixed
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
  chmod 0644 "$UNIT_FILE"
}

install_dir() {
  mode=$1
  dest=$2
  if command -v install >/dev/null 2>&1; then
    install -d -m "$mode" "$dest"
    return
  fi
  mkdir -p "$dest"
  chmod "$mode" "$dest"
}

install_file() {
  mode=$1
  src=$2
  dest=$3
  if command -v install >/dev/null 2>&1; then
    install -m "$mode" "$src" "$dest"
    return
  fi
  cp "$src" "$dest"
  chmod "$mode" "$dest"
}

prepare_dirs() {
  install_dir 0755 "$CONFIG_DIR"
  install_dir 0750 "$DATA_DIR"
  install_dir 0755 "$WEBROOT"
  install_dir 0750 "$CERT_DIR"
}

install_service() {
  require_root install
  require_systemd
  [ "$(uname -s)" = "Linux" ] || die "此脚本只能在 Linux 上安装服务"

  if [ "$DO_BUILD" -eq 1 ] && [ -n "$BIN_SRC" ]; then
    die "--bin 和 --build 不能同时使用"
  fi
  case $ADDR in
    *[!A-Za-z0-9:._-]*) die "监听地址包含不支持的字符: $ADDR" ;;
    "") die "监听地址不能为空" ;;
  esac

  here=$(script_dir)
  root=$(repo_root "$here")
  src=

  if [ "$DO_BUILD" -eq 1 ]; then
    built=$(mktemp)
    trap 'rm -f "$built"' EXIT INT TERM
    build_binary "$root" "$built"
    src=$built
  else
    src=$(find_binary "$root" "$here") || die "找不到本机架构的二进制。请执行 --build，或先在 Windows 上运行 scripts/build-platforms.ps1，再把 dist 中对应文件交给 --bin"
  fi
  [ -f "$src" ] || die "找不到二进制: $src"

  prepare_dirs
  write_config
  fresh=0
  if write_env; then
    fresh=1
  fi
  write_unit
  install_file 0755 "$src" "$BIN_DEST"
  if command -v restorecon >/dev/null 2>&1; then
    restorecon "$BIN_DEST" >/dev/null 2>&1 || true
  fi

  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME" >/dev/null
  if systemctl is-active --quiet "$SERVICE_NAME"; then
    systemctl restart "$SERVICE_NAME"
  else
    systemctl start "$SERVICE_NAME"
  fi

  if ! systemctl is-active --quiet "$SERVICE_NAME"; then
    systemctl --no-pager --full status "$SERVICE_NAME" || true
    journalctl -u "$SERVICE_NAME" -n 40 --no-pager || true
    die "服务未能启动"
  fi

  url=$(listen_url "$ADDR")
  if [ -f "$ENV_FILE" ]; then
    configured=$(awk -F= '$1=="SSL_SERVER_ADDR"{print $2}' "$ENV_FILE" | tail -n 1)
    if [ -n "$configured" ]; then
      url=$(listen_url "$configured")
    fi
  fi
  info "服务已启动: $url"
  info "日志: journalctl -u $SERVICE_NAME -f"
  if [ "$fresh" -eq 1 ]; then
    info "初始管理员写在 $CRED_FILE"
  else
    info "沿用已有管理员和 JWT 配置"
  fi
  if command -v curl >/dev/null 2>&1; then
    i=0
    while [ "$i" -lt 10 ]; do
      if curl -fsS -X POST "$url/api/v1/system/info" -H 'Content-Type: application/json' -d '{}' >/dev/null; then
        info "健康检查通过"
        return
      fi
      i=$((i + 1))
      sleep 1
    done
    info "进程已运行，但健康检查尚未通过，请查看 journalctl -u $SERVICE_NAME"
  fi
}

uninstall_service() {
  require_root uninstall
  if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
    systemctl disable --now "$SERVICE_NAME" >/dev/null 2>&1 || true
  fi
  rm -f "$UNIT_FILE" "$BIN_DEST"
  if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
    systemctl daemon-reload
  fi
  if [ "$PURGE" -eq 1 ]; then
    rm -rf "$CONFIG_DIR" "$DATA_DIR"
    info "已删除程序、服务、配置和数据库"
    return
  fi
  info "已停止服务并删除程序。配置与数据库仍保留在 $CONFIG_DIR 和 $DATA_DIR"
  info "若要一并删除，请执行: sudo $0 uninstall --purge"
}

status_service() {
  command -v systemctl >/dev/null 2>&1 || die "未找到 systemctl"
  systemctl --no-pager --full status "$SERVICE_NAME"
}

while [ $# -gt 0 ]; do
  case "$1" in
    install|uninstall|status)
      ACTION=$1
      ;;
    --bin)
      [ $# -ge 2 ] || die "--bin 需要路径"
      BIN_SRC=$2
      shift
      ;;
    --build)
      DO_BUILD=1
      ;;
    --purge)
      PURGE=1
      ;;
    --addr)
      [ $# -ge 2 ] || die "--addr 需要地址"
      ADDR=$2
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "未知参数: $1"
      ;;
  esac
  shift
done

case "$ACTION" in
  install) install_service ;;
  uninstall) uninstall_service ;;
  status) status_service ;;
esac
