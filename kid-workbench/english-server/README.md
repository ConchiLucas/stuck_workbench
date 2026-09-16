# English Server

独立英语学习 API，默认监听 `:19131`。直接读取共享 PostgreSQL 和 MinIO，只复用 `shared-go` 学习内核，不依赖素材后台、进度后台或拼音服务运行。

```bash
go run ./cmd/server
```
