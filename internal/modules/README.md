# 业务模块

每个目录是一个叶子模块，公开业务方法是一个请求进、一个响应出。父目录不承载实体和控制器。

| 模块 | 职责 | 允许依赖 |
| --- | --- | --- |
| datasource | ACME 账户、云 DNS 和 Nginx 目标 | platform、acme、process |
| acme | 账户注册和 DNS 提供商适配，不提供 HTTP | 外部 DNS SDK |
| certificate | 证书、签发流水线、仪表盘 | platform、datasource、task、acme |
| task | 签发任务记录 | platform |
| schedule | 定时续期 | platform、certificate、task |
| query | 当前租户的白名单只读查询 | platform、datasource |
| process | `nginx`、`systemctl`、`service` 命令白名单 | 无业务模块 |

证书流水线在同一个模块内删除证书关联的调度。调用 ACME 或 Nginx 之前结束数据库事务。调度器以系统 Scope 执行流水线，不携带用户令牌。
