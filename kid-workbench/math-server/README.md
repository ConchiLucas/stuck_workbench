# 算数服务

独立的算数孩子端 API，默认监听 `19141`。服务直接读取共享 PostgreSQL 中的算数目录、题目和计划，从 MinIO 只读预生成的题目音频，并通过 `shared-go` 原子写入作答、模块技能、掌握度、每日统计和奖励。

正式计划读取共享数据库，不调用进度后台。详情示例通过 `APP_CONTENT_ADMIN_URL` 读取素材后台已发布目录，缓存 60 秒；上游暂时不可用时可使用最近 24 小时内的缓存并标记 `stale`，没有缓存时明确返回不可用。App 不实时领取题目后台题包。

```bash
go test ./...
go run ./cmd/server
```

数据库使用 `APP_DB_*`，对象存储使用 `APP_MINIO_*`。`GET /healthz` 检查进程存活，`GET /readyz` 在两秒内检查 PostgreSQL 和 MinIO bucket。
