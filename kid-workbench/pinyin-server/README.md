# 拼音服务

独立的拼音孩子端 API。它直接读取共享 PostgreSQL 中由素材后台维护的 `pinyin` 目录、题库和 `pinyin_assets`，从共享 MinIO 只读素材，并在一个数据库事务内写入答题、掌握度、日统计和题单进度。

它不调用 `19091` 素材后台或 `19081` 进度后台。

```bash
go test ./...
go run ./cmd/server   # :19111
```

数据库使用 `APP_DB_*`，对象存储使用 `APP_MINIO_ENDPOINT`、`APP_MINIO_ACCESS_KEY`、`APP_MINIO_SECRET_KEY`、`APP_MINIO_BUCKET`、`APP_MINIO_BASE_PATH`、`APP_MINIO_USE_SSL`。
