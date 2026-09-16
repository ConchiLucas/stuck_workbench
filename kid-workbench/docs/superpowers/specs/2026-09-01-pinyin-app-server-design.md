# 拼音孩子端与拼音服务设计

## 目标

在 `kid-workbench` 中新增面向孩子的独立拼音学习前端 `pinyin-app`，以及只服务该前端的 Go 后端 `pinyin-server`。

第一版面向 iPad 横屏场景，覆盖拼音认读、听音练习、错题重试、练习结算和掌握度更新。内容后台、拼音服务和家长看板之间不进行 HTTP 调用，只共享 PostgreSQL 中约定的数据表，以及 MinIO 中已经生成的拼音素材。

## 已确认的产品边界

- `content-admin`（`http://localhost:19091`）负责维护拼音内容、题目、字图和音频。
- `pinyin-app` 只调用 `pinyin-server`，不直接访问内容后台、家长看板、PostgreSQL 或 MinIO。
- `pinyin-server` 直接读取 PostgreSQL 中的拼音素材和题目，并写入答题、计划、统计及掌握度数据。
- `parent-dashboard`（`http://localhost:19081`）直接从共享数据库读取拼音答题结果和掌握度，用于家长查看。
- 三个后端之间没有运行时依赖；任一后台服务停止，不应阻止另外两个服务直接访问数据库中已经存在的数据。

## 非目标

第一版不包含以下能力：

- 孩子录音、发音打分或语音识别。
- 拼音内容编辑、题目编辑、TTS 生成或字图生成。
- 自定义课程编排后台。
- 多人在线对战。
- 将五个学科一次性抽象为通用低代码学习平台。
- 修改内容后台中识字、算术、英语或科普模块的现有行为。

## 系统架构

```text
content-admin :19091
  ├─ 写 subjects / modules / knowledge_points
  ├─ 写 questions / pinyin_assets
  └─ 写 MinIO 拼音字图和音频

                         PostgreSQL + MinIO
                         ▲              ▲
                         │              │
pinyin-app :19112 ──HTTP──> pinyin-server :19111
                         │
                         ├─ 读拼音内容、题目和孩子进度
                         └─ 写计划、作答、掌握度、统计和奖励流水

parent-dashboard :19081
  └─ 读共享的计划、作答、掌握度和统计数据
```

建议开发端口为：

| 服务 | 端口 | 说明 |
| --- | ---: | --- |
| `pinyin-server` | `19111` | 拼音孩子端 API |
| `pinyin-app` | `19112` | Vite 开发端口及 Docker 暴露端口 |

端口避开现有的 `19081`、`19082`、`19083`、`19091`、`19092` 和 MinIO `19100`。

## 项目结构

