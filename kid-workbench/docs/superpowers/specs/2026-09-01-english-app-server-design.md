# 英语孩子端与英语服务设计

## 目标

在 `kid-workbench` 中新增面向孩子的独立英语学习前端 `english-app`，以及只服务该前端的 Go 后端 `english-server`。

第一版面向 iPad 横屏场景，覆盖英语单词认读、主题浏览、听音选词、听音选图、错题重试、练习结算和掌握度更新。内容后台、英语服务和家长看板之间不进行 HTTP 调用，只共享 PostgreSQL 中约定的数据表，以及 MinIO 中已经生成的英语素材。

## 已确认的产品边界

- `content-admin`（`http://localhost:19091/english`）负责维护英语知识点、题目、拼写图、义图和音频。
- `english-app` 只调用 `english-server`，不直接访问内容后台、家长看板、PostgreSQL 或 MinIO。
- `english-server` 直接读取 PostgreSQL 中的英语内容、素材记录和题目，并写入英语计划、作答、统计及掌握度数据。
- `parent-dashboard`（`http://localhost:19081`）直接从共享数据库读取英语答题结果和掌握度，用于家长查看。
- 三个后端之间没有运行时 HTTP 依赖；任一后台服务停止，不应阻止另外两个服务访问数据库中已经存在的数据。
- `english-app` 与 `english-server` 分别拥有独立的构建、部署、端口和健康检查。

## 非目标

第一版不包含以下能力：

- 键盘输入拼写或字母拼词。
- 孩子录音、发音评分或语音识别。
- 英语短句、语法和开放式对话练习。
- 英语内容编辑、题目编辑、TTS 或图片生成。
- 多人在线对战。
- 将全部学科一次性抽象为通用低代码学习平台。
- 修改内容后台中识字、拼音、算术或科普模块的现有行为。

## 方案选择

采用每学科独立前端和独立后端的方案：

```text
english-app + english-server + shared-go
```

不采用多学科共用单个学习后端，避免后端积累大量学科分支。不采用英语前端直接请求内容后台或家长后端，避免运行时耦合和职责混杂。

## 系统架构

```text
content-admin :19091
  ├─ 写 subjects / modules / knowledge_points
  ├─ 写 questions / english_assets
  └─ 写 MinIO 英语拼写图、义图和音频

                          PostgreSQL + MinIO
                          ▲              ▲
                          │              │
english-app :19122 ──HTTP──> english-server :19121
                          │
                          ├─ 读英语内容、题目、素材和孩子进度
                          └─ 写计划、作答、掌握度、统计和奖励流水

parent-dashboard :19081
  └─ 读共享的计划、作答、掌握度和统计数据
```

建议开发端口为：

| 服务 | 端口 | 说明 |
| --- | ---: | --- |
| `english-server` | `19121` | 英语孩子端 API |
| `english-app` | `19122` | Vite 开发端口及 Docker 暴露端口 |

## 项目结构

```text
kid-workbench/
├── literacy-app/
├── pinyin-app/
├── pinyin-server/
├── english-app/
│   ├── src/
│   │   ├── api/
│   │   ├── audio/
│   │   ├── components/
│   │   ├── features/
│   │   │   ├── home/
│   │   │   ├── words/
│   │   │   ├── practice/
│   │   │   ├── progress/
│   │   │   └── result/
│   │   ├── pages/
│   │   └── styles/
│   ├── Dockerfile
│   └── package.json
├── english-server/
│   ├── cmd/server/
│   ├── internal/
│   │   ├── asset/
│   │   ├── catalog/
│   │   ├── home/
│   │   ├── http/
│   │   ├── plan/
│   │   ├── practice/
│   │   ├── progress/
│   │   └── repository/
│   ├── Dockerfile
│   └── go.mod
└── shared-go/
    ├── learning/
    ├── mastery/
    ├── model/
    └── repository/
```

`shared-go` 是编译期本地 Go module，不是运行服务。它只承载已经在多个学科中共同成立的学习事务、掌握度、统计和奖励规则，不承载英语页面模型、英语素材规则或英语题目生成逻辑。

## 独立性与运行边界

