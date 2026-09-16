# 题目后台任务出题与复习出题 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `executing-plans` to implement this plan task-by-task. Steps use checkbox (`- `) syntax for tracking. 用户已授权自主决定细节；默认当前工作目录内联执行，不再次询问设计或执行方式。

**Goal:** 把 task-admin 从固定抽取识字题的后台升级为使用素材按任务生成题目、发布给孩子练习、根据答题结果生成复习任务的题目后台。

**Architecture:** content-admin 提供不可变素材修订，task-admin 负责模板生成、题目版本和发布。literacy-server 将已发布题包复制为学习计划，通过既有 shared-go 事务记录作答，并保存可追溯回执；复习任务复用同一出题引擎。

**Tech Stack:** 沿用 Go、Gin、GORM、PostgreSQL、SQLite 测试、MinIO、React、TypeScript、Vite、TanStack Query、Vitest。第一版不引入消息队列、大模型服务或新前端框架。

---

## 0. 执行约定与当前状态

- [x] 阅读现有代码并确认职责。
- [x] 固化[设计基线](../specs/2026-09-12-question-task-generation-design.md)。
- [x] 写入本计划，并按后续用户授权完成开发。
- [x] 完成 Task 0–13，交付识字普通任务与手动复习的完整版本。
- [x] 完成 Task 14，交付默认关闭的自动复习生成增量。

本计划所有文件路径均相对于 `/Users/conchi/workforce/english_workforce/kid-workbench`。执行命令时使用下面规定的工作目录，或将路径展开为绝对路径。

用户随后已授权开发，现已完成首版并部署。详细验收见 [实际验收记录](../../verification/2026-09-12-question-task-generation.md)。下方保留原始实施步骤作为设计与测试参考；实际文件组织、测试路径及必要调整以验收记录为准。实现期间依据测试、现有约束自行调整细节，并在本文件决策记录中说明；不因普通实现选择停止等待用户。仅遇到需要外部凭据、不可恢复数据删除等实际边界才说明阻塞。

Git 根目录位于上一级，当前有大量既有暂存、重命名和未暂存改动。开始前记录状态；不得 `git add .`、重置工作区或提交用户其他改动。每个任务保存一个可复查改动单元；如果当前 Git 状态无法精确隔离提交，则记录完成点，不强行提交。已有根目录 task_plan.md / findings.md / progress.md 持续追加本任务进展。

## 1. 里程碑与依赖

| 里程碑 | 任务 | 可验收结果 |
| --- | --- | --- |
| M1 素材可出题 | 0–3 | 素材就绪信息、冻结修订及生成题的存储结构可用 |
| M2 后台可生成发布 | 4–7 | 指定范围、题量、题型生成；试答、替题、发布、撤回 |
| M3 孩子可完成任务 | 8–10 | 领取题包、稳定呈现、服务端判题、回执与学习数据一致 |
| M4 手动复习闭环 | 11–13 | 从真实错选生成复习任务，完成兼容迁移与端到端验收 |
| M5 自动复习增量 | 14 | 已完成普通任务可自动生成复习草稿，默认关闭 |

依赖顺序：0 → 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11 → 12 → 13 → 14。每阶段通过对应检查后直接进入下一项，不逐项确认。

## 2. 文件职责地图

| 创建目录/文件 | 职责 |
| --- | --- |
| content-admin/backend/internal/materials/ | 素材候选、就绪判断、冻结、媒体修订读取 |
| content-admin/backend/internal/http/handler_materials.go | 素材出题接口适配 |
| task-admin/backend/internal/materialclient/ | 调用素材后台，超时、错误、响应校验 |
| task-admin/backend/internal/generation/ | 纯模板、请求校验、去重与题目快照 |
| task-admin/backend/internal/qtask/repository.go | 出题任务、修订、版本、幂等生成记录持久化 |
| task-admin/backend/internal/qtask/lifecycle.go | 编辑、生成、发布、撤回、归档规则 |
| task-admin/backend/internal/review/ | 作答事实读取、复习策略、复习任务创建 |
| task-admin/frontend/src/pages/TaskDetailPage.tsx | 可直接打开 URL 的任务详情 |
| task-admin/frontend/src/components/TaskCreateForm.tsx | 出题要求表单和素材可用性 |
| task-admin/frontend/src/components/ReviewTaskDialog.tsx | 从来源练习生成复习任务 |
| literacy-server/internal/taskbank/ | 已发布任务列表、领取与快照转换 |
| literacy-server/internal/practice/task_answer.go | 新生成题的快照判题与原子回执 |
| literacy-app/src/pages/QuestionTasksPage.tsx | 孩子练习任务入口 |

保持已有 service.go 对外兼容，只把新职责放入相应文件；不顺手改造无关学科或建立大型通用平台。

## Task 0：记录基线和实现边界

**文件**：修改 `AGENTS.md`、`task-admin/README.md`；追加根目录三个规划文件。

