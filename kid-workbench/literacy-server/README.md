# 识字服务

独立的识字孩子端 API。它直接读取共享 PostgreSQL 中由素材后台维护的 `literacy` 目录、题库和 `literacy_assets`，从共享 MinIO 只读字图、义图和读音，并在一个数据库事务内写入答题、掌握度、日统计和题单进度。

它不调用 `19091` 素材后台或 `19081` 进度后台。

```bash
go test ./...
go run ./cmd/server   # :19151
```

数据库使用 `APP_DB_*`，对象存储使用 `APP_MINIO_*`。默认 bucket 为 `study-assets`。