- `english-app` 有独立 `package.json`、Vite 配置、Dockerfile 和 PWA 配置。
- `english-server` 有独立 `go.mod`、启动入口、Dockerfile 和健康检查。
- `english-app` 停止不影响其他学科、内容后台或家长看板。
- `english-server` 停止只影响英语孩子端。
- 内容后台停止时，已经生成的英语内容和素材仍可被 `english-server` 使用。
- 家长看板停止时，英语孩子端仍可继续学习和写入结果。
- 服务共享数据库 Schema 和 MinIO 素材，但不共享运行进程。

## 数据所有权

共享数据库不代表任意服务都可以修改任意表。各类数据的主要写入方如下：

| 数据 | 主要写入方 | 主要读取方 |
| --- | --- | --- |
| `subjects`、`modules`、`knowledge_points` | 内容初始化或内容后台 | 全部系统 |
| `questions` | 内容初始化或内容后台 | `english-server` |
| `english_assets` | 内容后台 | `english-server` |
| MinIO 英语素材 | 内容后台 | `english-server` |
| `study_plans`、`plan_items` | `english-server` | `english-server`、家长看板 |
| `attempts` | `english-server` 通过共享学习事务写入 | `english-server`、家长看板 |
| `mastery_skills`、`mastery_states` | 共享学习事务 | 学科服务、家长看板 |
| `daily_stats`、`flower_ledger` | 共享学习事务 | 家长看板 |

`english-server` 所有内容和学习数据查询必须限制 `subjects.code = 'english'` 或计划学科为 `english`，防止读取或修改其他学科数据。

## 英语内容模型

第一版沿用当前 20 个主题、每组 10 个单词的英语知识点。每个英语知识点通过 `english_assets.kp_id` 关联 `knowledge_points.id`。

课程语义保存在 `knowledge_points.payload`，建议结构为：

```json
{
  "meaningZh": "猫",
  "phonetic": "/kæt/",
  "partOfSpeech": "noun",
  "example": "This is a cat.",
  "exampleMeaningZh": "这是一只猫。"
}
```

第一版要求：

- `meaningZh` 必填，并经过人工整理或审核。
- `phonetic` 可选，缺失时前端不展示音标区域。
- `partOfSpeech`、`example` 和 `exampleMeaningZh` 可选。
- 孩子端不临时调用 AI 翻译或生成未经审核的课程内容。

`english_assets` 继续只保存生成素材状态和引用：

- 英语单词文本和所属主题。
- 拼写图引用。
- 义图引用。
- 英语发音音频引用。
- 是否需要义图及人工覆盖状态。

## 素材引用

MinIO 对象路径继续采用：

```text
english/glyphs/{kpId}.png
english/senses/{kpId}.png
english/speech/{kpId}.mp3
```

目标状态是数据库保存稳定的 object key，而不是带 `19091` host 的完整 URL。过渡阶段，`english-server` 根据 `kpId` 和素材类型推导现有对象路径，忽略数据库 URL 的 host，并通过自身接口输出素材：

```http
GET /api/v1/english/words/:kpId/glyph.png
GET /api/v1/english/words/:kpId/sense.png
GET /api/v1/english/words/:kpId/speech.mp3
```

`english-server` 只读取已经存在的 MinIO 对象，不负责调用 TTS、图片模型或生成拼写图。

## 题型与掌握度

第一版沿用两个技能代码：

| 技能代码 | 题型 | 掌握内容 |
| --- | --- | --- |
| `listen` | 听英语发音，从四个单词拼写中选择 | 发音到拼写的对应能力 |
| `picture` | 听英语发音，从四张义图中选择 | 发音到词义的对应能力 |

`listen` 题目选项保存稳定的知识点引用和显示文本：

```json
{
  "kpId": 101,
  "label": "cat"
}
```

`picture` 题目选项保存稳定的知识点引用和素材类型：

```json
{
  "kpId": 101,
  "assetKind": "sense"
}
```

`english-server` 将素材引用转换为自身 API URL。题目 JSON 不保存 `19091` URL，也不以 emoji 作为正式课程素材引用。

### 出题资格

听音选词要求：

- 当前单词已有英语音频。
- 同主题至少有三个可用干扰项。
- 四个单词拼写互不重复。

听音选图要求：

- 当前单词已有英语音频和义图。
- 三个同主题干扰项也都有义图。
- 四个选项的 `kpId` 和图片互不重复。