-  在 AGENTS.md 新增题目后台分工：任务出题、素材消费、复习出题；明确不展示学习情况、不写掌握度。保留其他后台现有分工。
-  README 分别标记现有 legacy 抽题能力、首版目标与实现进度，不提前写成已完成。
-  记录 `git status --short -- task-admin content-admin literacy-server literacy-app shared-go parent-dashboard diagnosis-admin docs AGENTS.md`。
-  分别在 task-admin/backend、content-admin/backend、literacy-server、shared-go、parent-dashboard/backend、diagnosis-admin/backend 执行 `go test ./...`，记录实际基线失败。
-  在 task-admin/frontend 和 literacy-app 执行 `npm test`、`npm run build`；缺依赖时使用现有 lockfile 安装，不升级包版本。

**通过标准**：明确既有失败与本次改动无关；后续不能用“原来就失败”掩盖新增失败。这里只运行隔离测试，不向实际学习库插测试作答。

## Task 1：素材候选与出题就绪契约

**创建**：`content-admin/backend/internal/materials/types.go`、`service.go`、`service_test.go`、`content-admin/backend/internal/http/handler_materials.go`。  
**修改**：`content-admin/backend/internal/http/router.go`、`cmd/server/main.go`。

-  写表驱动测试：完整素材支持两种题型；缺义图不支持相应角色；缺音频不可用；未知组返回 404；同文字/同图片摘要不得充当不同答案。
-  在 content-admin/backend 运行 `go test ./internal/materials -run TestReadiness -v`，确认新测试先失败。
-  实现 `GET /api/v1/generation-materials/literacy?moduleCode=g1`。返回如下结构，items 包含组内全部候选，保留不可用项供表单解释原因：

```json
{"subjectCode":"literacy","moduleCode":"g1","items":[{"kpId":1,"text":"春","sourceRevision":"sha256:content-digest","capabilities":{"glyph_sense":{"ready":false,"reasons":["missing_sense_image"]},"sense_char":{"ready":false,"reasons":["missing_sense_image"]}}}]}
```

-  复用 literacy service 与 storage 读取能力；候选列表只做元数据检查，冻结时再验证真实媒体字节，避免列表批量下载。
-  增加路由测试确认返回业务 JSON，错误不返回 SPA HTML。运行 `go test ./internal/materials ./internal/http`。

**通过标准**：任务表单能回答“哪些字可用、缺什么”，不需要自己读取 literacy_assets 私有字段。

## Task 2：冻结素材与稳定媒体

**创建**：`content-admin/backend/internal/materials/revisions.go`、`revisions_test.go`、`content-admin/backend/internal/db/material_revisions.go`。  
**修改**：`content-admin/backend/internal/db/db.go`、`internal/http/handler_materials.go`、`internal/storage/minio.go`（仅必要的内容寻址支持）、`internal/literacy/service.go`（素材写入与冻结的并发协调）。

-  用 fake ObjectPutter 测试：冻结后覆盖原素材，旧 revision 字节不变；相同摘要复用；来源修订变化返回 409；对象写失败不产生 ready 修订。
-  运行 `go test ./internal/materials -run TestFreeze -v`，确认失败后实现。
-  增加 `material_revisions` 表，字段为 id（摘要字符串主键）、subject_code、kp_id、source_revision、content_json、media_json、created_at。PostgreSQL/SQLite 对应 DDL 均由素材后台迁移负责。
-  实现 POST freeze：最多 80 个候选，调用超时 30 秒；并发读取媒体上限 4；单张图片上限 10 MiB、单段音频上限 20 MiB。校验 PNG/MP3 类型、非空和摘要，拒绝非枚举资源。
-  用 SHA-256 内容寻址对象路径保存真实媒体；冻结前后校验 sourceRevision。存储读取只走内部对象 key，不使用客户端传来的 URL。
-  冻结和素材覆盖写共享按 kpId 的跨进程锁，多个 ID 按升序获取。PostgreSQL 使用行锁；SQLite 测试用写事务串行化。耗时的图片/音频生成放锁外，最终对象写入和元数据更新在锁内。用并发测试验证不产生混合版本。
-  实现 GET revision media：正确 Content-Type、ETag、不可变缓存；404/503 明确区分。后台同源代理在 Task 6 接入，孩子端代理在 Task 8 接入。
-  运行 `go test ./internal/materials ./internal/http ./internal/storage ./internal/db`。

**通过标准**：媒体字节稳定；仅添加 `?updatedAt=` 或保存最新 URL 不算完成。

## Task 3：任务、修订与题目版本存储

**创建**：`task-admin/backend/internal/db/question_task_v2.go`、`question_task_v2_test.go`、`task-admin/backend/internal/qtask/types.go`、`repository.go`、`repository_test.go`。  
**修改**：`task-admin/backend/internal/db/db.go`。

