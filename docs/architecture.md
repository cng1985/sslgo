# 架构

## 分层

`cmd/server` 只启动进程。`internal/app` 组合平台、业务模块和前端，并在提供 `database.Ready` 时完成迁移和幂等初始化。`internal/platform` 不导入业务模块。业务模块可以依赖平台，证书模块依赖数据源和任务的仓储来完成签发。

| 模块 | 数据所有权 | 能力 |
| --- | --- | --- |
| auth | 租户、用户、角色、权限 | 登录、资料、用户和角色 |
| datasource | 数据源配置 | ACME 账户、DNS、Nginx 目标 |
| certificate | 证书和流水线 | 申请、部署、仪表盘 |
| task | 任务 | 查询执行记录 |
| schedule | 调度 | Cron 续期 |
| query | 无独立表 | 只读白名单查询 |

## 启动

1. 解析定位参数：`--base-dir`、`--config-dir`、`--data-dir`、`--profile`，以及 `APP_*` / `SSL_CONFIG`。
2. 读取 `application.yaml`，再读取被选中的一个环境文件。
3. 用环境变量和 `--server-host`、`--server-port` 覆盖。
4. 迁移表结构并初始化权限。空库才创建默认租户和管理员。
5. 调度器注册启动钩子，然后 HTTP 先监听再 `Serve`。
6. 停止时先关闭 HTTP，再停止调度器，最后关闭数据库。

`GET /health/live` 只表示进程存活。`GET /health/ready` 在 2 秒内检查数据库。两者都返回真实 HTTP 状态，不走业务 POST 协议。

## 配置

仓库提供三份配置：

- `configs/application.yaml`：基础配置，默认 SQLite 和文本日志。
- `configs/application-dev.yaml`：调试日志。
- `configs/application-pro.yaml`：JSON 日志。

相对 SQLite 路径拼到数据目录。`SSL_DATABASE_PATH` 或 `APP_DATABASE_PATH` 为绝对路径时保持原值。

## 数据

SQLite 是唯一完成验证的驱动。备份时要同时处理主库和 WAL，不能只复制正在使用的 `sslplatform.db`。恢复时先停止服务，再放回数据库文件和配置。

结构迁移由启动时的 AutoMigrate 完成，并在 `schema_migrations` 记录版本。已有用户、角色和数据源不会被重新初始化覆盖。