若图片素材不足，只选择 `listen` 题。`hello`、`please`、`sorry`、`yesterday` 等没有稳定视觉含义的抽象词默认不生成或选择图片题。

未来增加看图选词、看词选图或拼写练习时，必须使用新的技能代码，不能把不同能力继续并入 `picture`。

### 判题与重试

- 前端不接收正确答案。
- `english-server` 根据计划中固定的 `question_id` 读取数据库答案并判题。
- 创建计划时固定题目和选项顺序；刷新、恢复或换设备后保持不变。
- 第一次答错立即写入 `attempts` 并参与掌握度计算。
- 答错后允许再试一次；第二次答对不能覆盖第一次错误。
- 两次仍未答对时结束该题，并返回正确选项用于反馈。
- 每次提交携带唯一 `clientId`；重复提交返回第一次处理结果，不重复计分或发放奖励。

## 共享学习事务

现有家长后端已经实现计划、作答和掌握度能力。英语服务依赖按拼音设计提取后的 `shared-go`，复用以下稳定规则：

- 共享学习模型。
- 掌握度状态机和技能汇总。
- 幂等作答写入。
- 每日统计更新。
- 新掌握奖励事务。
- 与上述事务直接相关的数据访问。

一次英语作答必须在同一数据库事务中完成：

1. 验证孩子存在。
2. 验证计划属于该孩子且学科为英语。
3. 验证计划题目和选项下标。
4. 根据服务端保存的答案判题。
5. 使用 `clientId` 幂等写入 `attempts`。
6. 更新 `listen` 或 `picture` 技能掌握度。
7. 汇总更新单词级 `mastery_states`。
8. 更新 `daily_stats`、`plan_items` 和 `study_plans`。
9. 新掌握时写入小红花和 `flower_ledger`。

任一步骤失败时，整个事务回滚。提取 `shared-go` 时保持现有掌握度算法和家长后端行为，不在同一阶段重新设计算法。

## 后端 API

### 首页与进度

```http
GET /api/v1/children/:childId/english/home
GET /api/v1/children/:childId/english/progress
```

首页一次返回孩子信息、今日英语任务、补做任务、待复习数量、英语掌握摘要和小红花数量，避免首页发出多个聚合请求。

### 英语内容

```http
GET /api/v1/english/modules
GET /api/v1/english/modules/:moduleCode/words
GET /api/v1/children/:childId/english/words/:kpId
```

单词详情只返回认读展示所需字段、本服务素材 URL 和该孩子的掌握状态，不返回数据库连接信息或正确答案。

### 练习计划

```http
POST /api/v1/children/:childId/english/plans
GET  /api/v1/children/:childId/english/plans/:planId
POST /api/v1/children/:childId/english/plans/:planId/start
POST /api/v1/children/:childId/english/plans/:planId/items/:itemId/answer
POST /api/v1/children/:childId/english/plans/:planId/finish
```

创建计划支持：

- `today`：今日学习和复习混合计划。
- `review`：只练待复习和薄弱单词。
- `module`：从指定主题创建练习，要求 `moduleCode`。

创建计划时按待复习、需巩固、学习中和新词的顺序选择候选，并过滤素材不满足题型要求的题目。

答题请求：

```json
{
  "clientId": "uuid-from-client",
  "optionIndex": 2,
  "costMs": 4100
}
```

答题响应包含是否正确、是否允许重试、当前题目状态、当前单词掌握度和计划进度。答对或两次尝试耗尽前，不返回正确选项下标。

## english-app 页面设计

第一版页面路由：

```text
/                         英语首页
/words                    单词地图
/words/:kpId              单词认读
/task/:planId             答题
/task/:planId/done        结算
```

### 孩子身份

第一版沿用本地家庭使用方式，默认孩子 ID 为 `1`，并把孩子 ID 作为所有个人学习请求路径的一部分。`english-server` 必须验证孩子真实存在，不能只信任客户端传入的 ID。本阶段不新建账号系统。

### 首页

- 孩子姓名和“今天学英语”。
- 今日英语练习卡和补做任务。
- 待复习单词数量。
- 当前小红花。
- “开始练习”和“单词地图”两个主要入口。

### 单词地图