-  为已有 task-admin schema 建 fixture，先写升级、重复升级、旧任务 ID 不变、事务失败回滚测试。
-  运行 `go test ./internal/db ./internal/qtask -run 'TestMigration|TestRevision' -v`，确认新约束尚未实现。
-  按设计第 6 节新增 revisions、question_versions、generation_runs、review_sources；任务增加 kind/spec/source_mode/target_child_id/parent_task_id/active_revision_id/published_revision_id/row_version。
-  约束：任务 kind 为 practice/review；修订序号 task 内唯一；question_versions 的 seq 与 fingerprint 在修订内唯一；生成幂等键 task + operation + key 唯一。
-  旧 task 默认 `source_mode=legacy_pool`；新 task 为 `material_template`。保留旧 question_task_items，不回填伪造历史版本，不修改 questions 唯一索引。
-  Repository 暴露 CreateDraft、GetRevision、CommitRevision、PublishRevision、Withdraw、Archive；CommitRevision 与指针切换同事务，使用 row_version 比较更新。
-  用两个并发数据库连接验证同 expectedRowVersion 只允许一个提交。SQLite 验证逻辑约束，PostgreSQL 验证真实锁行为。

**通过标准**：新旧任务可共存；历史修订不可修改；失败不损坏原可用修订。

## Task 4：纯识字出题引擎

**创建**：`task-admin/backend/internal/generation/types.go`、`validate.go`、`literacy.go`、`fingerprint.go`、`literacy_test.go`。

-  定义设计第 7 节快照对应 Go 类型；使用 json tag，与前端 DTO 字段保持一致。
-  先写确定性测试：4 个目标字、每型 4 题产生 8 道；每题四个不同选项、唯一正确项；相同 seed 产物相同；更换选项顺序不改变指纹。
-  运行 `go test ./internal/generation -run TestLiteracy -v`，确认失败后实现。
-  请求校验包括 count 1–20、typeCounts 合计、只接受注册题型、目标字属于选定组、素材就绪、每型容量。错误返回 field、code、message、required、available。
-  区分普通生成与复习生成的内部输入：普通按 typeCounts 分配且每个 kp/type 最多一题；复习接收明确的 source snapshot 与目标 kp/type 列表，允许同组原题和变式，仍禁止重复 fingerprint。两种输入共用模板、素材验证与指纹实现。
-  先按 kpId 排序候选，再用请求局部 rand.NewSource(seed) 洗牌；按题型预算轮转分配目标；从允许范围挑 3 个无歧义重复的干扰项。禁止全局随机状态。
-  题型模板文字固定为「看字，选出对应的图片」和「看图，选出对应的汉字」；解释为「这道题对应的汉字是『目标字』」。选项 ID 为 kp:<id>，答案记录 ID。
-  缺图缺音、同标签、同图片、目标混入干扰项、题量无法满足都拒绝；结构校验通过仍在预览页提示需要人工确认图义。
-  为不同 seed 的至少 100 次生成运行不变量测试；断言正确答案分布不固定在首项，但不写容易随机失败的精确频率测试。

**通过标准**：引擎无数据库和网络依赖，可用真实素材修订或测试 fixture 运行；同知识点多题不依赖 questions 表。

## Task 5：生成流程、幂等与生命周期

**创建**：`task-admin/backend/internal/materialclient/client.go`、`client_test.go`、`task-admin/backend/internal/qtask/generate.go`、`lifecycle.go`、`lifecycle_test.go`。  
**修改**：`task-admin/backend/internal/qtask/service.go`、`cmd/server/main.go`。

-  测试相同幂等键重复请求只生成一个修订；相同键不同 body 返回 409；冻结失败保留原版本；生成中并发撤回/编辑导致冲突而非覆盖。
-  运行 `go test ./internal/qtask -run 'TestGenerate|TestLifecycle' -v`。
-  创建草稿和生成分两步；POST task 返回可恢复的 task ID。生成 run 的 request_hash 使用规范化请求 JSON，包含 expectedRowVersion；seed 在 run 创建时保存。
-  流程固定为读取候选 → 校验容量 → 冻结所需素材 → 模板生成 → 校验 → 短事务保存完整修订。素材 HTTP 请求在数据库事务外，超时返回结构化 503。
-  进程启动恢复残留 running run 为 failed，记录 interrupted；同一 key 查询失败结果，不再次产生副作用；用户点重试用新 key。
-  替单题保留其余题目的 snapshot，生成新任务修订；新题 fingerprint 必须不同且不重复其他题。整包重生成没有新组合则返回 422。
-  发布要求题量/配比一致、所有媒体可读、人工勾选预览确认；已发布禁止编辑。撤回阻止新领取；再次发布可复用未变修订。
-  从未发布的草稿允许删除；其他只归档。归档阻止新领取并保留历史计划的媒体访问。
-  运行 `go test ./internal/materialclient ./internal/generation ./internal/qtask`。

**通过标准**：网络重试、服务重启和并发编辑不会重复出题或遗留半份题包。

## Task 6：题目后台 API 与兼容入口

**创建**：`task-admin/backend/internal/http/handler_generation.go`、`handler_media.go`、`generation_test.go`。  
**修改**：`task-admin/backend/internal/http/router.go`、`cmd/server/main.go`、`docker-compose.yml`、`task-admin/docker-compose.yml`。

