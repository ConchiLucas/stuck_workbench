# 拼音掌握页与 App 答题闭环开发方案

> **执行状态：已完成开发、验证及原端口部署。** 实际文件归并、验证方式和部署证据见 [验收记录](2026-09-12-pinyin-mastery-verification.md)。下文保留开发依据。

> 执行方式：后续使用 `executing-plans`，在当前任务内按下列清单逐项实施、验证和更新记录。本文为开发依据，本轮只编写方案，不修改业务代码、不执行迁移、不重新部署。未经明确要求不创建新任务、不提交其他任务的修改。

**Goal:** 将拼音 App 的四类练习结果真实记录到同一个孩子的学习账本，在进度后台展示四种题型进度、字母三项掌握地图和独立的音节拼读地图。

**Architecture:** 沿用 PostgreSQL、共享 `attempts / mastery_skills / mastery_states / daily_stats` 和现有掌握算法。字母使用 `listen / inword / shape`，音节使用 `blend`；生成题保存服务端快照，作答经服务端判分、幂等写入再更新界面。进度后台通过自身 API 读取统一数据，不跨端口拼接 App 的内存状态。

**Tech Stack:** Go、GORM、Gin、PostgreSQL / 测试 SQLite；React、TypeScript、React Query、Zustand、Vitest、Node test；Docker Compose。

---

## 1. 已确定的产品范围

### 1.1 页面结构

1. 标题「拼音掌握」，说明「从答题结果，看见字母认读与音节拼读的积累」。
2. 四种题型进度全部展开：听音选字母、字中找拼音、看形认读、声韵拼读。不使用下拉菜单。
3. 字母掌握地图：按当前内容配置的声母、韵母分组；直接切换「整体掌握 / 听音选字母 / 字中找拼音 / 看形认读」。
4. 音节拼读地图：同页独立区域，按韵母分组，展示实际配置的带调音节。所有分组可向下浏览，不藏入下拉框。
5. 搜索支持字母、带调音节、无调拼写；卡片悬浮看简要状态，点击看深色详情抽屉。

延续现有识字页深紫背景、粉色高亮、圆角和留白。默认不堆数量、百分比、正确率。需要的数值放在进度条悬浮和详情中。

### 1.2 字母与音节不混为一个考查对象

| 对象 | 适用题型 | 完全掌握条件 | 卡片表现 |
|---|---|---|---|
| 字母，例如 b、zh、ai | listen、inword、shape | 三项均 mastered 或 review_due | 「听｜找｜认」亮暗标记；全过显示 ✓ |
| 音节，例如 bā、bǎ | blend | 该音节 blend 为 mastered 或 review_due | 单一掌握亮暗；掌握显示 ✓ |

- 「看形认读」实际是看形后选择读音，不是麦克风跟读测评，不增加录音功能。
- 「声韵拼读」按具体带调音节记账；bā 与 bǎ 是不同对象。
- 不给字母增加第四个「拼」标记；拼对 bā 不自动点亮 b 或 a 的其他题型。
- 不将「答对一次」「累计答题多」直接等同于掌握。继续使用共享引擎现有配置，不另设前端阈值。
- 卡片题型标记只有亮、暗两态；未作答与已作答未掌握都暗，区别在悬浮、详情。
- 页面顶部题型条仍可三段：已掌握 / 已作答未掌握 / 未作答。条代表多个知识点的分布，和单卡亮暗不是同一层级。
- 暂不增加混合字母、音节的总环。题型条分别使用各自适用知识点为分母，不能直接平均四条百分比。

### 1.3 边界

- 本次调整进度后台拼音页、相关数据接口，以及拼音 App 的作答提交机制；App 四种题型的题面风格和主要交互保持。
- 识字页和总览布局不重做；统一数据接入后，总览原有拼音统计自然读取新结果。
- 不新增分钟、学习中、计划完成率、错因列表、薄弱点清单。
- 字详情保留作答事实，不加入后台「标记掌握」按钮。
- 不编辑题目后台的出题流程，不重建素材后台；不把素材表的主键直接当知识点主键。
- 当前种子只配置声母、韵母；不自行补齐课程清单或虚构整体认读音节进度。

## 2. 现状与必须补齐的缺口

