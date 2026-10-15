# 证枢 SSL 平台

用一个 Go 二进制管理 Let's Encrypt 证书：申请、替换服务器上的旧证书，然后重启 Nginx。Vue 3 管理台被打包进同一个文件。

## 目录

```text
cmd/server/main.go          只负责启动
scripts/install-linux.sh    Linux systemd 服务安装
scripts/build-platforms.ps1 Windows 上生成各平台应用
internal/app                Fx 根模块
internal/config             配置加载
internal/database           SQLite 迁移与初始化
internal/server             Gin、POST 路由和 HTTP 生命周期
internal/web                embed 的前端
internal/auth               登录、JWT、RBAC
internal/datasource         ACME 账户、云厂商 DNS 与 Nginx 目标
internal/acme               ACME 账户注册，以及阿里云、腾讯云等 DNS-01
internal/etl                证书与 Extract -> Transform -> Load
internal/scheduler          Cron 续期
internal/task               签发任务
internal/common             统一响应与通用查询
web                         Vue 3 源码
```

## Fx 模块

`cmd/server/main.go` 调用 `app.Run()`。各业务模块自行 `fx.Provide` 仓储、服务、控制器，并用 `fx.Invoke` 注册路由。HTTP 服务、调度器和数据库迁移都挂在 Fx 生命周期上。

## 数据库

SQLite 文件默认是 `data/sslplatform.db`，权限 `0600`。启动时自动迁移这些表：

| 表 | 用途 |
| --- | --- |
| tenants, users, roles, permissions, user_roles, role_permissions | 租户和 RBAC |
| datasources | `acme` 账户（可绑定云厂商 DNS）或 `nginx` 部署目标 |
| certificates, etl_jobs | 证书申请和签发流水线 |
| tasks | 每次执行的阶段和结果 |
| schedules | 续期计划 |

私钥不放进列表、详情和通用查询。单独申请时，证书私钥保存在证书记录里，只能通过下载接口取出；部署后会按域名覆盖 Nginx 证书目录里的 `域名.key`，例如 `/usr/local/nginx/conf/certs/nbsaas.com.key`。账户私钥留在数据源配置中。

首次启动会创建租户 `default`、角色 `admin` / `operator` / `viewer`，以及管理员。默认账号 `admin`，密码来自 `SSL_ADMIN_PASSWORD`，未设置时是 `Admin@123456`。请立刻修改。

## 签发流水线

一次执行是 `y = f(x)`：

1. **Extract**：按登录态里的租户读取证书、ACME 账户和 Nginx 目标。未进入续期窗口则停止。
2. **Transform**：按验证方式申请证书。`http-01` 把挑战文件写到 Nginx webroot。`dns-01` 调用 ACME 账户绑定的云厂商 DNS，写入 `_acme-challenge` TXT 记录，因此可以签发通配符。
3. **Load**：仅在需要部署时执行。按主域名覆盖证书目录里已有的 `域名.pem` 和 `域名.key`（`*.nbsaas.com` 对应 `nbsaas.com.pem` 与 `nbsaas.com.key`），执行 `nginx -t`，再重启或重载 Nginx。不会修改 Nginx 配置文件。检测失败会恢复旧文件。只申请时流水线在 Transform 结束，状态为 `issued`，不写证书目录，也不重启 Nginx。

默认动作是重启。`signal` 模式执行 `nginx -s quit` 后再启动；`systemd` 模式执行 `systemctl restart nginx`。只允许 `nginx`、`systemctl`、`service`，参数按数组传递，不经过 shell。

证书可以不绑定 Nginx。DNS-01 创建时省略 `nginxDataSourceId` 即可单独申请，适合先拿到证书再自行安装。HTTP-01 仍需要 Nginx 的 webroot 完成验证，但可以只申请、不立刻重启。

- `POST /api/v1/certificate/apply`：只向 ACME 申请。
- `POST /api/v1/certificate/issue`：申请并部署，证书必须已绑定 Nginx。
- `POST /api/v1/certificate/download`：按 `nginx`、`httpd` 或 `iis` 打包已申请的证书。
- `POST /api/v1/certificate/deploy`：把已申请的材料写入 Nginx 并重启。库里没有材料时，沿用磁盘上的证书文件只做重启。

调度续期时，没绑定 Nginx 的证书只会重新申请。`POST /api/v1/etlJob/execute` 可以传 `deploy: false`；省略时，已绑定 Nginx 就部署，否则只申请。

HTTP-01 在签发前必须能通过 80 端口提供 `webroot/.well-known/acme-challenge/`。通配符只能使用 DNS-01，并且 ACME 账户要先绑定云厂商 DNS。

创建 ACME 数据源时会生成账户私钥。可以立刻向证书机构注册，也可以稍后调用 `POST /api/v1/datasource/register`。ZeroSSL、Google Trust Services 和 SSL.com 需要外部账号绑定（EAB）。支持的 DNS 包括阿里云、腾讯云 DNS、DNSPod Token、华为云、百度智能云、火山引擎、Amazon Route 53、Azure DNS、Google Cloud DNS、Cloudflare 和 DigitalOcean。`POST /api/v1/datasource/catalog` 返回机构和凭证字段。密钥不会出现在接口响应里，更新时留空表示保留原值。

默认 ACME 目录是 Let's Encrypt 测试环境。生产目录是 `https://acme-v02.api.letsencrypt.org/directory`。DNS-01 会等待解析生效，耗时通常长于 HTTP-01。

## 证书下载

证书列表返回 `notAfter` 和 `daysRemaining`。已签发的证书可以下载三种压缩包：