| 方法与路径 | 行为 |
| --- | --- |
| GET /api/v1/question-tasks | 兼容旧列表，支持 kind、status、subject、分页 |
| POST /api/v1/question-tasks | 新 spec 创建空草稿；旧 body 没有 spec 时保留 legacy 抽题行为 |
| GET /api/v1/question-tasks/:id | 当前任务、active/published revision、生成错误 |
| PATCH /api/v1/question-tasks/:id | 草稿出题要求；expectedRowVersion 必填 |
| POST /api/v1/question-tasks/:id/generate | 完整生成，Idempotency-Key 必填 |
| POST /api/v1/question-tasks/:id/items/:seq/replace | 替换单题，创建新修订 |
| PUT /api/v1/question-tasks/:id/order | 传完整 seq 数组，只改变题序，创建新修订 |
| GET /api/v1/question-tasks/:id/revisions/:revisionId | 查看指定历史修订 |
| POST /api/v1/question-tasks/:id/publish | 发布并记录 previewConfirmed=true |
| POST /api/v1/question-tasks/:id/unpublish | 撤回 |
| POST /api/v1/question-tasks/:id/archive | 归档 |
| GET /api/v1/generation-materials/literacy | 后台同源候选代理 |
| GET /api/v1/material-revisions/:revisionId/media/:kind | 后台同源媒体代理 |

-  为 400/404/409/422/503 编写路由测试，再实现对应 handler。错误 envelope 使用 `error: {code,message,details}`，前端兼容旧字符串错误。
-  GET 列表新客户端使用 page/pageSize，旧客户端无分页参数保留数组响应；避免悄悄破坏旧页面。
-  新创建请求为 `{title, spec: <设计第4节对象>}`；title 以外出题字段以 spec 为唯一输入。未提供 spec 的请求走旧逻辑，不通过字段猜测新旧含义。
-  旧 reshuffle 路由保留 legacy 行为；新页面只调 generate。重排必须为现有 seq 的全排列，重复或漏项 422。
-  content base URL Docker 为 `http://content-admin:19091`，本地为 `http://127.0.0.1:19091`；只从环境变量 `APP_CONTENT_ADMIN_URL` 读取。
-  更新 CORS 允许 PUT、Idempotency-Key；所有 ID 为正整数，最大 body 1 MiB；记录 taskId/runId，禁止日志打印凭据和整段图片字节。
-  运行 `go test ./internal/http ./internal/qtask`，确认旧 router 测试仍通过。

**通过标准**：前端不直连跨端口素材服务；新旧请求语义明确；无效 API 不返回 index.html。

## Task 7：题目后台页面

**创建**：`task-admin/frontend/src/pages/TaskDetailPage.tsx`、`components/TaskCreateForm.tsx`、`components/GenerationIssues.tsx`、`pages/TasksPage.test.tsx`、`pages/TaskDetailPage.test.tsx`。  
**修改**：`src/pages/TasksPage.tsx`、`src/App.tsx`、`src/layout/AppShell.tsx`、`src/api/qtask.ts`、`src/api/qtaskTypes.ts`、`src/quiz/qtaskToQuiz.ts`、`src/quiz/QuizQuestionRow.tsx`、`src/styles/app.css`、`index.html`。

-  先写用户操作测试：设置 8 题及 4+4 配比、显示素材缺口、生成后导航详情、刷新详情保留任务、重复点击不发两次生成、发布失败保留题目。
-  运行 `npm test -- src/pages/TasksPage.test.tsx src/pages/TaskDetailPage.test.tsx`。
-  顶部名称改为题目后台；列表显示任务类型、知识范围、题量、状态、更新时间。筛选包含普通/复习、草稿/发布/归档，首版科目仅识字。
-  新建表单提供识字组、目标字多选、题量、两型数量、干扰项范围；实时呈现估算容量，实际生成仍以服务端为准。
-  使用 `/tasks/:id` 详情路由取代 selectedId-only 模式；详情展示题目、来源素材、生成错误和历史修订选择器。技术字段放折叠详情，不干扰常用操作。
-  复用已有预览组件，增加 snapshot adapter；支持四选一试答、图片/语音、替换单题、上下移动题序、重生成、勾选预览后发布、撤回和归档。
-  区分「没有新组合」「素材不足」「素材服务不可用」；错误保留表单和上一份题包，生成按钮使用同一次请求键直到请求结果明确。
-  legacy 任务标注「旧题包」并保留原预览；提供「按当前内容导入为新任务」，导入成功仍为草稿并要求重新检查。
-  执行 `npm test`、`npm run build`；浏览器检查 1440px 和 390px 宽度，重点检查长标题、20题列表和错误信息不遮挡操作。

**通过标准**：不用 API 手工调用即可生成和发布题包；页面没有学习时长、掌握率或薄弱点总览。

## Task 8：学习表迁移与领取服务

