# 算术素材、详情、任务与进度统一实施计划

> 用户已批准设计并要求直接实施。采用 subagent-driven-development 分工、TDD 与完成前验证；在现有共享工作区保留所有未提交修改，不提交、不 reset、不创建另一套服务。

**Goal:** 将 App 现有12种详情迁入素材后台，三个端共用题面；算术任务消费发布素材并冻结题目；进度按真实作答及适用题型展示。

**Architecture:** shared-go/mathcontent 定义发布详情契约和默认内容；素材后台管理草稿及不可变发布修订。math-server 代理已发布目录（短期缓存），App 不领取 task-admin 题包。packages/math-player 渲染同一详情对象；后台试做、App 内容示例不写学习账本。保留现有正式计划作答入口，当前正式必需题型仍为加减法 calc/story、图形 find/name；新增示例不自动扩张掌握要求。

**Tech Stack:** Go/GORM、PostgreSQL/SQLite测试、React/TypeScript、Vitest、Docker Compose。

## 固定契约

素材发布目录 GET `/api/v1/math/details/published` 返回 `{schemaVersion:1,items:MathDetail[]}`；App 经 math-server GET `/api/v1/math/details` 返回现有包裹 `{data:{schemaVersion:1,items:[...]}}`。

`MathDetail`：`id,groupId,title,moduleTitle,learningGoal,rules:string[],revision:number,example:MathExample`。

`MathExample`：`kind` 保留 choice/missing/objects/judgement/audio-shape/shape-name/shape-feature/shape-sort；`prompt:string`；可选 `options:string[],answer:string,groups:string[],statements:string[],buckets:{label:string,items:string[]}[]`。增加可选结构 `operation:'add'|'sub',counts:number[],object:string,shapeKeys:string[],audioUrl:string`。分类桶是目标分组，播放器呈现打散后的候选并允许选择归属；减法明确划去拿走数量。默认12项 ID 与 App 现有 questionTypeDetails 完全一致。题型入口5组是导航，不等于掌握技能。

素材管理 GET `/api/v1/math/details` 返回所有草稿详情及发布状态；PUT `/api/v1/math/details/:id` 接受完整 MathDetail，revision 为乐观锁版本，成功返回新草稿；POST `/api/v1/math/details/:id/publish` 接受 `{revision}`，发布当前草稿。缺素材/无效题面返回400，版本冲突409。首次启动初始化已有12份可验证示例；后续启动不覆盖编辑。

公开示例可含答案，用于本轮不入账的示范和试做；正式计划 API 继续保护答案。题目后台生成的草稿也有管理答案，暂不增加 App 领取接口。

## 任务与验证

### 1. 素材契约与发布（root）
- [x] 新增 `shared-go/mathcontent/{types.go,defaults.json,types_test.go}`：验证12种详情、答案可选项内、分类合法、计数非负、减法不超出原量。
- [x] 新增 `content-admin/backend/internal/math/details.go` 和测试：DB草稿/修订、初始化幂等、乐观锁、发布校验；接 handler/router。
- [x] 用 `go test ./internal/math ./internal/http` 验证：改草稿不改变发布；重复启动不重置；并发旧版本不覆盖新版本。

### 2. 共用题面及 App/素材编辑界面（math_surfaces）
- [x] 新建 `packages/math-player` 的类型、播放器与CSS；四选一/判断/数量图/图形/分类都有真实试做反馈及重置，答案不靠翻页判断。
- [x] App QuestionTypeDetailPage 与题型入口详情读取 math-server 发布目录；缺内容显示原因，不用本地详情兜底。保留5入口与正式计划页；主入口本轮使用素材示例，不从题目后台实时取题。
- [x] 素材 MathPage 新增详情管理：12项直接可见、可编辑文案/例题必要参数、保存草稿和发布、共用预览；不暴露JSON作为普通编辑流程。
- [x] Vitest 检查多宿主渲染一致、减法拿走、分类交互、网络错误、内容更新；build三个使用方。Docker复制共享包。

### 3. 题目后台算术任务（math_tasks）
- [x] 独立算术任务适配，不把 literacy 字段套在算术上。新增 math task API/页面，沿用当前任务后台导航和草稿/发布概念。
- [x] 从素材发布接口读取12详情，支持范围/题型/数量；算术以结构化参数产生合法变式并验证唯一答案，图形题按素材定义生成；题目快照保存来源 id/revision。
- [x] 用共用 MathPlayer 预览、试做；改素材不改变旧任务快照，发布前校验完整性；缺目录不得生成假题。
- [x] 测试生成、冻结、发布、缺素材、后台试做无学习写入；不实现孩子题包领取。

### 4. 进度展示与已有正式作答一致性（math_progress）
- [x] 算术专属页面：四项当前正式技能条按适用目录计算；加法/减法按5/10/20分组，图形单独组；搜索、悬浮和只读抽屉、两态技能标记。
- [x] 汇总和技能使用真实attempts/mastery_skills，卡片所有适用技能掌握才完整；不将12种示例当12个必需技能。
- [x] 核验旧正式计划 calc/story/find/name 作答与页面一致，未正式支持的题型不伪造进度；不改其他学科布局。
- [x] Node/Go测试、前端lint/build和窄屏验证。

### 5. App素材代理（root）
- [x] `math-server/internal/content` 发布目录客户端，超时/格式错误503、TTL60秒；已缓存内容在短暂失败时可读，响应标明 stale；无缓存无本地默认兜底。
- [x] 配置 APP_CONTENT_ADMIN_URL，代理到素材服务；只读GET，不依赖题目后台。
- [x] 测试发布数据、缓存、过期、故障；保留旧生成与正式计划API兼容。

### 6. 审查、集成与原端口部署（root）
- [x] 契约与行为审查；运行涉及服务全部测试和前端构建；只使用隔离测试库/路由模拟，不在正式孩子上作答。
- [x] 备份数据库、记录容器ID/镜像/启动时间；构建 content-admin、task-admin、math-server、math-app、backend。
- [x] 指定服务 `up -d --no-deps --no-build --force-recreate --wait`；原19091/19201/19141/19142/19081，不重启无改动服务或数据库。
- [x] 正式端口只读验证：12详情一致、后台/App同版本、进度读取正常、账本摘要未被示例修改；定向清理旧镜像，保留数据卷。
- [x] 写验收记录，列明详情示例与正式作答边界；不宣称已实现本轮明确暂缓的App题包领取。


## 实施结果（2026-09-12）

上述代码、测试与原端口部署已完成。验收记录：`docs/verification/2026-09-12-math-content-unification.md`。

- [x] 补充音频生成接口、内容摘要寻址、发布前音频对象校验；题干改动自动清除旧音频。
- [ ] 补齐十二份详情的实际题干音频：生产 TTS 上游返回 503 / `No available Grok accounts`，生成请求未成功，不能标记素材音频齐备。需上游账号恢复后重新生成并发布。

本轮边界：十二种详情是素材示例；正式进度仍按 calc/story/find/name 四项适用技能统计。App 示例不从题目后台领取题包，不写学习账本。