| 已核查的文件 | 当前行为 | 本次处理 |
|---|---|---|
| `pinyin-app/src/pages/HomePage.tsx` | 首页四类题型均指向 `/practice/type/:type` | 保留入口和名称 |
| `pinyin-app/src/pages/DemoPracticePage.tsx` | 选项写入 Zustand 后跳题 | 接入服务端提交，确认成功后跳题 |
| `pinyin-app/src/pages/DemoResultPage.tsx` | 用前端 answerIndex 判分 | 使用服务端已接受的结果 |
| `pinyin-server/internal/quiz/service.go` | 四类题可生成，实例未持久化，返回答案 | 新建持久化正式出题路径，答案不随正式题面返回 |
| `shared-go/mastery/skills.go` | 拼音只有 inword、listen | 按模块区分字母三项和音节一项 |
| `parent-dashboard/backend/internal/service/dashboard.go` | Matrix / KpDetail 使用学科级技能列表 | 拼音按知识点所属模块计算 |
| `pinyin-server/internal/progress/service.go` | 进度固定遍历两种拼音技能 | 改为与共享模块规则一致 |
| `content-admin/backend/internal/db/db.go` | 音节素材在 pinyin_syllable_assets | 只读同步知识点映射，不接管素材管理 |

旧计划练习 `/practice/:planId` 已有正式入库路径，仍须兼容；不能误认为它代表首页四种题型都已入库。

## 3. 数据与掌握规则

### 3.1 技能定义

修改 `shared-go/mastery/skills.go`，固定展示顺序为 listen、inword、shape、blend；字母三项、音节一项单独定义。

```go
const SkillPinyinShape = "shape"
const SkillPinyinBlend = "blend"
const PinyinSyllableModule = "syllables"

var PinyinLetterSkills = []string{"listen", "inword", "shape"}
var PinyinSyllableSkills = []string{"blend"}
```

规则：`SkillsFor("pinyin", "syllables")` 返回 blend，拼音其他既有字母模块返回前三项。`SkillsForSubject("pinyin")` 返回四种候选技能，供识别题码；具体知识点的必需项一律调用 `SkillsFor(subject,module)`。逐个检查旧 `PinyinSkills` 调用方，不能用四项候选集合直接汇总字母。

`SkillFromQuestionCode` 允许识别四种拼音题码；`learning.ApplyOne` 的模块级校验必须拒绝给字母提交 blend、给音节提交 shape。

### 3.2 音节映射

新增 `pinyin_syllable_links`：

| 字段 | 约束与用途 |
|---|---|
| asset_id | 主键，对应音节素材 ID；不增加跨素材服务管理的外键 |
| kp_id | 唯一，外键到 knowledge_points |
| initial_text、final_text、tone | 保存结构化音节身份，三者唯一 |
| syllable_text | 带调显示文本 |
| enabled | 是否属于当前有效课程目录 |
| synced_at | 最近目录同步时间 |

在拼音学科新增 `syllables / 音节拼读` 模块，知识点 code 使用稳定的 `py-syllable-<asset_id>`。从现有素材复制标题、难度和拼读结构；不能重编号、重建已有字母。

同步采用事务内 upsert：相同素材重复执行不产生重复知识点；停用内容保留知识点和历史，只从当前新题候选及地图分母排除。同一素材 ID 的声母/韵母/声调身份发生变化时报告冲突，不把旧成绩转给新音节。仅修正文案、音频地址不改变掌握身份。

目录同步实现为共享 `shared-go/pinyincatalog` 包，由进度后台启动后、读取拼音目录前，以及拼音正式出题前调用；只读素材、写映射和音节知识点，不写孩子答题数据。素材表尚不存在时返回 `catalog_unavailable`，页面显示「音节内容暂不可用」，不能伪装为零掌握。表存在但无启用音节时返回正常空目录。

### 3.3 正式出题实例与提交回执

新增 `pinyin_quiz_instances`：

| 字段 | 用途 |
|---|---|
| id | 随机不可预测实例 ID，主键 |
| child_id | 所属孩子 |
| kp_id | 本题唯一考查知识点 |
| skill_code | 四种题型之一 |
| snapshot_version | 初始值 1 |
| public_snapshot | 固定题干、视觉内容、选项 ID/顺序/音频引用，不含答案 |
| answer_option_id | 服务端正确选项 ID |
| created_at、expires_at | 出题时间、未答实例有效期 24 小时 |

