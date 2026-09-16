# 科普孩子端与科普服务设计

## 目标

在 `kid-workbench` 中新增面向孩子的独立科普学习前端 `science-app`，以及只服务该前端的 Go 后端 `science-server`。

第一版面向 iPad 横屏场景，覆盖主题探索、知识认读、科普问答、错题重试、练习结算和掌握度更新。内容后台、科普服务和家长看板之间不进行 HTTP 调用，只共享 PostgreSQL 中约定的数据表，以及 MinIO 中已经生成的科普素材。

第一版以“概念辨认”为唯一掌握技能。当前 `fact1`、`fact2` 只是同一问题的选项重排，不作为两个独立技能。等内容补齐真正不同的原理或情境问题后，再增加“原理理解”技能。

## 已确认的产品边界

- `content-admin`（`http://localhost:19091`）负责维护科普内容、题目、义图、字图和语音。
- `science-app` 只调用 `science-server`，不直接访问内容后台、家长看板、PostgreSQL 或 MinIO。
- `science-server` 直接读取 PostgreSQL 中已发布的科普内容、素材索引和题目，并写入答题、计划、统计及掌握度数据。
- `parent-dashboard`（`http://localhost:19081`）直接从共享数据库读取科普答题结果和掌握度，用于家长查看。
- 三个后端之间没有运行时 HTTP 依赖；内容后台或家长看板停止，不应阻止科普服务读取数据库与 MinIO 中已经发布的内容。
- 科普孩子端以义图、简短解释和问题反馈为核心；Emoji 只作为素材缺失时的降级展示。

## 非目标

第一版不包含以下能力：

- AI 实时问答、开放式聊天或自动追问。
- 视频生成、互动实验或复杂小游戏。
- 孩子录音、语音识别或口语评分。
- 科普内容、题目、图片或 TTS 的编辑与生成。
- 自定义课程编排后台。
- 多人在线对战。
- 同时实现识字、拼音、算术和英语独立前端。
- 将所有学科抽象为通用低代码学习平台。
- 重新设计现有掌握度算法或奖励规则。

## 系统架构

```text
content-admin :19091
  ├─ 写 subjects / modules / knowledge_points
  ├─ 写 questions / science_assets
  ├─ 管理内容审核与发布状态
  └─ 写 MinIO 科普义图、字图和语音

                         PostgreSQL + MinIO
                         ▲              ▲
                         │              │
science-app :19122 ─HTTP──> science-server :19121
                         │
                         ├─ 读已发布科普内容、题目、素材和孩子进度
                         └─ 写计划、作答、掌握度、统计和奖励流水

parent-dashboard :19081
  └─ 读共享的计划、作答、掌握度和统计数据
```

建议开发端口为：

| 服务 | 端口 | 说明 |
| --- | ---: | --- |
| `science-server` | `19121` | 科普孩子端 API |
| `science-app` | `19122` | Vite 开发端口及 Docker 暴露端口 |

端口避开现有的 `19081`、`19082`、`19083`、`19091`、`19092`、MinIO `19100`，以及拼音设计预留的 `19111`、`19112`。

## 项目结构