**创建**：`parent-dashboard/backend/internal/db/migrations/postgres/013_question_task_links.sql`、`migrations/sqlite/013_question_task_links.sql`、`literacy-server/internal/taskbank/service.go`、`service_test.go`、`literacy-server/internal/http/handler_question_tasks.go`。  
**修改**：`parent-dashboard/backend/internal/db/db_test.go`、`literacy-server/internal/http/router.go`、`literacy-server/cmd/server/main.go`、`literacy-server/internal/plan/model.go`、`internal/plan/service.go`。

-  首先确认 013 未被其他正在开发的迁移占用；若已占用，顺延编号并同步本文路径，不覆盖其他迁移。
-  添加来源任务列及 `plan_items.question_version_id`，放宽 legacy question_id 非空约束；添加至少一种题目来源非空的 CHECK。新增 question_attempt_receipts，完整字段见设计第6节。
-  study_plans 增加唯一 `(child_id, task_claim_key)`，task_claim_key 使用可空列，旧行保持 NULL；不可用默认空串导致旧计划唯一冲突。
-  PostgreSQL 修改参考如下；SQLite 重建 plan_items 时显式列出已存在全部列、复制数据、恢复索引，再做 foreign_key_check，不在迁移事务内部切换 foreign_keys。

```sql
ALTER TABLE plan_items ALTER COLUMN question_id DROP NOT NULL;
ALTER TABLE plan_items ADD COLUMN question_version_id BIGINT;
ALTER TABLE plan_items ADD CONSTRAINT plan_items_question_source
  CHECK (question_id IS NOT NULL OR question_version_id IS NOT NULL);
ALTER TABLE study_plans ADD COLUMN source_question_task_id BIGINT;
ALTER TABLE study_plans ADD COLUMN source_question_task_revision_id BIGINT;
ALTER TABLE study_plans ADD COLUMN task_claim_key TEXT;
CREATE UNIQUE INDEX uq_study_plans_task_claim
  ON study_plans(child_id, task_claim_key);
```

-  学习侧共享迁移不创建 task-admin 表也不引用未部署表的 FK。taskbank 启动就绪检查要求 task v2 schema 已存在；缺失时新增端点 503，旧学习端点照常运行。
-  新增 `GET /api/v1/children/:childId/literacy/question-tasks` 和 `POST /api/v1/children/:childId/literacy/question-tasks/:taskId/claim`，claim body 为 `{revisionId, claimKey}`。
-  领取事务锁序统一为孩子 → 任务 → 已有计划查询；检查 revision 当前可发布、未归档、定向孩子匹配。计划 seq_no 在孩子锁内分配，避免并发同日冲突。
-  复制完整 snapshot 与兼容 question_* 列；插入使用可空 QuestionID 的 taskbank 专用持久化结构，禁止把生成题的版本 ID 填进 legacy question_id。
-  plan.Get 新生成题分支从 snapshot 读取，不依赖 JOIN questions 或最新 knowledge_points.title；旧分支保持。新 DTO 明确 questionVersionId 和可空 question.id。
-  添加领取幂等、跨孩子禁止、撤回竞争、刷新顺序稳定、同日并发领取与旧计划读取测试，运行 `go test ./internal/taskbank ./internal/plan ./internal/http`。
-  增加识字同源修订媒体代理；只按已领取快照或可见题包中的 revision 读取所需媒体。

**通过标准**：一个题包能产生独立学习计划；撤回不破坏已有计划；并发领取不重复创建。

## Task 9：快照判题与原子作答回执

**创建**：`literacy-server/internal/practice/task_answer.go`、`task_answer_test.go`、`receipt.go`、`receipt_test.go`。  
**修改**：`literacy-server/internal/practice/service.go`、`internal/practice/service_test.go`。

-  先写回归场景：显示第0项实际对应原始第2项；答错后重试答对；重复首次请求返回首次响应；同clientId用于另一题或另一选项冲突；回执写失败整次学习更新回滚。
-  运行 `go test ./internal/practice -run 'TestTaskAnswer|TestReceipt' -v`。
-  根据 plan_items.question_version_id 进入新判题分支；读取含指针 QuestionID 的本地结构，旧 shared-go.PlanItem 保持字段兼容，不做跨九学科的指针重构。
-  验证 option_order 是完整无重复排列，提交 index 在范围内；从快照 options 映射 selectedOptionId，与 answerOptionId 比较；拒绝损坏快照，不从最新题库补答案。
-  调用既有事务：

```go
learning.AttemptInput{
    ClientID: input.ClientID,
    KpID: snapshot.KpID,
    QuestionID: nil,
    SkillCode: snapshot.SkillCode,
    IsCorrect: isCorrect,
    CostMs: input.CostMs,
    Source: mastery.SourceQuiz,
    At: now,
}
```

