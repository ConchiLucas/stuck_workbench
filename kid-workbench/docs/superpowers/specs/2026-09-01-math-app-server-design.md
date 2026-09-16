# 算数孩子端与算数服务设计

## 目标

在 `kid-workbench` 中新增面向孩子的独立算数学习前端 `math-app`，以及只服务该前端的 Go 后端 `math-server`。

第一版面向 iPad 横屏，覆盖 20 以内加法、20 以内减法、认识图形、错题重试、练习结算和掌握度更新。内容后台、算数服务和家长看板之间不进行 HTTP 调用，只共享 PostgreSQL 中约定的数据表，以及 MinIO 中已经生成的题目 TTS 音频。

## 已确认的产品边界

- `content-admin`（`http://localhost:19091`）负责维护算数内容、题目和题目 TTS 音频。
- `math-app` 只调用 `math-server`，不直接访问内容后台、家长看板、PostgreSQL 或 MinIO。
- `math-server` 直接读取 PostgreSQL 中的算数内容和题目，并写入计划、作答、统计、掌握度和奖励数据。
- `parent-dashboard`（`http://localhost:19081`）直接从共享数据库读取算数学习结果。
- 三个后端没有运行时 HTTP 依赖；内容后台或家长后端停止后，算数练习仍能使用已经发布的内容完成。
- 第一版默认孩子 ID 为 `1`，不新建账号或登录系统。
- 所有已发布、可组卷的算数题目保证具备对应 TTS 音频。系统不设计 TTS 缺失降级，也不调用浏览器 TTS。

## 非目标

第一版不包含：

- 内容编辑、题目编辑、TTS 生成或素材生成。
- 孩子录音、语音识别或发音评分。
- 拖拽运算、手写算式或复杂教学动画。
- 多人在线对战或排行榜。
- 新账号和登录系统。
- 将五个学科抽象成通用低代码学习平台。
- 修改内容后台中识字、拼音、英语或科普模块的现有行为。

## 系统架构

```text
content-admin :19091
  ├─ 写 subjects / modules / knowledge_points
  ├─ 写 questions / math_assets
  └─ 写 MinIO 题目 TTS 音频

                         PostgreSQL + MinIO
                         ▲              ▲
                         │              │
math-app :19142 ──HTTP──> math-server :19141
                         │
                         ├─ 读算数内容、题目、题目音频和孩子进度
                         └─ 写计划、作答、掌握度、统计和奖励流水

parent-dashboard :19081
  └─ 读共享的计划、作答、掌握度和统计数据
```

建议端口：

| 服务 | 端口 | 说明 |
| --- | ---: | --- |
| `math-server` | `19141` | 算数孩子端 API |
| `math-app` | `19142` | Vite 开发端口及 Docker 暴露端口 |

`19111` 和 `19112` 保留给规划中的拼音服务，学科服务按端口对排列。

## 项目结构

```text
kid-workbench/
├── literacy-app/
├── math-app/
│   ├── src/
│   │   ├── api/
│   │   ├── audio/
│   │   ├── components/
│   │   ├── features/
│   │   │   ├── home/
│   │   │   ├── map/
│   │   │   ├── module/
│   │   │   ├── practice/
│   │   │   └── result/
│   │   ├── pages/
│   │   └── styles/
│   ├── Dockerfile
│   ├── nginx.conf
│   └── package.json
├── math-server/
│   ├── cmd/server/
│   ├── internal/
│   │   ├── asset/
│   │   ├── catalog/
│   │   ├── home/
│   │   ├── http/
│   │   ├── plan/
│   │   ├── practice/
│   │   └── progress/
│   ├── Dockerfile
│   └── go.mod
└── shared-go/
    ├── learning/
    ├── mastery/
    ├── model/
    └── repository/
```

`math-app` 和 `math-server` 是独立项目、独立构建、独立进程和独立容器。`shared-go` 是编译期本地 Go module，不是需要启动的运行服务。

`shared-go` 只承载跨学科成立的掌握度、幂等作答、统计和奖励事务，不承载算数题型、页面 DTO 或素材规则。

## 数据所有权

共享数据库不表示任意服务都可以修改任意表：

| 数据 | 主要写入方 | 主要读取方 |
| --- | --- | --- |
| `subjects`、`modules`、`knowledge_points` | 内容初始化或内容后台 | 全部系统 |
| `questions` | 内容后台 | `math-server` |
| `math_assets` | 内容后台 | `math-server` |
| MinIO 算数题目音频 | 内容后台 | `math-server` |
| `study_plans`、`plan_items` | `math-server` | `math-server`、家长看板 |
| `attempts` | `math-server` | `math-server`、家长看板 |
| `mastery_skills`、`mastery_states` | `math-server` 通过共享学习事务 | 学科服务、家长看板 |
| `daily_stats`、`flower_ledger` | `math-server` 通过共享学习事务 | 家长看板 |