新增 `pinyin_answer_receipts`：

| 字段 | 用途 |
|---|---|
| child_id、client_id | 联合唯一幂等键 |
| instance_id | 唯一；每个实例只接受一次正式作答 |
| attempt_id | 唯一，外键到 attempts |
| skill_code、selected_option_id | 历史题型和孩子选择 |
| response_snapshot | 首次提交的完整返回结果，重试原样返回 |
| created_at | 服务端接受时间 |

实例持久化不能写 attempts 或增加任何进度。选项必须使用稳定 ID，提交时不信任客户端题型、知识点、答案和正确与否。

单次作答事务：校验孩子及实例归属 → 查询已有回执 → 校验有效期与选项 → 服务端判分 → 调用 `learning.ApplyOne` → 写回执 → 提交。沿用共享孩子行锁，锁顺序统一为孩子行在前，避免并发学习事务死锁。事务任一步失败必须全部回滚。

重复请求：同孩子、同 clientId、同实例、同选项返回原结果；同 clientId 换实例/选项返回 409；同实例换 clientId 也不得第二次入账。先检查回执再检查过期，成功提交后的网络重试即使跨过有效期也应成功恢复。

### 3.4 首次完全掌握与旧数据

新增 `pinyin_mastery_milestones`，主键 `(child_id,kp_id,rule_version)`，字段 `first_completed_at`；本版 `rule_version = 2`。

- 已有 listen、inword 技能行原样保留，shape 缺失即未作答；音节没有记录即未作答。
- 不把旧前端 picks 补写为正式成绩，没有可靠服务端记录的历史不回填。
- 新规则第一次全过时在同一事务插入 milestone；只在插入成功时计入本版首次新增掌握，重复答题、重新掌握、请求重试不重复计入。
- 旧字级掌握日期另存迁移审计快照，不作为三项全过日期显示。迁移时不制造“今天新掌握”。
- 老奖励保留；对于已有 `flower_ledger` 的该孩子/知识点 mastered 奖励，不因规则升级重复发奖。新知识点第一次完全掌握仍按现有奖励政策发放。相关判断只作用于拼音。
- 字级 `mastered_at` 在拼音分支取本版真实 milestone，不能取答得最多的技能日期；暂未全过则为空。详情日期来自本版 milestone。
- 迁移只重算拼音当前派生状态和本版首次完成记录，不重放 attempts，不修改旧 daily_stats、花朵余额、其他学科状态。
- 升级前的日历数据保留原历史口径，升级后的新增完全掌握用新规则；图例/提示中标注规则生效日期，不静默重写历史。

历史重算必须支持 `--dry-run` 报告：受影响孩子数、字母数、两项旧掌握改为部分掌握数量、已有奖励数、目录映射数。执行模式写入独立审计记录，重复执行无新增副作用。

## 4. API 契约

所有返回继续使用各服务已有成功/错误包裹，不再创设第三种包裹格式。

### 4.1 拼音服务：正式题面

`POST /api/v1/children/:childId/pinyin/quiz/generate`

```json
{"type":"shape","excludeTargetIds":[100,101]}
```

核心返回字段：`instanceId, type, targetId, kpId, expiresAt, stem, visual, options`。`targetId` 保持素材候选去重语义，`kpId` 才是统一知识点 ID；禁止把两者混用。`options` 每项包含 `id / label / speechText / speechUrl` 中适用字段，正式题面无 `answerIndex`。

旧 `/api/v1/pinyin/quiz/generate` 保持只读演示兼容，不参与成绩；App 正式入口全部迁到带孩子的新路径，避免旧 API 调用方立即失效。

### 4.2 提交与恢复

`POST /api/v1/children/:childId/pinyin/quiz/:instanceId/answer`

```json
{"clientId":"uuid","optionId":"123","costMs":1800}
```

ID 在新契约里统一转为字符串选项 ID，生成与提交保持同型；knowledge point ID 仍是数字。costMs 验证非负并上限 3600000，仅保留已有事实字段，不在进度页展示分钟。

```json
{
  "instanceId":"example",
  "attemptId":321,
  "selectedOptionId":"123",
  "correct":true,
  "answerOptionId":"123",
  "skill":{"code":"shape","status":"mastered"},
  "knowledge":{"kpId":100,"status":"mastered","newlyMastered":true}
}
```

