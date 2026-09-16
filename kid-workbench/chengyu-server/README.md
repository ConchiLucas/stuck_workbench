# 成语 API

独立成语学习 API，默认监听 `:19181`。直接读取共享 PostgreSQL 的 `chengyu` 题库，计划写入 `schema=1` 历史快照，成语读音冻结进 `chengyu_question_task_media`（SHA-256 来自真实 MP3 字节）。听释义播放成语读音，不再使用浏览器 `speechSynthesis`。

```bash
go run ./cmd/server
```