`math-server` 的所有内容、题目、计划、进度和掌握度查询必须限制 `subjects.code = 'math'`，防止读取或修改其他学科数据。

## 算数内容范围

第一版覆盖当前目录中的三个模块：

| 模块代码 | 名称 | 内容 |
| --- | --- | --- |
| `add10` | 20 以内加法 | `a + b <= 20` |
| `sub10` | 20 以内减法 | 被减数不超过 20，差不小于 0 |
| `shape` | 认识图形 | 圆形、正方形、长方形、三角形、椭圆形、梯形、菱形、五角星 |

现有模块代码 `add10`、`sub10` 与显示名称不完全一致。第一版保留数据库代码以避免迁移影响，页面始终显示“20 以内加法”和“20 以内减法”。

加减法数量较多，不在孩子端平铺所有知识点。地图按照数值范围划分阶段：

| 阶段代码 | 加法归属 | 减法归属 |
| --- | --- | --- |
| `within5` | 和不超过 5 | 被减数不超过 5 |
| `within10` | 和为 6–10 | 被减数为 6–10 |
| `within20` | 和为 11–20 | 被减数为 11–20 |

每个知识点只属于一个阶段，阶段之间不重复计数。`shape` 使用单一阶段 `basic-shapes`。

## 题型与视觉协议

第一版使用四种技能：

| 模块 | 技能代码 | 含义 |
| --- | --- | --- |
| `add10`、`sub10` | `calc` | 看算式选答案 |
| `add10`、`sub10` | `story` | 看数量图选答案 |
| `shape` | `find` | 听名称，在四个图形中选择 |
| `shape` | `name` | 看图形，在四个名称中选择 |

`calc` 只显示算式，不展示苹果、草莓或其他可数物，避免答案提示。`story` 不显示算式，只用实物数量表达“合起来”或“拿走”。

### 题型地图与扩展边界

孩子端提供独立的“算数题型地图”，用于回答“算数可以有哪些题型”并支持后续内容排期。题型地图是产品分类目录，不是已发布题库清单：

- “内容题型”表示练习的数学概念或数量关系，例如凑十法、购物找零、轴对称判断。
- “交互方式”表示如何作答，例如单选、拖拽排序、画线、口头回答；同一内容换交互方式不重复计算题型。
- 目录覆盖数感、比较估算、加减、乘除、分数小数百分数、代数启蒙、应用题、几何、测量、时间、人民币、统计、概率组合、逻辑、口算策略和综合探究。
- 只有 `calc`、`story`、`find`、`name` 对应的四种目录条目标记“当前可练”；其他条目只代表可扩展方向，不进入组卷。
- 新题型接入真实练习时，仍需逐项补充题目数据、视觉协议、TTS、掌握度技能和服务端白名单，不能仅依靠前端目录开放。

题目视觉使用有判别字段的结构化数据，不把 `math_assets.payload` 原样返回前端：

```json
{
  "kind": "add",
  "leftCount": 2,
  "rightCount": 5,
  "object": "apple"
}
```

减法视觉显示原数量，并将拿走部分划掉或淡化。图形使用受控 SVG key：

```text
circle, square, rect, triangle, oval, trapezoid, rhombus, star
```

前端不使用 Unicode 图形或 emoji 表示几何形状，避免不同平台字体差异。

## 题目 TTS 音频

TTS 音频必须绑定题目变体 `question_id`，不能只绑定知识点 `kp_id`。同一算式知识点至少存在两种不同朗读：

- `calc`：如“二加五等于几”。
- `story`：如“一共有几个”或“还剩几个”。

内容后台发布题目前，为每个可练 `question_id` 生成对应音频，并把稳定的 MinIO object key 保存到 `questions.media_url`。推荐对象路径：

```text
math/questions/<questionId>.mp3
```

数据库不再把带 `19091` host 的完整 URL 作为孩子端运行合同。`math-server` 读取 object key，并通过计划题目范围内的自身接口输出音频：

```http
GET /api/v1/children/:childId/math/plans/:planId/items/:itemId/audio.mp3
```

`math-server` 只读取已生成音频，不调用 TTS。`math-app` 不调用浏览器 TTS。已发布、可组卷题目具有音频是内容数据契约，不设计音频缺失分支。

模块学习页通过受控的孩子、知识点和题型路径播放已发布音频，不接受 question ID 或 object key：