-  以 child_id + client_id 取本事务创建的 attempt ID；同事务写入完整回执及稳定响应、更新 plan item 和 plan 计数。锁序与领取统一为孩子 → 计划 → 题目，再调用 ApplyOne 的同孩子锁。
-  幂等冲突比较 plan_item_id 和 selectedOptionId；重复发送时 costMs 可因前端重试变化，使用第一次已提交值，不因此重复计数。
-  回执保存错误尝试，即使该题最终正确也不覆盖历史错误。完成计划沿用原 Finish，不在作答事务内同步调用题目后台。
-  运行 `go test ./internal/practice ./internal/plan`；shared-go 运行 `go test ./...`，确认不改变掌握度算法或奖励规则。

**通过标准**：任务后台预览不产生 attempts；孩子真实作答恰好一次更新统计，回执能还原具体错选。

## Task 10：孩子端练习任务入口与冻结媒体呈现

**创建**：`literacy-app/src/pages/QuestionTasksPage.tsx`、`QuestionTasksPage.test.tsx`、`src/pages/TaskPractice.test.tsx`。  
**修改**：`src/App.tsx`、`src/pages/HomePage.tsx`、`src/pages/PracticePage.tsx`、`src/pages/ResultPage.tsx`、`src/api/literacy.ts`、`src/api/types.ts`、`src/store/pendingAnswerStore.ts`（仅确有必要时）。

-  写行为测试：从任务列表开始、继续未完成任务、完成后再练、刷新继续、媒体来自 revision、不泄露答案字段、不以示例题兜底真实任务。
-  执行 `npm test -- src/pages/QuestionTasksPage.test.tsx src/pages/TaskPractice.test.tsx`，失败后实现。
-  首页保留原四张题型卡，增加「练习任务」入口；`/tasks` 显示普通练习与复习题包，默认未完成/新任务优先。
-  开始操作生成并保留 claimKey，未知请求结果下重试同一个 key；成功后导航已有 `/practice/:planId`。已完成点再练才生成新 key。
-  PracticePage 对 task snapshot 使用题干、options 的固定 media refs；绝不再用 kpId 获取最新图片/音频。旧题型和手写分支保留。
-  无图片/音频时显示可重试状态；不能把看图选字自动变成直接显示答案字。后台故障不阻止已领取题包读取本地数据库快照。
-  结果页保留题目顺序、真实作答和反馈，不展示后台生成参数；移除展示 DTO 中的答案和内部解释，答案只在服务端反馈或已完成复盘中按原规则提供。
-  执行 `npm test`、`npm run build`；浏览器验证 iPad 横屏、刷新、断网恢复和任务做完后的结果页。

**通过标准**：孩子能通过真实入口完成后台生成的题目；素材更新后已领取任务不变化。

## Task 11：手动复习策略与页面

**创建**：`task-admin/backend/internal/review/types.go`、`sources.go`、`policy.go`、`service.go`、`policy_test.go`、`service_test.go`、`task-admin/backend/internal/http/handler_review.go`、`task-admin/frontend/src/components/ReviewTaskDialog.tsx`、`ReviewTaskDialog.test.tsx`。  
**修改**：`task-admin/backend/internal/http/router.go`、`cmd/server/main.go`、`task-admin/frontend/src/pages/TaskDetailPage.tsx`、`src/api/qtask.ts`、`src/api/qtaskTypes.ts`。

-  先测试：错后答对仍可复习、同知识点两种题型不合并、相同错误组合去重、无错题不建任务、缺变式明确失败、同请求返回同任务、只乱序不算变式。
-  运行 `go test ./internal/review -v`，失败后实现设计第10节的完整 `wrong-answer-v1` 策略。
-  增加 `GET /api/v1/question-tasks/:id/review-sources?childId=1`，仅返回该来源任务已完成计划的 planId、completedAt、eligibleWrongCount；不返回学科健康度。
-  增加 `POST /api/v1/question-tasks/:id/review-preview`，body `{childId, sourcePlanId, targetCount, mode}`；mode 为 mixed 或 original_only。返回容量、分组、缺口与将采用的来源回执 ID。
-  增加 `POST /api/v1/question-tasks/:id/review-tasks`，同 body 加幂等键；后端重新验证来源，而非信任前端 preview。创建 review 草稿并保存 review_sources 和 target_child_id。
-  对 mixed 先各错题组一原题再分配变式，错组超过题量时按优先级取前 N 组；高优先级选择规则固定可测。原题直接引用历史快照，变式走当前素材修订与模板校验。
-  preview 容量不足显示「只能生成 N 题」，可以明确修改题量或点「只重练原题」；不静默降级、不生成空草稿。
-  对已归档/媒体缺失来源保留追溯说明，无法可靠恢复时明确不可用；不从题目字面猜孩子选项。
-  页面在任务详情提供生成复习任务对话框；成功进入新复习任务详情，显示来源原题、实际错选和变式关系，这些信息限于本次出题依据。
-  运行 `go test ./internal/review ./internal/http`；前端执行 `npm test -- src/components/ReviewTaskDialog.test.tsx` 和 `npm run build`。

**通过标准**：从一份实际完成的练习产生可发布、可领取、可再次作答的复习任务。

