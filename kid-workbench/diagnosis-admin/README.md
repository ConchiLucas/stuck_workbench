# 孩子知识库

共用 `study_workbench`，按知识点查看能力状态、真实错题和复习建议。详见[产品分工](孩子知识库.md)与[开发文档](../docs/superpowers/plans/2026-09-12-child-knowledge-base.md)。

页面入口：`/` 知识总览，`/library` 能力档案，`/wrongs` 错题记录，`/reviews` 复习建议。首页是一排学科积累入口与能力状态、历史错误、复习状态三张图，明细从菜单进入。原 `/subjects/:code`、知识点和作答详情链接保留，旧首页筛选书签转至能力档案。历史日期用于定位作答，不提供第二套进度日历。

首页 Summary 返回各科 `wrongCount`、`abilities` 和 `unmappedPointCount`。能力分母为已练知识点中的适用能力，无法归属的旧作答记为状态待确认；没有能力拆分的知识点单独标注。`GET /api/v1/children/:cid/knowledge/review-status-summary` 只读统计本孩子未归档建议，阶段按有可归属真实作答、曾发布对应版本、有草稿、待生成依次判定，每份建议只计一次。有作答不表示已完成或已掌握；统计不可用不会填成零。

## 运行

在工作区根目录，只更新所需服务：

```sh
docker compose build diagnosis-admin
docker compose up -d --no-deps diagnosis-admin
```

打开 <http://localhost:19211>。题目后台 <http://localhost:19201> 负责预览、发布。

本地开发：后端 `APP_DB_HOST=127.0.0.1 APP_DB_PORT=15432 go run ./cmd/server`，前端目录 `npm install && npm run dev`，Vite 端口 `19212`。

## 配置

- `APP_TASK_ADMIN_URL`：固定题目后台地址。
- `APP_CONTENT_ADMIN_URL`、`APP_PINYIN_SERVER_URL`：历史媒体固定来源。
- `APP_REVIEW_SUGGESTIONS_ENABLED`：程序默认 false，Compose 显式启用。知识库与题目后台需同时启用才能保存和生成。旧自动复习开关不控制此功能。
- 数据库配置沿用 `APP_DB_*`。知识库本身不迁移或修改学习表；建议表由题目后台管理。

保存建议和生成草稿是两个操作，均用 `Idempotency-Key`；命令同时携带 `expectedRowVersion`。网络结果不明确时沿用原请求重试；内容版本已变则刷新后重新操作。

## 验证

后端 `go test ./...`，前端 `npm test` 与 `npm run build`。PostgreSQL 扩展测试仅使用独立验收库及临时 schema，不把正式孩子数据作为可写夹具。已执行的命令与限制记录在 [progress.md](docs/knowledge-base-documentation/progress.md)。
