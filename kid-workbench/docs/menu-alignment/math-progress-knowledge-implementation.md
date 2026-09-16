# 算术：进度与孩子知识库实施记录

2026-09-13。知识库纳入五端。未改学习写入、掌握算法、日周月统计口径。未宣称用户验收。

## 进度（parent-dashboard :19081）

掌握地图仍是正式四项：加减 `calc+story`，图形 `find+name`。12 详情不是 12 项必需技能。

- `mathMatrix` / `correctMathOverview`：按技能汇总，忽略过期 `mastery_states` 合计，避免单技能或旧聚合把知识点标成完全掌握。
- `mathDetail`：从 `attempts` 关联最新匹配的 `plan_items.question_snapshot` + `picks`（按 question_id+kp_id+child）。Postgres JSONB 用 `CAST(question_snapshot AS TEXT)`。历史只读，不重出题。
- `buildMathReview`：calc/story/name 去掉语音；find 走 `/api/math/children/{id}/plans/{plan}/items/{item}/speech.mp3` 代理 math-server 计划音频，并标 `audio_mutable`（对象键 `math/questions/{id}.mp3` 可变）。
- `MathHistoryQuestion`：共用 KidPlayer `readOnly`；文案「当时答对/当时答错」，无「再做一次」。缺快照明确提示。
- Docker 复制 `packages/math-player`；`MATH_SERVER_URL` 默认 `http://math-server:19141`。

## 孩子知识库（diagnosis-admin :19211）

- `decodeMath`：同一快照转换；选项稳定 ID；记录孩子选了哪一项。find 音频经 `math:` 前缀代理 `APP_MATH_SERVER_URL`，只允许 `/api/v1/children/`。
- `MathEvidence` + `AnswerEvidence` 分派；错选题型筛选名为 算式计算 / 情境应用 / 听音找图形 / 看图认名称。
- `ReviewBlockReasons` 含 `unsupported_subject`：可保存分析，专属复习生成未接入。
- 无第二套日历/矩阵；不写掌握账本。测试夹具隔离，不写真实孩子。

无 `question_snapshot` 列的隔离库跳过算术 join，避免拖垮识字/拼音事实查询。

## 文件

- `parent-dashboard/backend/internal/service/math_progress.go`、`math_review.go`、`math_review_test.go`、`math_progress_test.go`
- `parent-dashboard/backend/internal/http/math_media.go`、`math_media_test.go`
- `parent-dashboard/frontend/src/components/question/MathHistoryQuestion.tsx`、`mastery/math/MathDetailDrawer.tsx`
- `diagnosis-admin/backend/internal/knowledge/math.go`、`sources.go`、`library.go`
- `diagnosis-admin/frontend/src/components/knowledge/MathEvidence.tsx`、`AnswerEvidence.tsx`、`WrongAnswersPage.tsx`

## 验证

- 进度 backend `go test ./internal/service ./internal/http -run 'Math'` 通过；历史长度仍为 6。
- 进度 frontend `npm test && npm run build` 通过。
- 知识库 backend `go test ./internal/knowledge ./internal/http` 通过。
- 知识库 frontend `npm test` 30 tests；`npm run build` 通过。

## 确切限制

- 听音历史音频是可变对象键，页面提示可能已更新；禁止 TTS 顶替。
- 多次作答同一 plan item 时，所选下标取 `picks` 最后一项。
- `math_assets.enabled` 不参与进度/知识库过滤。
- App 取题改走题目后台、复习分析→专属错题生成→归因，仍未做。