`GET /api/v1/children/:childId/pinyin/quiz/:instanceId` 返回本人实例的公开题面及可选 acceptedResult；恢复已提交页面不能再次入账。未作答实例不暴露答案；素材后来调整不改变已有实例的选项文本和判分。

错误约定：400 无效题型/请求/选项；404 孩子或本人实例不存在（其他孩子的实例同样 404）；409 幂等内容冲突/实例已提交/素材不足；410 未提交实例过期；503 数据库或素材目录暂不可用。客户端失败停留当前题并可重试，不显示“已完成”。

### 4.3 进度后台

沿用现有 Matrix 和知识点详情路径，新增字段为向后兼容字段：

- `MatrixPoint.kind`: `letter | syllable`（其他学科可省略）。
- 拼音点返回本模块的 `skills`，字母三项、音节一项；音节另带 `initial / final / tone / syllable`。
- 拼音汇总含 `ruleVersion / ruleEffectiveAt` 和四题型的 `mastered / answeredUnmastered / unattempted / total`。
- `total = mastered + answeredUnmastered + unattempted`，每类只按适用、有效知识点计数。
- 详情历史优先从 pinyin_answer_receipts 识别题型，再回退旧 questions.code；避免 question_id 为空导致“题型未记录”。
- 所有历史查询按唯一 attempt_id 合并，不能因多表连接把一次作答变成多条。
- 作答统计从 attempts 聚合；不要使用现有字级 rollup 中“最多作答技能的 attempts”冒充总次数。
- 同一天新掌握、全科总览的拼音掌握数和拼音页使用相同有效目录与本版规则。页面布局不改，只修数据口径。
- 列表批量读取技能，禁止每个音节/字母单独发请求；空数组返回 `[]`，前端也兼容历史 null。

## 5. 页面交互细节

### 5.1 顶部题型条

四条等宽轨道，桌面并排，窄屏两列。颜色沿用识字页：掌握 `#f45b9b`，已答未掌握 `#885c80`，未作答 `#372e45`。悬浮显示名称及三类数量。零分母显示空轨道和「暂无内容」，不能出现 NaN 或假进度。

点击前三条切换字母视图并定位字母地图；点击拼读条定位音节地图。键盘可操作；减少动态效果偏好下直接定位，不强制平滑滚动。

### 5.2 字母地图

按真实模块顺序展示声母、韵母，保留拼音字形与 ü；每组右上显示细掌握条。搜索过滤卡片但不改变整组进度分母。多字母拼音如 zh、ang 不拆成多个知识点。

`听｜找｜认` 标记顺序固定，亮为掌握、暗为未掌握。题型视图只改变当前突出状态，位置、分组稳定。整体三项全过才显示 ✓，部分掌握可用较浅卡片底色。

### 5.3 音节地图

按标准化韵母分组，组内按声母、声调、稳定 ID 排序；韵母组标签用于阅读，不能更改资产身份。带调 final 的归组需显式拼音映射，不能一律删除所有附加符导致 ü 与 u 混淆。

卡片主文本为音节，例如 bā；悬浮显示 b + ā、掌握状态和最近作答；单一 blend 状态无需再叠三个标记。空音节区不放示例进度，显示内容为空或目录不可用的真实原因。

### 5.4 搜索与详情

搜索对大小写、声调符号做专门拼音归一化；ü 保持与 u 区别，并支持 v / u: 作为 ü 的输入别名。无调 ba 能匹配 bā/bá/bǎ/bà，输入具体 bǎ 优先精确匹配；搜索字母和音节均有效。

详情抽屉：标题/所属分组 → 适用题型状态与各自作答次数、正确率 → 有可靠 milestone 才显示首次完全掌握日期 → 时间倒序作答记录。零作答正确率为 —。Escape 关闭、Tab 焦点约束、关闭后回到原卡片，窄屏不横向溢出。

卡片悬浮不展示错因或错选分析；历史深度分析仍归孩子知识库。已有隐藏的后台拼音测验弹层和手动掌握入口不放入新拼音页；不删除其他学科使用的通用组件。

## 6. 文件与实施任务