## Task 12：历史读取兼容与旧题包导入

**创建**：`task-admin/backend/internal/qtask/legacy_import.go`、`legacy_import_test.go`。  
**修改**：`parent-dashboard/backend/internal/service/plan_review.go`、`plan_test.go`、`diagnosis-admin/backend/internal/diagnosis/service.go`、`service_test.go`、`internal/testdb/fixture.go`、`task-admin/backend/internal/http/router.go`。

-  先写混合 fixture：同一孩子具有旧 questions 题和新 question_versions 题；进度历史详情题数不减少；知识点档案的新错选不为空；旧数据呈现保持原样。
-  运行对应 service 测试，确认 INNER JOIN 导致新题丢失的失败被测试捕获。
-  历史详情先读取 plan snapshot；legacy question_id 为空时仍能读到题目；只为旧 snapshot 缺失数据使用 LEFT JOIN questions 回退。不得让新快照缺失悄悄回退。
-  知识库新 attempts 通过 question_attempt_receipts 的 attempt_id 关联取题型、版本和错选；旧 attempts 保持旧分支；只改数据兼容，不加新总览页面。
-  增加 `POST /api/v1/question-tasks/:id/import`：旧题包按当前可读取内容转换新草稿，冻结所有引用素材并记录 legacy question ID；缺项返回逐题错误，不覆盖旧 task。
-  全仓库搜索 `JOIN questions`、`question_id`、`QuestionID` 对新题涉及的读取路径作一次审计；特别检查 parent-dashboard 历史任务接口，不因新增空 legacy ID 丢题。
-  在 parent-dashboard/backend 执行 `go test ./internal/service ./internal/http ./internal/db`；diagnosis-admin/backend 执行 `go test ./...`；task-admin/backend 执行 `go test ./internal/qtask ./internal/http`。

**通过标准**：新题的学习记录在现有进度和知识库中仍可读；不改变两个后台的产品主语和功能边界。

## Task 13：部署、端到端验收与交付记录

**创建**：`docs/verification/2026-09-12-question-task-generation.md`（实施时填写实际证据，不提前写通过）。  
**修改**：`docker-compose.yml`、`task-admin/docker-compose.yml`、`scripts/docker-up.sh`、`task-admin/README.md`、`content-admin/README.md`、`literacy-app/README.md`、根目录 README 和规划记录。

-  部署顺序：备份当前数据库 → shared 学习表迁移 → content 素材修订 → task-admin v2 → literacy-server → 两个前端及兼容读取服务。只重建受影响服务。
-  在临时 PostgreSQL 数据库做真实迁移、事务与并发测试；重复执行迁移不变更已有行。不要用孩子真实账户写验收作答。
-  隔离验收数据库准备至少 8 个有字图/义图/音频的识字知识点，另加缺图、缺音、同图片干扰项 fixture；演示孩子单独创建。
-  执行以下端到端步骤并记录实际 taskId、revisionId、planId、receiptId 与截图路径：
  1. 后台选择4个目标字，生成8题，确认两型各4题、同组干扰项。
  2. 试答、替换一题、调整题序、发布，确认每次成功变更形成修订。
  3. 孩子领取，刷新后题序与选项不变；在至少2个知识点上故意选错，再完成计划。
  4. 重放一次相同作答请求，attempts、计数和回执数量不增加；换题复用相同 key 返回409。
  5. 素材后台更新原图片/音频，旧版本图片/音频摘要不变；新生成任务能使用新素材。
  6. 从原任务生成复习，看到真实错选依据，原题和变式关系正确，重复提交只有一个复习任务。
  7. 发布复习并完成；其他孩子不能领取；撤回后新领取被拒绝，已领取仍可完成。
  8. 在进度历史作答和知识点档案中读取新题，不丢记录；旧题包与旧计划仍可打开。
  9. 模拟缺素材和素材服务不可用，不出现半份题包；恢复后可重试。
-  后端分别运行 `go test ./...`：task-admin/backend、content-admin/backend、literacy-server、shared-go、parent-dashboard/backend、diagnosis-admin/backend。
-  前端分别运行 `npm test`、`npm run build`：task-admin/frontend、literacy-app。未修改其他前端时不扩展到全仓无关测试。
-  在浏览器检查题目后台桌面与窄屏、孩子端横屏；确认没有 API 404、资源失效和控制台异常。
-  交付记录列明实际完成内容、测试结果、未覆盖能力、访问地址、数据库迁移版本。只将真实通过项勾选。

**回退方案**：关闭新增任务入口和自动复习开关，保留数据库新列、修订、媒体及回执，不做降级删表。已有生成题计划需要保留支持新格式的 literacy-server 和读取服务直至完成；不能直接回滚到会 INNER JOIN 丢题的旧镜像。旧独立练习入口继续可用。

## Task 14：自动复习草稿增量

