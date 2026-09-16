# 英语短句 API

独立英语短句学习 API，默认监听 `:19171`。直接读取共享 PostgreSQL 的 `phrase` 题库，快照写入 `study_plans` / `plan_items`，通过 `shared-go` 记录掌握度。语音由孩子端浏览器 `speechSynthesis` 朗读快照文本，不使用 MinIO。

```bash
go run ./cmd/server
```

题型变化后需要重新 `make seed`，数据库里才会出现 `scene` / `reply` 题目。