以下路径相对仓库根目录。现有工作区有其他任务的修改，开始每项前先检查相关 diff；禁止 reset、批量 git add 或覆盖其他任务。迁移编号以实施时最新编号的下一号为准，下面预定 015；若已被占用，两个数据库目录同时顺延，不改已发布迁移。

### 任务 1：建立规则和契约测试

**修改** `shared-go/mastery/skills.go`、`skills_test.go`。
**新增** `shared-go/pinyincontract/types.go`、`types_test.go`。

- [ ] 先写失败测试：拼音字母三项、音节一项、缺 shape 不完整、review_due 仍掌握、跨模块技能被拒绝。
- [ ] 定义 `GeneratedQuestion / AnswerRequest / AnswerResult / InstanceSnapshot`，JSON 字段严格匹配第 4 节；公开结构无 answerIndex。
- [ ] 实现模块技能规则，逐一修正所有 PinyinSkills 引用，其他学科行为不变。
- [ ] 在 shared-go 目录运行 `go test ./mastery ./pinyincontract`，通过后再接数据库。

测试核心断言示例：

```go
if got := mastery.SkillsFor("pinyin", "shengmu"); !reflect.DeepEqual(got, []string{"listen", "inword", "shape"}) { t.Fatalf("letter skills: %v", got) }
if got := mastery.SkillsFor("pinyin", "syllables"); !reflect.DeepEqual(got, []string{"blend"}) { t.Fatalf("syllable skills: %v", got) }
if mastery.IsSkillDone(mastery.RollupSkills([]mastery.Status{mastery.StatusMastered, mastery.StatusMastered, mastery.StatusNotStarted})) { t.Fatal("missing shape cannot complete a letter") }
```

### 任务 2：迁移、音节映射和历史审计

**新增** `parent-dashboard/backend/internal/db/migrations/{postgres,sqlite}/015_pinyin_mastery.sql`、`parent-dashboard/backend/internal/db/pinyin_mastery_test.go`。
**新增** `shared-go/pinyincatalog/{sync.go,sync_test.go}`、`shared-go/model/pinyin.go`。
**新增** `parent-dashboard/backend/cmd/pinyin-upgrade/main.go`、`parent-dashboard/backend/internal/service/pinyin_upgrade.go`、`pinyin_upgrade_test.go`。

- [ ] 编写迁移与同步失败测试：首次建表、重复迁移、重复映射、内容停用、身份变更冲突、素材表不存在。
- [ ] 创建第 3 节四张业务表和 `pinyin_upgrade_audits` 审计表；补外键、唯一键及 child/kp/time 查询索引。SQL 不使用当前迁移器不能处理的分号函数体。
- [ ] 实现目录同步，所有 upsert 冲突通过唯一约束处理，支持多服务并发同步。
- [ ] 实现升级命令 `--dry-run` 与 `--apply`，记录升级前拼音状态 JSON、原日期和升级时间；只重算拼音派生状态，不写 attempts/daily_stats/flower_ledger。
- [ ] 在 SQLite 自动测试和独立临时 PostgreSQL 数据库验证升级；不拿真实孩子作答作测试。
- [ ] 验证重复执行 apply 不改变第二次报告中的迁移数据、不产生新奖励。

### 任务 3：拼音完整掌握与统计一致性

**修改** `shared-go/learning/service.go`、`service_test.go`。
**新增** `shared-go/learning/pinyin_mastery.go`、`pinyin_mastery_test.go`。

- [ ] 写事务测试：前两项全过但 shape 未过，新增掌握为零；第三项真正达标后只增加一次。
- [ ] 在拼音分支接入本版 milestone，首次完成时间使用事件服务端时间；字级状态由必需技能收敛。
- [ ] 写重复、并发、先失稳再掌握测试，确保回执重试、重新掌握不重复发放首次奖励。
- [ ] 迁移审计中的历史奖励按 flower_ledger 对应 child/kp 的 mastered 记录校验，不能把余额高低当作是否领过该知识点奖励。
- [ ] 对旧计划练习和新生成练习使用相同学习服务；保留其他学科已有逻辑，运行 shared-go 全部测试。

事务约束伪代码：

```text
ApplyOne(pinyin answer):
  record attempt once
  update only the applicable skill
  roll up required skills for this module
  if fully mastered and no milestone for rule v2:
    insert milestone once
    increment today's new mastery once
    grant mastery reward only if this KP has not already received one
  update today's attempts/correct once
```