**创建**：`task-admin/backend/internal/review/worker.go`、`worker_test.go`、`jobs.go`、`task-admin/backend/internal/db/review_jobs.go`。  
**修改**：`task-admin/backend/cmd/server/main.go`、`internal/db/db.go`、`internal/http/handler_review.go`、`docker-compose.yml`、`task-admin/README.md`、任务详情错误展示。

-  前提：Task 13 手动闭环已验收。不把 worker 作为补救手动流程不完整的方案。
-  写 fake clock 测试：关闭时不扫描；同计划多轮扫描只建一个 job；重启恢复；无错题 skipped；复习计划不递归；任务生成失败不影响作答。
-  开关 `APP_AUTO_REVIEW_ENABLED=false` 默认关闭，轮询60秒；首次启用时间持久化为 watermark，只有之后完成的普通任务来源计划进入候选。
-  review_generation_jobs 持久化 child_id、source_plan_id、policy_version、state、attempts、next_attempt_at、lease_until、task_id、error；唯一 child+plan+policy。取任务加锁并租约，进程崩溃租约到期可恢复。
-  使用与手动请求相同的去重计算、`wrong-answer-v1` 和生成服务；默认 mixed，不足时 failed 显示缺口，不自动凑题。成功仅草稿，禁止自动发布。
-  每轮总共最多执行3次；第1次失败后等1分钟，第2次失败后等5分钟，第3次失败转终态。`POST /api/v1/review-generation-jobs/:id/retry` 支持显式新一轮排队，记录重试轮次，仍复用业务去重键。
-  运行 `go test ./internal/review ./internal/http`；隔离环境启用后完成一份普通任务，等待一轮扫描，确认恰好生成一份复习草稿；关闭后停止新扫描。

**通过标准**：自动生成可关闭、可追溯、可恢复，不重放历史全部作答，不递归产生无限复习任务。

## 3. 回归测试矩阵

| 风险 | 必须覆盖的证据 |
| --- | --- |
| 历史媒体变化 | 覆盖当前素材后旧 revision 字节摘要相同 |
| 题型配比错误 | 8题4+4、奇数默认分配、越界和容量不足 |
| 假变式 | 仅乱序指纹不变；新干扰项才构成变式 |
| 同日领取并发 | PostgreSQL 下没有重复计划、seq_no 冲突或丢更新 |
| 判题错位 | 保存的 option_order 映射正确，刷新和回执一致 |
| 重复作答计数 | 同 clientId 返回原回执，统计只增一次 |
| 伪造复习来源 | 跨孩子、未完成计划、其他任务 planId 均拒绝 |
| 事务半成功 | 回执写失败时 attempts/mastery/plan 一起回滚 |
| 旧学习流程退化 | legacy 题包、旧计划、原识字卡和手写测试通过 |
| 历史读取丢题 | 新题旧题混合计划和知识点档案均完整 |
| 网络故障 | 素材服务超时不清空旧题；重复生成不重复写入 |
| 自动任务失控 | 默认关闭、普通来源、持久去重、不自动发布 |

## 4. 后续学科扩展边界

完成上述计划后，每次接入一个学科：先提供素材 capability 与修订，再实现真实模板和孩子端快照 adapter，最后复用回执与复习策略。推荐顺序为拼音、英语词汇、算数，再到短句/成语/科普/古诗/逻辑；具体模板数量以该学科素材与题型实际就绪为准。

这里不把其他8个学科勾选为已支持，也不新增空壳入口。写字、语音作答、开放题、大模型生成及间隔复习日历需要各自的评分或调度设计，不包含在本次首版实现中。

## 5. 完成定义

首版完成必须同时满足 M1–M4，不能仅以改名、增加表单或后端测试通过宣告完成。Task14独立标记完成状态。每项实际验证写入验收文档；用户查看最终效果后提出的调整继续沿本计划追加，避免另起一套不兼容的任务概念。

## 6. 决策记录

| 日期 | 决定 | 原因 |
| --- | --- | --- |
| 2026-09-12 | 保留 task-admin 名称与端口，页面叫题目后台 | 避免部署和调用路径无收益迁移 |
| 2026-09-12 | 独立 question_versions，旧 question_id 可空 | 旧题库 kp+code 唯一，不能承载变式；避免污染既有题目规则 |
| 2026-09-12 | 实际媒体冻结由素材后台负责 | 当前资源会覆盖，URL快照不足以保留历史 |
| 2026-09-12 | 先识字两型闭环，再扩学科 | 复用现成预览与孩子端，同时验证完整数据链路 |
| 2026-09-12 | 手动复习先行，自动只生成草稿 | 先验证来源与变式质量，再增加后台触发 |
| 2026-09-12 | 按用户后续授权完成开发、隔离验收与本地部署 | 六个Go项目和两个前端回归通过，验收作答仅写独立库 |
| 2026-09-12 | 素材兼容真实PNG/JPEG原字节、20MP上限 | 原素材存在JPEG字节但PNG路径，按内容识别而非扩展名拒绝 |
| 2026-09-12 | 新服务按现有包组织，实际完成依据见验收记录 | 避免为匹配计划文件名建立重复服务层 |