```http
GET /api/v1/children/:childId/math/items/:kpId/audio/:code.mp3
```

`math-server` 必须验证孩子存在、知识点属于 `math`、题型属于该知识点，并只允许 `calc|story|find|name`。计划内答题仍使用更严格的计划题目音频路径。

MinIO 临时不可用与数据缺失是不同问题。临时不可用时返回 `503 audio_unavailable`，前端提供重新播放，不改用其他朗读方式。

## 掌握度技能

算数按知识点和技能分别记录掌握度：

```text
math + add10/sub10 → [calc, story]
math + shape       → [find, name]
```

`shared-go` 当前按学科返回固定技能列表的接口需要扩展为按“学科 + 模块”返回技能列表。该扩展必须保持识字和拼音现有行为不变。

知识点汇总规则：

1. 任一必需技能为 `shaky`，知识点为 `shaky`。
2. 所有必需技能都是 `mastered` 或 `review_due` 时，知识点才算完成。
3. 完成技能中存在 `review_due` 时，知识点为 `review_due`；否则为 `mastered`。
4. 有部分进展但未全部完成时，知识点为 `learning`。
5. 没有技能记录时为 `not_started`。

一次作答在同一数据库事务中完成：

1. 校验孩子、计划、计划题目和选项下标。
2. 使用计划题目快照判题。
3. 用唯一 `client_id` 写入 `attempts`。
4. 更新对应技能的 `mastery_skills`。
5. 汇总更新知识点级 `mastery_states`。
6. 更新 `daily_stats`。
7. 新掌握时写入小红花和 `flower_ledger`。
8. 更新 `plan_items` 和 `study_plans`。

第一次答错立即写入尝试和掌握度；允许再试一次，第二次答对不能覆盖第一次错误。相同 `client_id` 重复提交返回首次处理结果，不重复计分或发奖励。

## 计划与题目快照

仅在 `plan_items` 保存 `question_id` 不能保证练习稳定。当前内容灌库可能按知识点和题型代码原地更新同一题目，使题目 ID 不变但题干、选项、答案或音频发生变化。

给 `plan_items` 增加 `question_snapshot`，创建计划时保存完整服务端题目快照：

```json
{
  "questionId": 123,
  "code": "calc",
  "stem": "2 + 5 = ?",
  "options": ["5", "6", "7", "8"],
  "answerIndex": 2,
  "visual": {
    "kind": "equation",
    "a": 2,
    "b": 5,
    "operator": "+"
  },
  "audioObjectKey": "math/questions/123.mp3"
}
```

PostgreSQL 使用 `JSONB`，SQLite 测试使用 `TEXT`，由共享模型提供统一访问方式。完整快照只保存在服务端；返回前端时删除 `answerIndex`、`audioObjectKey` 和其他内部字段。

给 `study_plans` 增加可空字段：

- `plan_kind`：`daily`、`module` 或 `review`。
- `module_code`：专项练习模块。
- `stage_code`：专项练习阶段。

迁移必须向前兼容，现有计划保留空值并按旧行为展示。

## 后端 API

### 首页与进度

```http
GET /api/v1/children/:childId/math/home
GET /api/v1/children/:childId/math/progress
```

首页返回今日计划、待复习数量，以及加法、减法和图形进度摘要。进度按模块、阶段和知识点返回必需技能状态。

### 内容与模块

```http
GET /api/v1/math/modules
GET /api/v1/math/modules/:moduleCode
GET /api/v1/math/modules/:moduleCode/stages/:stageCode
```

内容接口只返回孩子学习展示所需字段，不返回原始数据库 payload、正确答案、内部 object key 或外部服务地址。

模块学习页需要朗读时，前端根据返回的 `kpId` 和允许的题型代码构造受控音频路径；图形名称默认使用 `find` 题型音频。

### 练习计划

```http
POST /api/v1/children/:childId/math/plans
GET  /api/v1/children/:childId/math/plans/:planId
POST /api/v1/children/:childId/math/plans/:planId/start
POST /api/v1/children/:childId/math/plans/:planId/items/:itemId/answer
POST /api/v1/children/:childId/math/plans/:planId/finish
```

创建每日练习：

```json
{ "kind": "daily" }
```

创建模块专项练习：

```json
{
  "kind": "module",
  "moduleCode": "add10",
  "stageCode": "within10"
}
```

创建复习练习：

```json
{ "kind": "review" }
```

每日练习默认 10 题，初始配额为 4 道加法、4 道减法和 2 道图形。某模块候选不足时由其他模块补齐。模块内部按“待复习、需巩固、学习中、新题”的优先级选择，并尽量均衡该模块的两个技能。