### 任务 4：正式生成题与服务端判分

**修改** `pinyin-server/internal/quiz/service.go`、`internal/http/{handler_quiz.go,router.go}`、`cmd/server/main.go`。
**新增** `pinyin-server/internal/quiz/{instances.go,answer.go,answer_test.go}`。
**扩展测试** `pinyin-server/internal/http/router_test.go`。

- [ ] 先写 API/事务失败测试：四种正式题型生成、题面不含答案、实例绑定孩子、错误选项不落库。
- [ ] 复用现有出题候选与视觉构造，在新路径保存快照；blend 通过映射解析 kpId，不能使用素材 ID 直接调用 ApplyOne。
- [ ] 实现第 4 节三个带孩子接口与完整错误映射；已有无孩子演示生成接口保留。
- [ ] 实现第一次提交、重复提交、同键不同内容、跨孩子访问、并发重复、过期、已答过期重试测试。
- [ ] 测试生成后素材标题/答案相关源内容变化，实例判分仍使用冻结快照；停用素材不继续出新题，已发有效实例仍可提交。
- [ ] 测试回执写入失败整个事务回滚；同时验证 attempts、技能、日统计、奖励无半完成状态。
- [ ] 在 pinyin-server 运行 `go test ./...`。

### 任务 5：App 提交、重试与刷新恢复

**修改** `pinyin-app/src/api/{types.ts,pinyin.ts}`、`src/pages/{DemoPracticePage.tsx,DemoResultPage.tsx}`、`src/store/{demoQuizStore.ts,demoAnswerStore.ts}`。
**新增** `pinyin-app/src/store/pinyinPracticeSession.ts`、`pinyinPracticeSession.test.ts`、`src/pages/PinyinAnswerFlow.test.tsx`。

- [ ] 写失败测试：点击后等待服务器确认才跳题、失败可重试、连续点选不重复、正确与否来自响应。
- [ ] 新练习会话按 childId + type 隔离，保存实例 ID 列表、当前位置、未确认提交的 clientId/optionId；存储键带版本。
- [ ] 选项 ID 与题目 ID绑定，重试必须使用原 clientId；失败时冻结本次选择，不能改选后继续复用幂等键。
- [ ] 用已接受的响应替代前端 answerIndex 计算；结果页展示服务端正确数，跳过题显示未作答且不入账。
- [ ] 刷新后通过实例 GET 核对服务端结果；不能只信本地 accepted 标志。成功但响应丢失时恢复原结果，不生成新提交。
- [ ] childId 切换、重新练习、异步旧请求返回时隔离会话；取消/过期请求不得覆盖新孩子页面。
- [ ] 已提交题回看只读；重新练习产生新实例，可正常积累练习次数。
- [ ] 保留 shape/blend 的播放读音再选择交互；这类选项试听是题型本身，不当作额外提示而误判 assisted。
- [ ] 提交成功后失效当前孩子的 home/progress 缓存；运行 `npm test -- --run` 和 `npm run build`。

### 任务 6：后台统一目录、摘要和历史

**修改** `parent-dashboard/backend/internal/service/dashboard.go`、`stats.go`、`internal/http/handler_dashboard.go`；`pinyin-server/internal/progress/service.go`、`internal/catalog/repository.go`、`internal/home/service.go`。
**新增** `parent-dashboard/backend/internal/service/{pinyin_progress.go,pinyin_progress_test.go,pinyin_history_test.go}`。

- [ ] 写字母/音节混合矩阵测试、空目录测试、无技能行测试、停用内容分母测试。
- [ ] Matrix/KpDetail 查询模块 code 并按行选技能集合；批量获取相关 mastery_skills，不使用一个全学科技能数组算所有点。
- [ ] 字母 API 目录继续只返回字母，避免旧 App LearnPage 把音节当字母播放；新后台矩阵包含明确 kind 的音节。
- [ ] 四题型汇总与分组进度使用有效目录；当前 Subjects/Overview 的拼音计数使用同一结果，不改主页布局。
- [ ] 历史查询接新回执，返回正确题型、实际选择和快照事实，兼容旧 question_id；日记录中也能识别新生成题的来源。
- [ ] 规则生效日期从审计读取，首次掌握日期从 milestone 读取，不推算旧日期。
- [ ] 扩展旧计划兼容测试，候选依旧限制为已有 listen/inword 题，不给旧题面塞入不支持的音节交互。
- [ ] 运行 parent-dashboard/backend 和 pinyin-server 的全部 Go 测试。