- 按现有英语主题分组展示。
- 单词卡显示英文、小型义图或拼写降级卡、掌握状态。
- 状态包括未开始、学习中、需巩固、待复习和已掌握。
- 点击单词进入认读页。
- 播放新单词前停止当前音频。

### 单词认读页

- 大号英文拼写。
- 必填的中文释义。
- 义图；缺失时显示拼写卡。
- 可选音标。
- 英语发音按钮。
- 当前掌握状态。
- 上一个、下一个单词和“练一练”入口。
- 例句数据存在时才展示，不用未经审核的自动内容填充。

### 答题页

- 一题一屏，不允许滚动看到其他题。
- 听音选词显示两列四个英文单词选项。
- 听音选图显示两列四张义图选项。
- 进入题目后自动播放一次音频。
- 首页开始按钮的孩子手势用于解锁 Safari 音频播放。
- 预加载当前题和下一题的音频及图片。
- 选项点击后立即锁定，防止连点。
- 答错时允许一次重试；答对或次数耗尽后进入下一题。
- 中途退出需要二次确认，重新进入后从第一道未完成题继续。

### 结算页

- 完成题数和正确题数。
- 星星和新获得的小红花。
- 最多六个需要再练的单词。
- “复习错词”和“回到首页”入口。
- 不向孩子展示复杂正确率、掌握度百分比或统计图。

## 前端组件边界

`english-app` 可以复用 `literacy-app` 已验证的 iPad 外壳、任务卡、进度点、反馈动画、退出确认、断点恢复、结算、音效和 PWA 模式，但不保留 Hanzi Writer、汉字视觉类型或识字专属逻辑。

建议组件结构：

```text
src/
├── api/
│   ├── client.ts
│   ├── catalog.ts
│   ├── plans.ts
│   └── progress.ts
├── audio/
│   ├── AudioController.ts
│   └── preload.ts
├── components/
│   ├── AppShell.tsx
│   ├── WordCard.tsx
│   ├── SenseImage.tsx
│   ├── AudioButton.tsx
│   ├── OptionButton.tsx
│   ├── ProgressDots.tsx
│   └── ResultSummary.tsx
├── features/
│   ├── home/
│   ├── words/
│   ├── practice/
│   ├── progress/
│   └── result/
└── pages/
```

音频由单例 `AudioController` 管理，保证同时只播放一个音频，切题、退出或组件卸载时停止播放，并区分对象不存在、网络失败和浏览器禁止播放。

可以在多个孩子端形成第二个稳定消费者后建立 `shared-web`，但只放通用学习流程、音频控制和 iPad UI 原语，不放英语单词卡、拼音认读或识字书写等学科组件。

## iPad 交互要求

- 以横屏 `1024 × 768` 为主要验收尺寸，并兼容更高分辨率横屏。
- 主要触控区域不小于约 `56px`。
- 不依赖 hover 才能发现操作。
- 处理安全区域和添加到主屏幕后全屏显示。
- 音频首次播放由明确的孩子手势触发，以满足 Safari 自动播放限制。
- 同一时刻只允许播放一个音频。
- 网络较慢时保留题面和明确的重试按钮，不能让按钮无反馈。

## 错误处理

| 场景 | 服务端行为 | 前端行为 |
| --- | --- | --- |
| 单词或题目不存在 | `404 content_not_found` | 返回主题页并刷新内容 |
| 图片尚未生成 | `404 asset_missing` | 显示拼写卡，不显示破图 |
| 音频尚未生成 | `404 asset_missing` | 当前题不计错，提示稍后再试 |
| MinIO 暂时不可用 | `503 asset_unavailable` | 保留当前题并允许重播 |
| 数据库不可用 | `503 database_unavailable` | 保留题目和选择，允许重新提交 |
| 计划不属于孩子 | `403 plan_forbidden` | 返回首页，不泄露计划内容 |
| 计划已经完成 | `409 plan_completed` 或当前结算 | 进入结算页 |
| 重复提交 | 返回首次幂等结果 | 不重复播放奖励动画 |
| 选项下标无效 | `400 invalid_option` | 恢复按钮并重新加载该题 |
| 没有可练题目 | `409 no_eligible_questions` | 提示请家长准备内容 |

