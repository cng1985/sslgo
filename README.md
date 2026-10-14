# 证枢 SSL 平台

用一个 Go 二进制管理 Let's Encrypt 证书：申请、替换服务器上的旧证书，然后重启 Nginx。Vue 3 管理台被打包进同一个文件。

## 目录

```text
cmd/server/main.go          只负责启动
internal/app                Fx 根模块
internal/config             配置加载
internal/database           SQLite 迁移与初始化
internal/server             Gin、POST 路由和 HTTP 生命周期
internal/web                embed 的前端
internal/auth               登录、JWT、RBAC
internal/datasource         ACME 账户与 Nginx 目标
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
| datasources | `acme` 账户或 `nginx` 部署目标 |
| certificates, etl_jobs | 证书申请和签发流水线 |
| tasks | 每次执行的阶段和结果 |
| schedules | 续期计划 |

私钥不放进接口响应。证书私钥写在 Nginx 的 `certDir/cert-{id}.key`，账户私钥留在数据源配置中。

首次启动会创建租户 `default`、角色 `admin` / `operator` / `viewer`，以及管理员。默认账号 `admin`，密码来自 `SSL_ADMIN_PASSWORD`，未设置时是 `Admin@123456`。请立刻修改。

## 签发流水线

一次执行是 `y = f(x)`：

1. **Extract**：按登录态里的租户读取证书、ACME 账户和 Nginx 目标。未进入续期窗口则停止。
2. **Transform**：用 HTTP-01 webroot 向 Let's Encrypt 申请证书。
3. **Load**：备份旧文件，写入新的 `cert-{id}.crt`、`.key`、`.issuer.crt`，执行 `nginx -t`，再按配置重启或重载 Nginx。检测失败会恢复旧文件。

默认动作是重启。`signal` 模式执行 `nginx -s quit` 后再启动；`systemd` 模式执行 `systemctl restart nginx`。只允许 `nginx`、`systemctl`、`service`，参数按数组传递，不经过 shell。

Nginx 在签发前必须能通过 80 端口提供 `webroot/.well-known/acme-challenge/`。通配符证书需要 DNS-01，当前版本不签发通配符。

默认 ACME 目录是测试环境。生产目录是 `https://acme-v02.api.letsencrypt.org/directory`。

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
- 证书目录 `/etc/nginx/ssl`
- 动作 `restart`

把域名解析到这台机器，在页面上选择生产 ACME 目录并签发。证书文件名是 `cert-{id}.crt` 与 `cert-{id}.key`。如果填写了片段目录，还会生成 `/etc/nginx/conf.d/sslplatform-cert-{id}.conf`。

## 测试

```sh
go test ./...
```

集成测试使用内存签发器替换真实 ACME，并检查证书文件、Nginx 重启参数、RBAC、租户隔离和 SPA。