### 任务 7：拼音页与只读详情

**新增** `parent-dashboard/frontend/src/lib/pinyinMastery.ts`、`tests/pinyinMastery.test.mjs`。
**新增** `parent-dashboard/frontend/src/components/mastery/pinyin/{PinyinMastery.tsx,PinyinTypeSummary.tsx,PinyinLetterMap.tsx,PinyinSyllableMap.tsx,PinyinDetailDrawer.tsx,pinyin.css}`。
**修改** `src/pages/SubjectDetail.tsx`、`src/components/mastery/KpDetailDrawer.tsx`、`src/components/layout/Shell.tsx`、`src/api/types.ts`。

- [ ] 为纯展示模型写失败测试：三技能全过、两种暗态、音节单技能、分母为零、搜索声调/ü、搜索不改变分组进度。
- [ ] 实现数据归一化与类型守卫；缺失技能、null modules、未知类型不崩溃，不用演示数字填空。
- [ ] 实现第 5 节全部布局及搜索；主入口仅 pinyin 分支使用新组件，其他学科不受影响。
- [ ] 四条进度复用识字页颜色和视觉尺寸；仅提取确实相同的条形基础组件，不大规模重构识字页。
- [ ] 实现抽屉与焦点恢复；数字和正确率只在悬浮/详情出现；去掉拼音旧五状态摘要与手动标记入口。
- [ ] 数据刷新：后台可见页面的拼音查询每 30 秒刷新，窗口重新聚焦立即刷新；离开拼音页、抽屉关闭后停止相关轮询。
- [ ] 检查 1280px、768px、390px 的排列、标签、详情和键盘，字母与音节全部可浏览。
- [ ] 在 parent-dashboard/frontend 运行 `npm test`、`npm run lint`、`npm run build`。

### 任务 8：端到端验证与代码审查

**新增** `pinyin-app/e2e/pinyin-mastery-flow.spec.ts`；更新既有 App 测试，不将正式孩子作为测试对象。

- [ ] 隔离测试数据中完成 listen/inword，后台字母仍不全亮；完成 shape 达到阈值后显示 ✓。
- [ ] 完成 bā 的 blend 后只点亮 bā，不点亮其他声调，也不改变 b/a 的认读题型。
- [ ] 模拟提交成功但响应丢失，重试后作答、日统计和奖励都只增加一次。
- [ ] 刷新 App 后结果保持；切换孩子看不到另一孩子会话。
- [ ] 旧计划答题仍入库、旧作答历史可查看；识字三个技能和抽屉回归通过。
- [ ] 迁移前后真实数据只做只读核对，统计差异必须能由新增 shape 要求或有效音节目录解释。
- [ ] 执行只读代码审查，修复实质问题后复测涉及路径；记录测试命令和结果。

注意现有 `pinyin-app/playwright.config.ts` 指向本地 19112；自动提交测试必须使用路由模拟或隔离测试后端，不直接对日常使用的容器执行写入用例。

### 任务 9：仅替换修改服务，沿用原端口

**修改** `parent-dashboard/backend/Dockerfile`：打包新的 pinyin-upgrade 命令，供新镜像一次性执行历史重算。仅增加可执行文件，不更改默认服务命令。

