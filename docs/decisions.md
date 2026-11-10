# 已记录的项目决策

这些选择覆盖规范里的默认值，用来保持已经发布的客户端、安装脚本和测试。

## 协议

- 主键继续使用数字 `uint`，JSON 字段名保持 `id`。
- 分页继续返回 `{code,msg,data,page,size,total,responseType:"page"}`。`size` 大于 100 时截断为 100，不因此拒绝请求。
- 业务错误的 HTTP 状态码等于应用码。登录失效是 401，参数错误是 400。前端以响应体 `code === 200` 判断成功，同时依赖 HTTP 401 跳转登录。
- 路由保持 `POST /api/v1/{resource}/{action}`，字段继续使用 camelCase。

## 运行

- SQLite 日志模式使用 WAL。外键打开，busy timeout 5 秒，连接池最大连接和空闲连接都是 1，数据库文件权限 `0600`。
- 当前发行版只验证 SQLite。选择 mysql、postgres 或 sqlserver 会启动失败，不会悄悄退回 SQLite。
- 普通请求超时是 120 秒，与 ACME 调用超时一致。HTTP 服务不设置 ReadTimeout 和 WriteTimeout，避免切断长时间连接。
- 默认日志是文本。`application-pro.yaml` 使用 JSON。
- 监听地址默认 `127.0.0.1:8080`。`SSL_SERVER_ADDR` 和 `APP_SERVER_ADDR` 可以覆盖，容器使用 `:8080`。
- 环境变量同时接受 `APP_*` 和已经发布的 `SSL_*`。

## 安装

- 程序仍安装到 `/usr/local/bin/sslplatform`，配置在 `/etc/sslplatform`，数据在 `/var/lib/sslplatform`。
- 服务继续以 root 运行，因为部署证书需要写入 Nginx 目录并执行 `nginx` 或 `systemctl`。
- 已安装机器上的 `/etc/sslplatform/config.yaml` 可通过 `SSL_CONFIG` 继续加载。

## 模块边界

- Resource 的 HTTP 业务方法遵循一个 Request 或 Search 进、一个 Response 或 PageResponse 出。租户和用户来自上下文中的 Scope。
- 证书流水线跨模块读写数据源和任务。这些 Repository 方法是持久化操作，事务不包裹 ACME 的外部 HTTP 调用。
- 下载接口的业务结果放在 `CertificateDownload` 中，再由统一 JSON 外壳返回，不另开文件流协议。