模块专项练习只从指定模块和阶段选 10 题，并尽量平均两个技能。复习练习最多 10 题，只选择 `review_due`、`shaky` 或近期答错的知识点；没有足够复习题时返回实际数量，不用新题补齐。

计划响应中的题目示例：

```json
{
  "itemId": 456,
  "questionId": 123,
  "code": "calc",
  "stem": "2 + 5 = ?",
  "options": [
    { "label": "5" },
    { "label": "6" },
    { "label": "7" },
    { "label": "8" }
  ],
  "visual": {
    "kind": "equation",
    "a": 2,
    "b": 5,
    "operator": "+"
  },
  "audioUrl": "/api/v1/children/1/math/plans/789/items/456/audio.mp3"
}
```

作答请求：

```json
{
  "clientId": "stable-client-generated-uuid",
  "optionIndex": 2,
  "costMs": 4100
}
```

作答响应包含是否正确、是否允许重试、提交后可展示的正确选项下标、当前题目状态、技能及知识点掌握状态和计划进度。

## math-app 页面设计

### 路由

```text
/                         首页
/map                      算数地图
/types                    算数题型地图
/module/:moduleCode       模块学习页
/practice/:planId         答题页
/result/:planId           结算页
```

### 首页

- 今日算数任务卡。
- 待复习题目数量。
- 加法、减法和图形进度摘要。
- “开始今天的练习”、“算数题型地图”和“算数地图”入口，其中题型地图重点说明可扩展范围。

### 算数题型地图

- 按数学领域展示完整内容题型目录、领域数量和题型总数。
- 支持领域筛选和中文关键词搜索。
- 使用醒目标记区分“当前可练”和未来候选题型。
- 页面底部单列交互方式，并解释内容题型与作答方式的区别。

### 算数地图

- 三个区域：加法岛、减法岛、图形屋。
- 加减法展示 5 以内、10 以内、20 以内三个互斥进度阶段。
- 图形屋展示 8 个具体图形。
- 展示未开始、学习中、需巩固、待复习和已掌握状态。

### 模块学习页

- 加法和减法展示当前阶段的数量关系示例，不列出全部算式。
- 使用简单点击演示“合起来”和“拿走”，不加入拖拽或复杂动画。
- 图形模块展示 8 个 SVG 图形、名称和播放按钮。
- 学习和试听不写入作答记录。

### 答题页

- iPad 横屏一题一屏，不允许滚动看到其他题。
- 左侧约 45% 显示题干、算式或数量图，右侧约 55% 显示 2×2 四个大选项。
- 顶部显示计划进度和退出按钮。
- 第一次由孩子点击“开始”解锁 Safari 音频，后续每题自动播放预生成 TTS。
- 预加载当前题和下一题音频；同一时刻只播放一个音频。
- 点击选项后立即锁定，防止连续点击。
- 第一次答错时提示“再想想”并允许一次重试。
- 第二次仍错时显示正确答案，再进入下一题。
- 网络提交失败时保留当前题、选择和原 `clientId`，由孩子明确重试。
- 刷新、退出再进入或换设备后恢复相同计划、题目和选项顺序。

### 结算页

- 完成题数、正确题数、星星和小红花。
- 展示需要再练的阶段或图形，不展示复杂百分比或算法细节。
- 提供“回到首页”和“再练错题”入口。

## iPad 与视觉要求

- 主要验收尺寸为横屏 `1024 × 768`，兼容更高分辨率横屏。
- 主要触控区域不小于约 `56px`。
- 不依赖 hover 才能发现操作。
- 处理安全区域和添加到主屏幕后全屏显示。
- 算数端建立数字、运算符和方格纸视觉语言，不复制内容后台界面。
- `calc` 使用清晰的大号数字与运算符；`story` 使用可数且不拥挤的实物；图形使用统一 SVG。
- 尊重 `prefers-reduced-motion`，反馈动画不得阻碍答题。

## 错误处理

| 场景 | 服务端行为 | 前端行为 |
| --- | --- | --- |
| 数据库暂时不可用 | `503 database_unavailable` | 保留题面、选择和 `clientId`，允许重试 |
| MinIO 暂时不可用 | `503 audio_unavailable` | 提供重新播放，不使用浏览器 TTS |
| 重复提交 | 返回首次幂等结果 | 不重复播放奖励动画 |
| 计划已完成 | 返回当前结算 | 跳转结算页 |
| 访问其他孩子的计划 | 返回 `404` | 回到首页，不泄露资源是否存在 |
| 内容在练习中更新 | 继续使用计划快照 | 当前练习不变 |
| 网络中途断开 | 不提交不完整事务 | 保留当前状态并重试原请求 |