- [ ] 记录当前 backend、pinyin-server、pinyin-app 的容器 ID、镜像 ID，以及全部运行容器的启动时间。
- [ ] 构建这三个镜像：`docker compose build backend pinyin-server pinyin-app`。构建失败时原容器继续服务。
- [ ] 对新增迁移和 dry-run 结果做备份/核对；只使用数据库备份命令，不重启 PostgreSQL。
- [ ] 短暂停止这三个将被替换的服务，防止旧拼音写入规则与新规则同时运行；其他服务不停止。
- [ ] 新 backend 启动并完成加法迁移：`docker compose up -d --no-deps --no-build --force-recreate --wait backend`。
- [ ] 通过新 backend 镜像内 `pinyin-upgrade --dry-run` 核对后执行 `--apply`；过程只重算拼音派生状态，并保留审计和备份。执行命令使用现有数据库配置，不输出连接密码。
- [ ] 更新拼音服务：`docker compose up -d --no-deps --no-build --force-recreate --wait pinyin-server`。
- [ ] 更新拼音 App：`docker compose up -d --no-deps --no-build --force-recreate --wait pinyin-app`。
- [ ] 原端口核验：后台 19081、拼音服务 19111、拼音 App 19112；页面加载、目录查询、详情、健康检查正常。正式环境不做测试作答。
- [ ] 对比容器 ID/启动时间，只有这三个服务变化；数据库、识字、题目后台、素材后台、知识库等未重启。
- [ ] 新服务验证正常后删除这三项本次替换下来的旧镜像；若镜像仍被其他容器引用，不强制删除共享镜像，记录引用关系。不运行全局 image prune / compose down / 删除卷。
- [ ] 关闭本任务临时预览进程，最终只交付原端口。清理仅针对本任务资源。

部署故障：停止继续清理旧镜像；保留新增表，先恢复已验证镜像。若需恢复旧拼音写入逻辑，先暂停拼音入口，依据升级审计恢复旧派生快照；新产生的真实答题记录不可删除，需重算验证后恢复入口。不能仅回滚前端就宣称全部恢复。

## 7. 完成验收清单

- [x] App 四种题型均由服务端判分、正式入账，页面刷新和重试不丢失结果、不重复计数。
- [x] 字母三项全过才显示 ✓；音节单独判断；卡片题型只有亮暗两态。
- [x] 四题型进度全部展开，字母与音节均按真实目录展示，无下拉选择学科、无假数据。
- [x] 汇总、卡片、详情、App 进度对同一孩子一致。
- [x] 历史两项成绩保留，shape 未作答不补造；旧奖励和旧日历不因升级重放。
- [x] 正式题面不返回答案；无效/跨孩子/重复/过期提交都有确定行为。
- [x] 搜索、悬浮、抽屉、错误和空状态、窄屏与键盘完成验证。
- [x] 识字与其他学科回归正常，未修改其产品结构。
- [x] 仅更新三个修改服务，沿用原端口，旧容器替换、旧镜像清理有记录。

## 8. 执行顺序与交付记录

按任务 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 执行。每项先写关键失败测试、最小实现、验证，再推进；遇到当前代码与本文不一致，先核实新改动并在本文件记录具体调整，不覆盖并行任务。

最终交付应包含：修改文件范围、测试结果、迁移前后摘要、仅三个服务发生替换的证据、原 19081 拼音页链接。无需用户再次选择开发模式；后续明确开始开发时按本方案内联实施。

**方案自检（本轮）**：已覆盖四题型与两类考查对象、旧两题型兼容、服务端快照判分、幂等、首次掌握、目录停用、历史与奖励、UI、自动测试和原端口部署；本轮仅生成方案，全部实施项保持未完成。


## 实施进展（2026-09-12）

- 已实现四型正式生成/提交/恢复、三技能字母与独立音节、服务端快照和回执、音节映射、首次掌握、升级审计及新拼音页。
- shared-go 与 parent backend 全量 Go 测试通过；独立 PostgreSQL 的015迁移、初次目录同步总数、音节掌握和升级保留里程碑测试通过。
- App 39项单测、9项隔离WebKit E2E通过；进度页11项测试、1280/768/390px检查通过。所有浏览器写入用例使用内存模拟，正式孩子数据未参与测试。
- 只读审查修复：初次音节同步总数、not_started显式状态漏计、升级覆盖新里程碑、QuestionID推导后的模块技能校验、历史回执快照缺失。
- 实现补充：正式选项ID使用实例内option-N，拼读公开题面不包含答案音节；旧无题型作答只保留事实，不作为掌握证据。
- 旧demo重算器不再用未分类拼音事实推导单行掌握，保留拼音当前派生状态；正式升级使用pinyin-upgrade，禁止用demo重放修改正式账本。
- 收尾已完成：真实PG并发提交通过；跨孩子且URL带题号的2项新增回归通过；三个服务原端口部署与只读验收通过；测试库已清理。
- 原始任务清单保留逐项实施描述，最终完成状态以第7节和验收记录为准；历史查询文件归并和分层测试方式见验收记录。
