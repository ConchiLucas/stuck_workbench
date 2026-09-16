# 科普服务

独立的科普儿童端 API，默认监听 `19121`。它直接读取共享 PostgreSQL 中已发布的科普内容和 `recognize` 题目，从 MinIO 只读素材，并通过 `shared-go` 原子写入作答、掌握度、日统计与题单进度。

它不会调用 `19091` 素材后台或 `19081` 进度后台；素材后台停止后，已发布内容和学习闭环仍可继续运行。

```bash
go test ./...
go run ./cmd/server
```

数据库使用 `APP_DB_*`，对象存储使用 `APP_MINIO_*`。`GET /healthz` 检查进程存活，`GET /readyz` 检查数据库并单独报告素材配置状态。