| format | 内容 |
| --- | --- |
| `nginx` | `域名.pem`、`域名.key`，以及一份仅供对照的 `nginx.conf`。`*.nbsaas.com` 写成 `nbsaas.com.pem` 和 `nbsaas.com.key`。自动部署不会把这份配置写到服务器上 |
| `httpd` | `server.crt`、`server.key`、`chain.crt`、`httpd.conf`。没有中间证书时省略证书链 |
| `iis` | `certificate.pfx`、`iis.txt`。请求体要带 8 到 64 位 `password`，PFX 使用 IIS 可导入的传统 PKCS#12 |

`apache` 与 `httpd` 相同。尚未签发时下载会返回 400。

## 接口

全部业务接口都是 `POST /api/v1/{resource}/{action}`，请求和响应为 JSON。

```json
{ "code": 200, "msg": "success", "data": {} }
```

分页：

```json
{ "code": 200, "msg": "success", "data": [], "page": 1, "size": 20, "total": 100, "responseType": "page" }
```

通用查询接受 `certificate`、`etlJob`、`task`、`datasource`、`user`、`schedule`、`role`，以及 `pageETLJob` 这类别名：

- `POST /api/v1/query/search`
- `POST /api/v1/query/list`
- `POST /api/v1/query/one`

`POST /api/v1/system/catalog` 返回资源、动作和字段说明。请求体里出现 `tenantId` 会被拒绝，租户只从 JWT 解析。

## 前端与 embed

```go
//go:embed all:dist
var Dist embed.FS
```

`internal/web` 提供 `dist`。Gin 托管静态文件，未知的页面路径回退到 `index.html`，因此 Vue Router 的 history 模式可用。`/api` 只接受 POST。

```sh
cd web && npm install && npm run build
CGO_ENABLED=0 go build -o bin/sslplatform ./cmd/server
./bin/sslplatform
```

浏览器打开 `http://127.0.0.1:8080`。

## SQLite 初始化

配置见 `configs/config.yaml`，环境变量优先：

| 变量 | 含义 |
| --- | --- |
| `SSL_DATABASE_PATH` | SQLite 路径 |
| `SSL_JWT_SECRET` | 至少 16 位的签名密钥 |
| `SSL_ADMIN_USER` / `SSL_ADMIN_PASSWORD` | 初始管理员，只在空库生效 |
| `SSL_ACME_DIRECTORY` | 默认 ACME 目录 |
| `SSL_BOOTSTRAP_NGINX` | 为 `true` 时写入容器内的 ACME 和 Nginx 数据源 |

## Docker

```sh
docker compose up --build
```

容器内同时运行 Nginx 和平台。80 端口提供 HTTP-01 目录，8080 是控制台。初始化后的 Nginx 数据源指向：

- 程序 `/usr/sbin/nginx`
- webroot `/var/www/acme`
- 证书目录 `/usr/local/nginx/conf/certs`
- 动作 `restart`

把域名解析到这台机器，在页面上选择生产 ACME 目录并签发。部署只覆盖该目录中的 `域名.pem` 与 `域名.key`，例如 `nbsaas.com.pem` 和 `nbsaas.com.key`，然后重启 Nginx。Nginx 配置需要事先写好这两个文件的路径，平台不会改配置。

## Linux 服务

在目标机器上用 root 安装 systemd 服务。服务以 root 运行，这样部署证书时可以写入 Nginx 目录并执行 `nginx` / `systemctl`。

```sh
sudo ./scripts/install-linux.sh --build
```

已经有二进制时：

```sh
sudo ./scripts/install-linux.sh --bin ./dist/sslplatform-linux-amd64
```

脚本会安装 `/usr/local/bin/sslplatform`，写入 `/etc/sslplatform/config.yaml` 和权限为 `0600` 的 `/etc/sslplatform/sslplatform.env`，数据放在 `/var/lib/sslplatform`。首次安装会生成管理员密码，写在 `/etc/sslplatform/initial-admin.txt`。再次执行会替换程序并重启，已有配置、密钥和管理员密码保持不变。`--addr` 只在首次安装时生效。

同时准备 HTTP-01 目录 `/var/www/acme` 和证书目录 `/usr/local/nginx/conf/certs`。控制台默认是 `http://127.0.0.1:8080`。日志用 `journalctl -u sslplatform -f`。

```sh
./scripts/install-linux.sh status
sudo ./scripts/install-linux.sh uninstall
sudo ./scripts/install-linux.sh uninstall --purge
```

`uninstall` 保留配置和数据库。`--purge` 会删除这两项目录。

## 各平台应用

在 Windows 上生成 Linux、Windows 和 macOS 的可执行文件与压缩包：

```bat
scripts\build-platforms.cmd
scripts\build-platforms.cmd -Version 1.2.0 -Platform linux/amd64,windows/amd64
scripts\build-platforms.cmd -SkipWeb
```

默认目标是 `linux/amd64`、`linux/arm64`、`linux/arm/7`、`windows/amd64`、`windows/arm64`、`darwin/amd64`、`darwin/arm64`。产物在 `dist/`：

| 文件 | 内容 |
| --- | --- |
| `sslplatform-linux-amd64` | 裸二进制，可直接交给安装脚本的 `--bin` |
| `sslplatform-linux-armv7` | 32 位 ARM，`GOARM=7` |
| `sslplatform-<系统>-<架构>.zip` | 程序、`configs/config.yaml` 和启动脚本 |
| `SHA256SUMS.txt` | 上述文件的 SHA-256 |

压缩包在对应目录下执行 `sslplatform`、`sslplatform.exe` 或 `start.sh` / `start.cmd`。版本号写入 `internal/appinfo.Version`。需要 Go 1.24 和 Node.js；前端已经构建过时可以用 `-SkipWeb`。

## 测试

```sh
go test ./...
```

集成测试使用内存签发器替换真实 ACME，并检查证书文件、Nginx 重启参数、RBAC、租户隔离和 SPA。