```text
kid-workbench/
├── literacy-app/
├── pinyin-app/
│   ├── src/
│   │   ├── api/
│   │   ├── audio/
│   │   ├── components/
│   │   ├── features/
│   │   │   ├── home/
│   │   │   ├── learn/
│   │   │   ├── practice/
│   │   │   ├── progress/
│   │   │   └── result/
│   │   ├── pages/
│   │   └── styles/
│   ├── Dockerfile
│   └── package.json
├── pinyin-server/
│   ├── cmd/server/
│   ├── internal/
│   │   ├── asset/
│   │   ├── catalog/
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

`shared-go` 只承载已经在识字和拼音中共同成立的学习数据规则，不承载页面模型、拼音素材规则或学科题目生成逻辑。

## 数据所有权

共享数据库不代表任意服务都可以修改任意表。各类数据的主要写入方如下：

| 数据 | 主要写入方 | 主要读取方 |
| --- | --- | --- |
| `subjects`、`modules`、`knowledge_points` | 内容初始化或内容后台 | 全部系统 |
| `questions` | 内容后台 | `pinyin-server` |
| `pinyin_assets` | 内容后台 | `pinyin-server` |
| MinIO 拼音素材 | 内容后台 | `pinyin-server` |
| `study_plans`、`plan_items` | 学科孩子端后端 | 学科后端、家长看板 |
| `attempts` | 学科孩子端后端 | 学科后端、家长看板 |
| `mastery_skills`、`mastery_states` | 共享学习事务 | 学科后端、家长看板 |
| `daily_stats`、`flower_ledger` | 共享学习事务 | 家长看板 |

`pinyin-server` 所有内容查询必须限制 `subjects.code = 'pinyin'`，防止错误地读取或修改其他学科数据。

## 拼音内容模型

第一版沿用当前的 45 个拼音知识点，包括声母和韵母。每个知识点通过 `pinyin_assets.kp_id` 关联 `knowledge_points.id`。

`pinyin_assets` 提供：

- 拼音文本 `letter`。
- 所属模块及排序。
- 单读承载文本 `solo_text`。
- 例字 `word_text`。
- 单读音频、例字音频及拼音字图的素材引用。

拼音前端不保存 `Readings`、`PINYIN_READING` 或 `CONFUSION` 的副本。例字和读音文本以 `pinyin_assets` 为准，题干、选项、正确答案及易混音选择以数据库中的 `questions` 为准。

`eng` 没有可靠的单读承载文本，第一版只提供“听例字选音”，不生成或展示“听单读选字母”。

## 素材引用

当前 `pinyin_assets` 中可能保存带 `19091` 域名的完整 URL。新系统不能把该 URL 原样返回给孩子端，否则会形成对内容后台的运行时依赖。

目标状态是数据库保存稳定的 MinIO object key。过渡阶段，`pinyin-server` 根据 `kpId` 和素材类型推导现有对象路径，忽略已存 URL 的 host，并通过自身接口输出素材：

```http
GET /api/v1/pinyin/items/:kpId/glyph.png
GET /api/v1/pinyin/items/:kpId/speech/solo.mp3
GET /api/v1/pinyin/items/:kpId/speech/word.mp3
```

`pinyin-server` 只读取已经存在的 MinIO 对象，不负责调用 TTS 或生成字图。素材缺失时返回明确的 `404 asset_missing`，前端显示拼音文本并允许跳过音频依赖的题目。

## 练习与掌握度

拼音沿用两个技能代码：

| 技能 | 题目代码 | 含义 |
| --- | --- | --- |
| 听例字选音 | `inword` | 播放并展示例字，从四个拼音中选择对应音 |
| 听单读选字母 | `listen` | 播放单读音频，从四个拼音中选择正确项 |

前端不接收正确答案。`pinyin-server` 根据 `question_id` 读取数据库答案并在服务端判题。

一次作答必须在同一数据库事务中完成：

1. 校验孩子、练习计划、计划题目和选项下标。
2. 读取固定的 `question_id` 并判题。
3. 用唯一 `client_id` 写入 `attempts`。
4. 更新对应技能的 `mastery_skills`。
5. 汇总更新知识点级 `mastery_states`。
6. 更新 `daily_stats`。
7. 新掌握时写入小红花和 `flower_ledger`。
8. 更新 `plan_items` 和 `study_plans`。

若相同 `client_id` 重复提交，返回第一次处理后的结果，不重复计分或发放奖励。

答错后允许再试一次。第一次错误立即进入 `attempts` 和掌握度计算；第二次答对不能覆盖第一次错误。用完两次仍未答对时结束该题。

## 共享 Go 代码

现有 `parent-dashboard/backend` 已实现上述答题事务和掌握度引擎。为了保证未来各学科服务写入一致，应提取以下稳定能力到 `shared-go`：

- 共享学习模型。
- 掌握度状态机与技能汇总。
- 幂等作答写入。
- 每日统计更新。
- 新掌握奖励事务。
- 与这些事务直接相关的数据访问。

`parent-dashboard/backend` 和 `pinyin-server` 都依赖该本地 Go module。提取时保持现有行为和测试通过，不在同一阶段重新设计掌握度算法。

## 后端 API

### 首页与进度

```http
GET /api/v1/children/:childId/pinyin/home
GET /api/v1/children/:childId/pinyin/progress
```

首页响应包含今日待完成计划、待复习数量、声母和韵母的掌握概况。进度响应按模块返回知识点及其两个技能的状态。

### 认读内容

```http
GET /api/v1/pinyin/modules
GET /api/v1/pinyin/modules/:moduleCode/items
GET /api/v1/pinyin/items/:kpId
```

只返回学习展示所需字段和本服务素材 URL，不返回数据库内部连接信息或正确答案。

### 练习计划

```http
POST /api/v1/children/:childId/pinyin/plans
GET  /api/v1/children/:childId/pinyin/plans/:planId
POST /api/v1/children/:childId/pinyin/plans/:planId/start
POST /api/v1/children/:childId/pinyin/plans/:planId/items/:itemId/answer
POST /api/v1/children/:childId/pinyin/plans/:planId/finish
```

创建计划时固定 `question_id` 和选项顺序。刷新页面、恢复会话或换设备后必须返回同一题目和相同选项顺序。

答题请求：

```json
{
  "clientId": "uuid-from-client",
  "optionIndex": 2,
  "costMs": 4100
}
```

答题响应包含是否正确、是否允许重试、当前题目状态、当前知识点掌握度和计划进度。只有提交后才能返回本题正确选项下标。

## pinyin-app 页面设计

### 孩子身份

第一版沿用现有 `literacy-app` 的本地家庭使用方式，默认孩子 ID 为 `1`，并把孩子 ID 作为所有学习请求路径的一部分。`pinyin-server` 必须验证该孩子真实存在，不能仅信任客户端传入的 ID。

本阶段不新建登录和账号系统。未来接入家庭账号后，由统一认证层确定可访问的孩子 ID，不改变拼音内容、计划和掌握度模型。

### 首页

- 今日拼音任务卡。
- 待复习数量。
- 声母和韵母进度摘要。
- “开始练习”和“拼音地图”两个主要入口。

### 拼音地图

- 声母、韵母分组展示。
- 每个拼音显示未开始、学习中、需巩固、待复习或已掌握状态。
- 点击拼音进入认读页。

### 认读页

- 使用适合拼音教学的四线三格展示，不复用内容后台的田字格样式。
- 显示单读、例字和两个独立播放按钮。
- `solo_text` 为空时隐藏单读区域。
- 第一版不提供录音、跟读评分或手写轨迹。

### 答题页

- iPad 横屏固定三栏：左侧题干，右侧两列四个选项。
- 页面只显示当前题，不允许孩子通过滚动看到其他题。
- 自动预加载当前题和下一题音频。
- 选项点击后立即锁定，等待服务端结果，防止连点。
- 答错时允许一次重试；答对或次数耗尽后进入下一题。

### 结算页

- 正确题数、完成题数、星星和小红花。
- 展示需要再练的拼音，不展示复杂百分比。
- 提供“回到首页”和“复习错题”入口。

## iPad 交互要求

- 以横屏 `1024 × 768` 为主要验收尺寸，并兼容更高分辨率横屏。
- 主要触控区域不小于约 `56px`。
- 不依赖 hover 才能发现的操作。
- 处理安全区域和添加到主屏幕后全屏显示。
- 音频首次播放由明确的孩子手势触发，以满足 Safari 自动播放限制。
- 同一时刻只允许播放一个音频；切题或离开页面时停止上一个音频。
- 网络较慢时保留题面骨架和明确的重试按钮，不能让按钮无反馈。

## 错误处理

| 场景 | 服务端行为 | 前端行为 |
| --- | --- | --- |
| 素材未生成 | `404 asset_missing` | 显示文字；音频题跳过并记录内容问题 |
| 题目不存在或未发布 | 不加入新计划 | 提示暂无可练题目 |
| 重复提交 | 返回幂等结果 | 不重复播放奖励动画 |
| 计划已完成 | 返回当前结算 | 跳转结算页 |
| 数据库暂时不可用 | `503 database_unavailable` | 保留当前题和选择，允许重试提交 |
| MinIO 暂时不可用 | `503 asset_unavailable` | 提供重试播放，不重新生成素材 |
| 内容在练习中被更新 | 继续使用计划固定的 `question_id` | 当前练习不换题 |

## 测试策略

### 共享学习模块

- 保留并迁移现有掌握度状态机测试。
- 验证答对、答错、两次尝试和复习到期。
- 验证相同 `client_id` 不重复写入、计分和奖励。
- 验证技能级掌握度正确汇总到知识点级。

### pinyin-server

- 验证所有查询限定 `subject = pinyin`。
- 验证正确答案不会出现在取题响应中。
- 验证计划生成后题目和选项顺序稳定。
- 验证作答事务要么全部成功，要么全部回滚。
- 验证 `eng` 不生成 `listen` 题。
- 使用 PostgreSQL 集成测试覆盖生产方言相关行为。
- 使用 MinIO 接口替身测试字图和音频读取，不在测试中调用真实 TTS。

### pinyin-app

- 使用 Vitest 和 Testing Library 测试页面状态与答题交互。
- 测试音频播放失败、重复点击、重试提交和页面恢复。
- 使用 Playwright 的 iPad 横屏视口验证首页、认读、答题和结算主流程。
- 验证所有主要操作均可通过触摸完成。

### 跨系统验收

1. 内容后台生成一个拼音字图和两类音频。
2. `pinyin-server` 在不调用内容后台 HTTP API 的情况下读取并输出素材。
3. 孩子完成一份拼音练习。
4. 数据库出现对应的计划、作答、技能掌握度、知识点掌握度和每日统计。
5. 家长看板在不调用 `pinyin-server` 的情况下展示该结果。

## 部署

- 根 `kid-workbench/docker-compose.yml` 增加 `pinyin-server` 和 `pinyin-app`。
- 两个新服务加入与 PostgreSQL、MinIO 相同的 Docker 网络。
- `pinyin-app` 生产环境只配置一个 API 地址，即 `pinyin-server`。
- `pinyin-app` 提供 PWA manifest 和适合 iPad 主屏幕的图标，可添加到主屏幕后横屏全屏运行。
- `pinyin-server` 使用现有数据库环境变量规范，并配置 MinIO 只读访问能力。
- 数据库迁移必须向前兼容；部署新服务不能要求先停止家长看板或内容后台。
- 健康检查区分进程存活与数据库、MinIO 依赖状态。

## 实施顺序

1. 为共享学习事务补齐现有行为测试。
2. 提取 `shared-go`，保持家长后端测试和行为不变。
3. 建立 `pinyin-server`，完成健康检查、内容读取和素材输出。
4. 完成拼音计划生成、服务端判题、幂等作答和掌握度写入。
5. 建立 `pinyin-app`，完成首页、拼音地图和认读页。
6. 完成答题、断点恢复和结算页。
7. 接入根 Compose，并进行 PostgreSQL、MinIO 集成验证。
8. 验证家长看板能读取新产生的拼音学习结果。

## 完成标准

- `pinyin-app` 在 iPad 横屏上可以独立完成认读和一份拼音练习。
- `pinyin-app` 只请求 `pinyin-server`。
- `pinyin-server` 不请求 `19091` 或 `19081` 的 HTTP API。
- 内容后台停机时，已生成的拼音素材和题目仍可被新系统使用。
- 一次作答同时、正确地更新计划、尝试记录、技能掌握度、综合掌握度和每日统计。
- 重复提交不会重复计分或发放奖励。
- 家长看板可以直接从共享数据库看到新产生的拼音结果。
- 内容、学习和展示三类职责具有明确的数据写入边界。
