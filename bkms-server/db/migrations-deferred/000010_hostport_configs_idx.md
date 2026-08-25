# hostport_configs 唯一索引（已暂缓，未打入二进制）

> 本文件在 `db/migrations-deferred/`，**不会**被 `go:embed` 打包。上线前移回 `db/migrations/`。

- hostport_configs 新建索引：
  - appID_1: HostPortStoreMongo 约束每个应用仅一份 HostPort 端口映射配置。