```text
kid-workbench/
├── literacy-app/
├── science-app/
│   ├── src/
│   │   ├── api/
│   │   ├── audio/
│   │   ├── components/
│   │   ├── features/
│   │   │   ├── home/
│   │   │   ├── explore/
│   │   │   ├── concept/
│   │   │   ├── practice/
│   │   │   ├── progress/
│   │   │   └── result/
│   │   ├── pages/
│   │   └── styles/
│   ├── Dockerfile
│   └── package.json
├── science-server/
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

`shared-go` 只承载已经在识字、拼音和科普中共同成立的学习数据规则，不承载页面模型、科普内容结构、科普素材规则或科普题目生成逻辑。

当前仓库尚未建立 `shared-go`。实现科普服务前，先从 `parent-dashboard/backend` 提取稳定学习事务，并保持现有行为和测试不变。

## 数据所有权

共享数据库不代表任意服务都可以修改任意表。各类数据的主要写入方如下：

| 数据 | 主要写入方 | 主要读取方 |
| --- | --- | --- |
| `subjects`、`modules`、`knowledge_points` | 内容初始化或内容后台 | 全部系统 |
| `questions` | 内容后台 | `science-server` |
| `science_assets` | 内容后台 | `science-server` |
| 科普审核与发布字段 | 内容后台 | `science-server` |
| MinIO 科普素材 | 内容后台 | `science-server` |
| `study_plans`、`plan_items` | `science-server` | 科普服务、家长看板 |
| `attempts` | `science-server` 通过共享学习事务写入 | 科普服务、家长看板 |
| `mastery_skills`、`mastery_states` | 共享学习事务 | 科普服务、家长看板 |
| `daily_stats`、`flower_ledger` | 共享学习事务 | 家长看板 |

`science-server` 所有内容、题目、计划和进度查询必须限制 `subjects.code = 'science'`，防止错误地读取或修改其他学科数据。

## 科普内容模型

当前科普目录约有 99 个知识点、10 个模块：动物、植物、天气、宇宙、身体、交通、环保、安全、材料和生活常识。每个知识点通过 `science_assets.kp_id` 关联 `knowledge_points.id`。

当前知识点 `payload` 主要包含：

```json
{
  "kind": "fact",
  "q": "冬天躲起来睡觉过冬的是？",
  "a": "熊",
  "wrong": ["燕子", "蝴蝶", "鸭子"],
  "emoji": "🐻"
}
```

为了支持独立科普学习体验，目标内容模型增加一句话解释、原因说明、趣味知识和可区分的题目集合：

```json
{
  "kind": "fact",
  "summary": "冬眠是一些动物度过寒冷冬天的方法。",
  "explanation": "冬天食物少，熊通过减少活动来节省能量。",
  "funFact": "熊冬眠时心跳会比平时慢很多。",
  "questions": [
    {
      "code": "recognize",
      "q": "冬天躲起来睡觉过冬的是？",
      "a": "熊",
      "wrong": ["燕子", "蝴蝶", "鸭子"]
    }
  ]
}
```

内容后台或种子流程把题干、选项、答案和难度发布到 `questions`。`science-server` 从 `questions` 组装计划并在服务端判题，不根据前端提交内容判断答案。

为避免阻塞第一版，旧 payload 继续兼容：缺少 `summary`、`explanation` 或 `funFact` 时仍允许概念辨认练习，知识卡隐藏缺失区域，答题反馈退化为只展示正确答案。内容后台同步科普知识点时，将这些展示字段写入 `science_assets`，孩子端服务不需要在运行时解析或依赖内容后台内部的生成流程。

## 内容审核与发布

科普内容包含急救、防火、防电、防溺水、身体健康等安全相关主题，需要明确的人工审核和发布状态。AI 可以生成候选插图或文案，但不能自动发布安全、急救和健康内容。

第一版按知识点整体审核和发布，不对同一知识点下的单道题分别发布。在 `science_assets` 增加以下字段：

- `summary`：一句话解释。
- `explanation`：答题后的原因说明。
- `fun_fact`：可选趣味知识。
- `review_status`：`draft`、`reviewed` 或 `published`。
- `reviewed_at`：最近人工审核时间。
- `content_version`：从 `1` 开始递增的整数，用于审计和缓存更新。

`science-server` 查询题目时必须连接 `science_assets`，只把 `review_status = 'published'` 的知识点及其题目加入新计划。过渡阶段现有数据尚无发布字段时，通过一次向前兼容迁移将当前科普素材记录标记为已发布；迁移完成后，新同步内容默认 `draft`。

素材接口在 URL 中携带 `content_version`，例如 `/sense.png?v=3`。内容或素材更新并重新发布时递增版本，避免 iPad 继续使用旧缓存。

练习创建后继续使用当时固定的 `question_id` 和选项顺序。内容后来下线或更新，不改变正在进行的计划；新计划只使用当前已发布版本。

## 科普素材模型

`science_assets` 当前提供：

- 标题 `title`。
- 所属模块和排序。
- 标题字图 `glyph_image_url`。
- 概念义图 `sense_image_url`。
- 标题读音 `speech_audio_url`。
- 是否需要义图及人工覆盖状态。

科普孩子端素材优先级为：

1. 概念义图，用于知识卡和图片题。
2. 标题或问题语音，用于认读和答题。
3. 标题字图，用于需要强调概念名称的场景。
4. Emoji，只作为义图缺失时的降级。

## 素材引用

当前 `science_assets` 中可能保存带 `19091` 域名的完整 URL。新系统不能把该 URL 原样返回给孩子端，否则会形成对内容后台的运行时依赖，并且 iPad 上的 `localhost` 会指向设备自身。

目标状态是数据库保存稳定的 MinIO object key。过渡阶段，`science-server` 根据 `kpId` 和素材类型推导现有对象路径，忽略已存 URL 的 host，并通过自身接口输出素材：

```http
GET /api/v1/science/items/:kpId/sense.png
GET /api/v1/science/items/:kpId/glyph.png
GET /api/v1/science/items/:kpId/speech.mp3
```

`science-server` 只读取已经存在的 MinIO 对象，不负责调用图片模型、TTS 或生成字图。素材缺失时返回明确的 `404 asset_missing`。

缺少义图的普通文字题仍可作答；依赖义图才能成立的题目不加入新计划。缺少语音时前端显示文字并允许继续，不把普通科普题整体阻断。

## 练习与掌握度

第一版只启用一个科普技能：

| 技能 | 题目代码 | 含义 |
| --- | --- | --- |
| 概念辨认 | `recognize` | 根据图片、描述、现象或生活场景辨认概念或正确做法 |

当前数据库中的 `fact1`、`fact2` 在迁移或重新灌题时映射为 `recognize`，但同一份计划中每个知识点只选一道，避免重复问题伪装成不同练习。

第二阶段内容补齐真正不同的问题后，可增加：

| 技能 | 题目代码 | 含义 |
| --- | --- | --- |
| 原理理解 | `understand` | 理解简单原因、结果、规则或情境中的正确应用 |

启用 `understand` 前，必须保证参与计划的知识点拥有独立的理解题；不能仅通过打乱选项创建第二技能。

前端不接收正确答案。`science-server` 根据 `question_id` 读取数据库答案并在服务端判题。

一次作答必须在同一数据库事务中完成：

1. 校验孩子、科普计划、计划题目和选项下标。
2. 校验计划题目所属学科为 `science`。
3. 读取固定的 `question_id` 并判题。
4. 用唯一 `client_id` 写入 `attempts`。
5. 更新对应技能的 `mastery_skills`。
6. 汇总更新知识点级 `mastery_states`。
7. 更新 `daily_stats`。
8. 新掌握时写入小红花和 `flower_ledger`。
9. 更新 `plan_items` 和 `study_plans`。

若相同 `client_id` 重复提交，返回第一次处理后的结果，不重复计分或发放奖励。

答错后允许再试一次。第一次错误立即进入 `attempts` 和掌握度计算；第二次答对不能覆盖第一次错误。用完两次仍未答对时结束该题，并向前端返回正确选项和简短解释。

## 共享 Go 代码

现有 `parent-dashboard/backend` 已实现计划、答题事务和掌握度引擎。为了保证未来各学科服务写入一致，应提取以下稳定能力到 `shared-go`：

- 共享学习模型。
- 掌握度状态机与技能汇总。
- 幂等作答写入。
- 每日统计更新。
- 新掌握奖励事务。
- 与这些事务直接相关的数据访问。

`parent-dashboard/backend`、后续的 `pinyin-server` 和 `science-server` 都依赖该本地 Go module。提取时保持现有行为和测试通过，不在同一阶段重新设计掌握度算法。

科普内容解析、发布过滤、素材寻址、模块浏览和科普题目选择保留在 `science-server`，不进入 `shared-go`。

## 后端 API

### 首页与进度

```http
GET /api/v1/children/:childId/science/home
GET /api/v1/children/:childId/science/progress
```

首页响应包含今日待完成科普计划、待复习数量、已探索知识点数，以及各主题的掌握概况。进度响应按模块返回知识点和概念辨认技能状态。

### 探索与知识卡

```http
GET /api/v1/science/modules
GET /api/v1/science/modules/:moduleCode/items
GET /api/v1/science/items/:kpId
```

知识点详情只返回孩子展示需要的字段：标题、模块、难度、解释、趣味知识、当前掌握状态和本服务素材 URL。接口不返回数据库连接信息、MinIO 凭据、未发布内容或题目正确答案。

### 练习计划

```http
POST /api/v1/children/:childId/science/plans
GET  /api/v1/children/:childId/science/plans/:planId
POST /api/v1/children/:childId/science/plans/:planId/start
POST /api/v1/children/:childId/science/plans/:planId/items/:itemId/answer
POST /api/v1/children/:childId/science/plans/:planId/finish
```

创建计划支持两种模式：

```json
{
  "mode": "daily"
}
```

```json
{
  "mode": "module",
  "moduleCode": "space"
}
```

`daily` 根据待复习、需巩固、学习中和新知识候选生成今日计划。`module` 仍遵守掌握度优先级，但只在指定科普模块内选题。前端不能直接提交任意 `kpId` 列表。

创建计划时固定 `question_id` 和选项顺序。刷新页面、恢复会话或换设备后必须返回同一题目和相同选项顺序。

答题请求：

```json
{
  "clientId": "uuid-from-client",
  "optionIndex": 2,
  "costMs": 4100
}
```

答题响应包含是否正确、是否允许重试、当前题目状态、当前知识点掌握度和计划进度。只有提交后才能返回本题正确选项下标；答对或次数耗尽时可同时返回简短解释。

## science-app 页面设计

### 孩子身份

第一版沿用现有 `literacy-app` 的本地家庭使用方式，默认孩子 ID 为 `1`，并把孩子 ID 作为所有学习请求路径的一部分。`science-server` 必须验证该孩子真实存在，不能仅信任客户端传入的 ID。

本阶段不新建登录和账号系统。未来接入家庭账号后，由统一认证层确定可访问的孩子 ID，不改变科普内容、计划和掌握度模型。

### 首页

- 今日科普任务卡。
- 待复习数量和已探索知识点数。
- “开始今日探索”和“科普地图”两个主要入口。
- 动物、植物、天气、宇宙等主题的简要进度。

### 科普地图

- 按模块展示动物世界、植物花园、天气观察站、宇宙空间站、身体实验室、交通、环保、安全、材料和生活常识。
- 每个模块显示已探索、待复习和已掌握数量。
- 每个知识点显示未开始、学习中、需巩固、待复习或已掌握状态。
- 点击知识点进入知识卡，不在地图页直接泄露题目答案。

### 知识卡

- 以概念义图作为主要视觉内容。
- 显示标题、简短解释和可选的“你知道吗”。
- 提供一个明确的播放按钮；不依赖自动播放才能理解内容。
- 素材缺失时使用标题或 Emoji 降级，不显示破损图片。
- 提供“试一题”和返回当前主题两个操作。

### 答题页

- iPad 横屏一题一屏，不允许滚动看到后续问题。
- 默认使用两列四个选项，较长答案允许自动调整字号和行高。
- 自动预加载当前题和下一题所需图片或音频。
- 选项点击后立即锁定，等待服务端结果，防止连点。
- 答错时允许一次重试；答对或次数耗尽后显示简短解释再进入下一题。
- 安全类问题答错后必须展示明确的正确做法，不使用含糊的“下次记住”作为唯一反馈。

### 结算页

- 展示本次发现的新知识数量、正确题数、星星和小红花。
- 展示需要再看的知识点，不展示复杂百分比。
- 提供“回到首页”“复习错题”和“继续这个主题”入口。

## iPad 交互要求

- 以横屏 `1024 × 768` 为主要验收尺寸，并兼容更高分辨率横屏。
- 主要触控区域不小于约 `56px`。
- 不依赖 hover 才能发现的操作。
- 处理安全区域和添加到主屏幕后全屏显示。
- 音频首次播放由明确的孩子手势触发，以满足 Safari 自动播放限制。
- 同一时刻只允许播放一个音频；切题或离开页面时停止上一个音频。
- 网络较慢时保留题面骨架和明确的重试按钮，不能让按钮无反馈。
- 图片需要固定占位尺寸，加载完成时不应造成选项位置跳动。
- 支持 `prefers-reduced-motion`，奖励动画不能阻止继续操作。

## 错误处理

| 场景 | 服务端行为 | 前端行为 |
| --- | --- | --- |
| 义图未生成 | `404 asset_missing` | 显示 Emoji 或标题；普通文字题继续 |
| 依赖义图的题缺图 | 不加入新计划 | 不向孩子展示残缺题目 |
| 语音未生成 | `404 asset_missing` | 显示文字；提供重试，不阻止普通题 |
| 题目不存在、未审核或未发布 | 不加入新计划 | 提示当前主题暂无可练题目 |
| 解释内容缺失 | 正常判题 | 答后只显示正确答案，不显示空白解释框 |
| 重复提交 | 返回幂等结果 | 不重复播放奖励动画 |
| 计划已完成 | 返回当前结算 | 跳转结算页 |
| 数据库暂时不可用 | `503 database_unavailable` | 保留当前题和选择，允许重试提交 |
| MinIO 暂时不可用 | `503 asset_unavailable` | 显示降级内容并提供重试，不生成素材 |
| 内容在练习中被更新 | 继续使用计划固定题目 | 当前练习不换题 |

## 测试策略

### 共享学习模块

- 保留并迁移现有掌握度状态机测试。
- 验证答对、答错、两次尝试和复习到期。
- 验证相同 `client_id` 不重复写入、计分和奖励。
- 验证概念辨认技能正确汇总到知识点掌握度。

### science-server

- 验证所有查询限定 `subject = science`。
- 验证未发布题目不进入新计划。
- 验证正确答案不会出现在取题响应中。
- 验证计划生成后题目和选项顺序稳定。
- 验证同一知识点不会同时加入内容相同的 `fact1`、`fact2`。
- 验证作答事务要么全部成功，要么全部回滚。
- 验证安全类内容只有已发布状态才能进入计划。
- 使用 PostgreSQL 集成测试覆盖生产方言相关行为。
- 使用 MinIO 接口替身测试义图、字图和音频读取，不在测试中调用真实图片模型或 TTS。

### science-app

- 使用 Vitest 和 Testing Library 测试页面状态与答题交互。
- 测试义图、语音和解释缺失时的降级。
- 测试重复点击、答错重试、提交重试和页面恢复。
- 测试安全类题目答错后的明确反馈。
- 使用 Playwright 的 iPad 横屏视口验证首页、科普地图、知识卡、答题和结算主流程。
- 验证所有主要操作均可通过触摸完成。

### 跨系统验收

1. 内容后台发布一个科普知识点、问题、义图和语音。
2. `science-server` 在不调用内容后台 HTTP API 的情况下读取并输出内容与素材。
3. 停止内容后台后，孩子仍能打开已发布知识卡并完成练习。
4. 孩子完成一份科普练习。
5. 数据库出现对应的计划、作答、技能掌握度、知识点掌握度和每日统计。
6. 家长看板在不调用 `science-server` 的情况下展示该结果。

## 部署

- 根 `kid-workbench/docker-compose.yml` 增加 `science-server` 和 `science-app`。
- 两个新服务加入与 PostgreSQL、MinIO 相同的 Docker 网络。
- `science-app` 生产环境只配置一个 API 地址，即 `science-server`。
- `science-app` 提供 PWA manifest 和适合 iPad 主屏幕的图标，可添加到主屏幕后横屏全屏运行。
- `science-server` 使用现有数据库环境变量规范，并配置 MinIO 只读访问能力。
- 数据库迁移必须向前兼容；部署新服务不能要求先停止家长看板或内容后台。
- 健康检查区分进程存活与 PostgreSQL、MinIO 依赖状态。
- 不把 `content-admin` 保存的 `localhost:19091` 素材 URL 返回给孩子端。

## 实施顺序

1. 为共享学习事务补齐现有行为测试。
2. 提取 `shared-go`，保持家长后端测试和行为不变。
3. 增加科普内容审核、发布状态和向前兼容迁移。
4. 扩充科普内容模型，至少支持可选的解释与趣味知识。
5. 建立 `science-server`，完成健康检查、已发布内容读取和素材输出。
6. 完成科普计划生成、服务端判题、幂等作答和掌握度写入。
7. 建立 `science-app`，完成首页、科普地图和知识卡。
8. 完成答题、解释反馈、断点恢复和结算页。
9. 接入根 Compose，并进行 PostgreSQL、MinIO 集成验证。
10. 验证内容后台停机时已发布内容仍然可学。
11. 验证家长看板能读取新产生的科普学习结果。
12. 第二阶段补齐独立理解题后，再增加 `understand` 技能。

## 完成标准

- `science-app` 在 iPad 横屏上可以独立浏览科普知识卡并完成一份科普练习。
- `science-app` 只请求 `science-server`。
- `science-server` 不请求 `19091` 或 `19081` 的 HTTP API。
- 内容后台停机时，已发布的科普内容、素材和题目仍可被新系统使用。
- 科普服务只读取和修改 `subject = science` 范围内的数据。
- 未审核或未发布的科普内容不会进入新计划。
- 孩子端在提交答案前无法获得正确答案。
- 一次作答同时、正确地更新计划、尝试记录、技能掌握度、综合掌握度和每日统计。
- 重复提交不会重复计分或发放奖励。
- 素材缺失时有明确降级；依赖缺失素材才能成立的题目不会进入计划。
- 家长看板可以直接从共享数据库看到新产生的科普结果。
- 第一版只以真实的 `recognize` 题衡量掌握度，不把选项重排当作第二技能。
- 内容、学习和展示三类职责具有明确的数据写入边界。