## 测试策略

### shared-go

- 验证 `add10`、`sub10` 的必需技能为 `calc`、`story`。
- 验证 `shape` 的必需技能为 `find`、`name`。
- 验证两个技能都完成后知识点才完成。
- 验证答对、答错、两次尝试和复习到期。
- 验证相同 `client_id` 不重复写入、计分或发奖励。
- 保持识字和拼音技能映射及汇总测试通过。

### math-server

- 验证所有查询限制 `subject = math`。
- 验证每日组卷的模块配额、候选不足补齐和技能均衡。
- 验证 `within5`、`within10`、`within20` 边界互斥且完整。
- 验证计划创建后题目快照和选项顺序稳定。
- 验证取题响应不包含正确答案和内部 object key。
- 验证内容后台更新原题后，旧计划仍返回原快照。
- 验证作答事务全部成功或全部回滚。
- 验证不能访问其他孩子的计划或音频。
- 使用 PostgreSQL 集成测试覆盖 JSONB、事务和生产 SQL 行为。
- 使用 MinIO 接口替身测试音频读取，不在测试中调用真实 TTS。

### math-app

- 使用 Vitest 和 Testing Library 测试首页、地图、模块、答题和结算。
- 测试选择锁定、第一次答错重试、第二次答错揭示答案。
- 测试重复点击、重复请求和幂等结果。
- 测试网络失败后使用原 `clientId` 重试。
- 测试刷新后恢复相同计划。
- 测试同一时间只播放一个音频，切题或离开时停止上一段。
- 使用 Playwright 的 iPad 横屏视口验收主流程。
- 验证所有主要操作可以通过触摸完成。

### 跨系统验收

1. 内容后台发布加法、减法和图形题目及对应题目 TTS。
2. `math-server` 不调用内容后台 HTTP API，直接读取 PostgreSQL 和 MinIO。
3. 孩子分别完成每日练习、模块专项和复习计划。
4. 数据库出现匹配的计划、作答、技能掌握度、知识点掌握度、每日统计和奖励流水。
5. 家长看板不调用 `math-server`，仍能展示新产生的算数结果。
6. 停止内容后台和家长后端后，再完成一份使用已发布内容的算数练习。

## 部署

- 根 `kid-workbench/docker-compose.yml` 增加 `math-server` 和 `math-app`。
- `math-server` 暴露 `19141`，使用现有 `APP_DB_*` 和 MinIO 环境变量规范。
- `math-app` 暴露 `19142`，Nginx 将同源 `/api` 代理到 `math-server:19141`。
- `math-app` 生产环境只配置一个 API 来源。
- `math-app` 仅依赖健康的 `math-server`。
- `math-server` 依赖 PostgreSQL 和 MinIO，不依赖 `backend` 或 `content-admin` 容器。
- `math-app` 提供 PWA manifest 和适合 iPad 主屏幕的图标。
- 数据库迁移向前兼容，不要求先停止内容后台或家长看板。
- 健康检查区分进程存活与 PostgreSQL、MinIO 依赖就绪状态。

## 实施顺序

1. 完成当前 `shared-go` 的掌握度和共享学习事务提取，保持家长后端行为不变。
2. 扩展 `shared-go`，支持按学科和模块返回必需技能。
3. 增加题目级 TTS object key、计划题目快照和计划类型字段迁移。
4. 调整内容后台的算数发布流程，为每个题目变体生成 TTS。
5. 建立 `math-server`，完成健康检查、目录、进度和音频读取。
6. 完成组卷、题目快照、服务端判题、幂等作答和掌握度事务。
7. 建立 `math-app`，完成首页、地图和模块学习页。
8. 完成答题、断点恢复和结算页。
9. 接入根 Compose，执行 PostgreSQL、MinIO 和家长看板跨系统验收。

## 完成标准

- `math-app` 和 `math-server` 是独立项目、独立构建和独立运行服务。
- 20 以内加法、20 以内减法和认识图形均可学习和练习。
- `calc`、`story`、`find`、`name` 分别更新正确的技能掌握度。
- 所有题目使用预生成 TTS，不调用浏览器 TTS。
- `math-app` 只请求 `math-server`。
- `math-server` 不请求 `19091` 或 `19081`。
- 内容后台或家长后端停止后，已发布算数内容仍可完成练习。
- 计划题目在内容更新、刷新或换设备后保持不变。
- 一次作答原子更新计划、尝试、技能掌握度、知识点掌握度、每日统计和奖励。
- 重复提交不会重复计分或发放奖励。
- 家长看板可以直接从共享数据库看到算数学习结果。
