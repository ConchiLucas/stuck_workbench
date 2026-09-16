# 算术题目后台实施记录（2026-09-13）

依据用户最新五端授权及 `03-五端菜单实施标准.md`。识字/拼音是样板；算术不套识字正方形图卡或拼音先听再选。部署与五端浏览器验收由主代理统一记录，本文不宣称用户已验收。

## 用户要求与实现

- 沿用识字任务卡片（`gen-task-list`）和全屏查看弹窗；右上保留小型「生成算术题目」。不显示发布/草稿/删除/重新生成。HTTP `POST .../publish` 仍保留兼容，UI 不暴露。
- 从素材后台已发布的 12 条详情生成；保存完整题干、四个选项顺序、稳定选项 ID（`o1`…）、正确答案 ID、来源修订。普通算式生成写入 `3 + 5`，不加 `= ?`。补空/判断保留等号。听音图形缺少音频时明确失败，不写半包。
- 真实 MP3 字节校验 ID3 或 MPEG 帧后按 SHA256 写入专用 `math_question_task_media`，题包改写为 `/api/v1/math/task-media/{sha}.mp3`。数量图/图形位图若出现未冻结 URL 会失败（当前生成器不写图片 URL，题面用字形回退）。题目与媒体同一事务。
- 预览复用 `@kid-workbench/math-player` 的 KidPlayer。播放不等于作答。试答只在 React 本地状态，没有预览 POST。关闭和 Esc 恢复触发按钮焦点。宿主用 `.math-task-player` 覆盖 App 的 `height:100%`，避免题卡列表裁切选项。
- 列表 GET 不含题目 `items`/`sequence`；详情 GET 才返回原题。

## 文件

- `shared-go/mathcontent/`：选择题干可无等号；冻结 `task-media` 合法；`PlanExampleFromSnapshot` 供进度/知识库还原。
- `task-admin/backend/internal/mathtask/`：生成、选项 ID、冻结音频、列表摘要。
- `task-admin/backend/internal/http/handler_math.go`、`math_tasks_test.go`：任务 API 与不可变媒体。
- `task-admin/frontend/src/pages/MathTasksPage.tsx`、`MathTasksPage.test.tsx`、`mathTasks.css`：精简列表、生成弹窗、同款预览。
- `task-admin/frontend/src/api/mathTasks.ts`：客户端不再调用发布。

## API

- `GET /api/v1/math/question-tasks`
- `POST /api/v1/math/question-tasks`，如 `{"title":"算术练习","detailIds":["addition-equation"],"rangeMax":5,"count":8}`
- `GET /api/v1/math/question-tasks/:id`
- `GET /api/v1/math/task-media/:sha256.mp3`
- `GET /api/v1/math/detail-audio/:file`（旧包代理；新包走冻结媒体）

## 验证

- `cd shared-go && go test ./mathcontent/` 通过。
- `cd task-admin/backend && go test ./internal/mathtask ./internal/http` 通过。HTTP 生成 addition-equation 不拉音频；列表 JSON 不含 `"sequence"`；无 attempts/mastery 表写入。
- `cd task-admin/frontend && npm test`：15 tests 通过（含本地试答无 POST、生成后 GET、Esc/焦点、无「发布/生成草稿」）。
- `cd task-admin/frontend && npm run build` 通过。

## 边界

- App 正式取题仍走 math-server 发布详情，不改走本题包。
- `defaults.json` 的听音图形没有 `audioUrl`；用该素材生成听音题会可见失败，这是要求而不是静默跳过。
- 网络已保存但响应丢失时重试可能再生成一包；页面生成按钮有并发锁。
- 「保存复习分析 → 专属错题复习生成 → 归因」未接入。