计划创建阶段过滤素材不完整的题目。网络提交失败时，前端保留当前题目和孩子选择，使用相同 `clientId` 重试，不擅自进入下一题。

## 测试策略

### shared-go

- 答对、答错、两次尝试和复习到期。
- 技能级掌握度正确汇总到单词级。
- 相同 `clientId` 不重复写入、计分或发放奖励。
- 事务任一步失败时全部回滚。
- 保持家长后端现有掌握度行为和测试通过。

### english-server

- 所有内容查询限定 `subject = english`。
- 无法通过英语接口读取或修改其他学科计划。
- 正确答案不会出现在取题响应中。
- 计划生成后题目和选项顺序稳定。
- `listen` 干扰项来自同一主题。
- `picture` 四个选项都有有效义图。
- 抽象词不选择图片题。
- 素材缺失的题目不进入新计划。
- 相同 `clientId` 重复提交返回一致结果。
- 使用 PostgreSQL 集成测试覆盖生产 SQL 行为。
- 使用 MinIO 接口替身测试素材读取，不调用真实 AI 或 TTS。

### english-app

- 首页创建和恢复英语计划。
- 单词地图按主题和掌握状态展示。
- 单词认读页播放、停止和切换音频。
- 听音选词和听音选图完整交互。
- 连续点击锁定和答错重试一次。
- 提交失败后以相同 `clientId` 重试。
- 刷新页面后恢复到第一道未完成题。
- 图片缺失时正常显示降级界面。
- 使用 Playwright 的 iPad 横屏视口验证首页、认读、答题和结算主流程。

### 跨系统验收

1. 内容后台生成一个英语单词的拼写图、义图和音频。
2. 停止内容后台。
3. `english-server` 在不请求内容后台 HTTP API 的情况下读取并输出素材。
4. 孩子在 `english-app` 完成一份英语练习。
5. 数据库出现对应的英语计划、作答、技能掌握度、单词掌握度和每日统计。
6. 家长看板在不调用 `english-server` 的情况下展示英语结果。

## 部署

- 根 `kid-workbench/docker-compose.yml` 增加 `english-server` 和 `english-app`。
- 两个新服务加入与 PostgreSQL、MinIO 相同的 Docker 网络。
- `english-app` 生产环境只配置一个 API 地址，即 `english-server`。
- `english-app` 提供 PWA manifest 和适合 iPad 主屏幕的图标。
- `english-server` 使用现有数据库环境变量规范，并配置 MinIO 只读访问能力。
- `/healthz` 只表示进程存活；`/readyz` 检查 PostgreSQL 和 MinIO 依赖。
- 数据库迁移必须向前兼容；部署英语服务不要求停止其他学科、内容后台或家长看板。

## 实施顺序

1. 完成拼音规范中定义的 `shared-go` 提取，并保持家长后端行为不变。
2. 为英语知识点补充人工审核的中文释义。
3. 调整英语题目选项，使其保存稳定的 `kpId` 和素材类型。
4. 建立 `english-server`，完成健康检查、内容查询和素材输出。
5. 完成英语计划生成、服务端判题、幂等作答和掌握度写入。
6. 建立 `english-app`，复用识字孩子端已经验证的稳定交互模式。
7. 完成首页、单词地图和认读页。
8. 完成听音选词、听音选图、断点恢复和结算页。
9. 接入根 Compose，并进行 PostgreSQL、MinIO 集成验证。
10. 验证家长看板能够读取新产生的英语学习结果。

## 完成标准

- `english-app` 与 `english-server` 可以独立构建和部署。
- `english-app` 只请求 `english-server`。
- `english-server` 不请求 `19091` 或 `19081` 的 HTTP API。
- 内容后台停机时，已有英语内容仍可正常学习。
- 孩子可以浏览主题、认读单词并完成一份英语练习。
- 第一版支持听音选词和听音选图。
- 正确答案不会在作答前发送到前端。
- 刷新或退出后可以继续原练习。
- 一次作答原子更新计划、作答、掌握度、统计和奖励。
- 重复提交不会重复计分或发放奖励。
- 家长看板可以直接从共享数据库看到英语学习结果。
- iPad 横屏主要操作区域不小于约 `56px`。
- 第一版不包含键盘拼写、录音评分、短句、语法或多人对战。
