# 识字 App、素材后台、题目后台三端统一开发计划

> **2026-09-12 用户范围修正（优先于下文原计划/验收记录）**：本阶段对齐每个题型的题面、图片、读音位置、交互与反馈，样式以 App 原有页面为准、图片以素材后台为准。App 保留原首页和直接练习流程，数据可独立组织，不要求领取题目后台任务，也不以端到端同步作为本轮验收条件。已实现的任务领取、快照与回执作为可选后续能力保留，不再强制接入首页三个题型入口。原 App 听写本地评估与后台版本化评估尚非同一策略，后续效果对齐需单独比对，不宣称全链路完全一致。


> **For agentic workers:** 使用 `executing-plans` 按任务执行。本计划已依据用户明确要求作出默认技术选择，不逐项询问实现细节。核心功能现已实施部署；以下按实际证据勾选，完整保障项未完成部分保留。

**Goal:** 三个端使用一致的题型定义、练习组件、素材版本与作答协议，补齐现有“写一写”，让后台试做和孩子实际练习一致。

**Architecture:** 以识字 App 现有练习界面提取共享 React 组件，以素材后台作为图片、音频、书写模板的唯一业务来源；题目后台负责选择素材、配置任务和冻结题目。学习服务统一判定与记录，后台预览使用同一评估逻辑但不写学习账本。

**Tech Stack:** 保留 React、TypeScript、Vite、Go、Gin、GORM、PostgreSQL、现有对象存储与 Hanzi Writer。不引入大模型判卷、消息队列或重做整套后台。

---

## 1. 必须遵守的统一规则

| 内容 | 以谁为准 | 实际约束 |
| --- | --- | --- |
| 练习布局、字体、田字格、按钮、选项、反馈、书写手势 | 识字 App | 从 App 提取共享组件，三端不能各复制一份样式 |
| 字图、义图、音频 | 素材后台 | 使用其实际资产及不可变版本，不在后台自画、不用 emoji、示例图或浏览器合成语音替代正式资源 |
| 标准字形、笔画中线、笔顺数据 | 素材后台 | 补充素材类型，导入、校验后保存版本；练习端不临时拉第三方数据作为判题依据 |
| 题型代码、素材要求、响应类型 | 共享题型契约 | 三端从一份定义生成代码，禁止分别维护题型名单 |
| 题量、范围、配比、发布版本 | 题目后台 | 生成不可变题目快照，App消费已发布任务 |
| 真实作答判定、完成状态与回执 | 学习服务 | 服务端判定，预览不记账；进度后台和孩子知识库职责不变 |

“样式以 App 为准”指三个端的**练习区域**。素材编辑表单、任务列表等管理外壳保留后台布局；练习区域必须继承 App 的底色、字号、间距、图片呈现及交互，不能被后台全局 CSS 覆盖。

“图片以素材为准”同时包括来源和呈现：保持原图比例，默认完整展示；不拉伸、不随意裁切、不把素材字图替换为另一个字体。若现有 App 展示与素材原图不一致，优先修 App 的媒体接入，再让三端共同使用。

## 2. 已核实的现状及范围

- `literacy-app/src/pages/HomePage.tsx` 的三张题型卡片进入 `DemoPracticePage.tsx`，目前是固定示例；真实题包入口是 `/tasks`。
- `PracticePage.tsx` 的 `write_char` 使用 `ListenWritePad`：听音自由书写，前端字形匹配成功后提交 `optionIndex:0`。
- `WriteCharPad.tsx` 另有按笔顺书写组件，但不能据此认为题包已支持描写。
- `lib/handwriting.ts` 使用 Hanzi Writer 读取字符数据做覆盖率匹配，尚无素材版本绑定。
- 题目生成和新回执以四选一为基础；素材能力检查也仅覆盖两种选择题。
- 当前保留8组120道选择题草稿。旧题包已按用户要求删除，本次不重新创建旧题包或改写这些新题。

本轮必须闭环的三种题型：`glyph_sense` 看字选义、`sense_char` 看义选字、`write_char` 听音写字。首页可继续使用“写一写”作为友好名称，后台说明其具体交互是听音自由书写。

描写、按笔顺默写是后续独立模式，本轮不把未接通能力显示为可发布。它们可以在能力表标注“未接入”，不能与当前听写的完成含义混用。识字图是导航和积累展示，不列为可生成题型。

## 3. 方案选择

采用“共享契约＋共享练习组件＋版本化素材”。

- 不采用三端分别实现：开始改动较少，但后续样式、题型和判题继续分叉。
- 不采用把 App 整页 iframe 嵌入后台：虽然外观接近，但草稿预览、容器尺寸、错误处理和只读调试会复杂化。
- 不将三个项目合并为单个应用：保留部署边界，仅共享需要一致的协议和组件。

## 4. 契约与数据设计

### 4.1 题型注册与能力

新增 `contracts/literacy/question-types.json` 作为人工维护源，使用 `scripts/generate-literacy-contracts.mjs` 生成 TS/Go 常量、版本标识和基础校验。生成产物提交到仓库；`--check` 只比对，发现手工漂移退出非零。

```json
{
  "contractVersion": 2,
  "types": [
    {"code":"glyph_sense","label":"看字选义","interaction":"choice","materials":["glyph","sense","speech"]},
    {"code":"sense_char","label":"看义选字","interaction":"choice","materials":["sense","glyph","speech"]},
    {"code":"write_char","label":"听音写字","interaction":"handwriting","materials":["speech","writing_template"]}
  ]
}
```

素材要求按题干、正确项和干扰项分角色展开实现；上表是题型总需求，不能简单要求每个字都具备所有素材。听写不要求义图，不要求至少4个候选。

`GET /api/v1/question-types/literacy` 返回契约版本和类型；素材候选返回逐题型 `ready/reasons`；题目后台组合素材就绪、生成器、App呈现及服务端评估支持情况决定是否允许生成发布。部署就绪检查来自实际服务声明的版本，不凭前端一个硬编码 `supported=true` 开放。

### 4.2 题目快照

保留已存在 schemaVersion=1 选择题读取，新增 schemaVersion=2 判别联合结构。公共字段仍有kpId、题型、模板版本、素材修订、提示语；新增 `interaction`、`presentation`、`responseSchemaVersion`、`evaluationPolicyVersion`。

```ts
type ResponsePayload =
  | { kind: 'choice'; selectedOptionId: string }
  | { kind: 'handwriting'; strokes: {x: number; y: number; t: number}[][]; hintsUsed: number }

type EvaluationResult = {
  outcome: 'passed' | 'not_passed'
  assistance: 'none' | 'hinted'
  evaluatorVersion: string
  metrics: Record<string, number>
}
```

选择题有options和服务端answer；听写有音频、书写模板引用及评估策略，没有虚构的options/answerIndex。后台admin DTO可显示答案与标准字；孩子DTO按题型裁剪。听写不能通过`targetText`、字符文件名、图片alt或播放按钮标签把答案直接展示出来。

第一版听写评估放服务端，模板通过revision关联在服务端读取，孩子端只需空白书写板和音频。将来描写需要公开标准字形时使用独立DTO规则。

### 4.3 素材、评分与回执

- `writing_template` JSON保存标准字、坐标系、笔画路径及中线、导入来源、校验状态和版本；保留来源许可说明，不把第三方动态URL当最终资产。
- 归一化坐标在0–1；最多64笔、每笔512点、总8192点；仅有限数值，时间非递减；请求上限512KiB。超限明确400/413，不静默截断影响评分。
- 将现有字形覆盖率算法移植到 `shared-go/handwriting`；用共同fixture校准，保留当前64网格及阈值作为显式策略v1，不宣称等同于专业书写评分。阈值调整必须新建评估策略版本。
- 空白、缺笔、明显乱涂不通过；模板缺失或评估服务故障返回503，不记为孩子答错。
- 新回执补充 response_kind、response_json（作答内容使用独立列名如answer_payload_json，不能覆盖现有响应回放列）、evaluation_json、evaluator_version；选项字段对书写允许NULL，旧选择题保留。
- 书写回执记录规范化笔迹及真实评估结果。原有幂等响应JSON继续保存；同clientId比较题目与规范化作答payload摘要，costMs重试差异不重复记账。
- 一次明确点击“写完了”作为一次提交；不同笔迹重写用新clientId，网络重放沿用旧key。经过提示完成仍记录为hinted，不自动把它当独立掌握；具体掌握规则由学习服务处理，题目后台不直接写掌握度。
- 后台试做通过自己的预览API调用同一Go评估包，读取冻结模板但不创建attempt、plan、receipt、mastery。浏览器不跨端口访问学习服务。

## 5. 文件职责地图

| 文件/目录 | 用途 |
| --- | --- |
| `contracts/literacy/question-types.json` | 唯一题型配置 |
| `scripts/generate-literacy-contracts.mjs` | 生成及漂移检查 |
| `packages/literacy-contract/` | TS协议、校验和共同fixture |
| `shared-go/literacycontract/` | Go协议、生成类型和校验 |
| `packages/literacy-player/` | 从App提取的练习组件与作用域样式 |
| `shared-go/handwriting/` | 服务端纯书写评估及fixture测试 |
| `content-admin/backend/internal/literacy/writing_templates.go` | 模板导入校验、就绪与修订 |
| `content-admin/frontend/src/features/literacy/` | 逐题型素材检查及App样式试做 |
| `task-admin/backend/internal/generation/` | 按interaction分发生成器 |
| `task-admin/backend/internal/taskgen/` | 混合题包生命周期、复习与预览 |
| `task-admin/frontend/src/pages/GenerationPage.tsx` | 任务管理，练习区域交给共享播放器 |
| `literacy-server/internal/plan/`、`internal/practice/` | 新旧快照、分型DTO、判题与回执 |
| `literacy-app/src/pages/` | 正式题型入口和共享播放器宿主 |

共享player对外只接收题目、媒体解析器、提交回调、状态及评估反馈；不依赖三个应用的router、QueryClient、孩子store或数据库。React作为peer dependency，由宿主提供，避免重复React。

## 6. 分阶段开发清单

### Task 0：建立基线和共同验收样本

**修改**：根目录 `task_plan.md`、`findings.md`、`progress.md`；新增 `docs/verification/literacy-three-surface-unification.md`。

- [x] 记录当前Git状态，保留用户已有改动，不运行git add .，不混合提交。
- [ ] 记录App当前选择题与听写在390×844、1024×768的截图，作为视觉基准。
- [x] 准备独立验收库及专用孩子；保存两种现有题包和一个旧学习计划fixture。
- [x] 准备“山、水、一、的”素材fixture：正常、缺音频、缺义图、缺模板、模板损坏各一例。
- [x] 分别在三个前端运行 `npm test`、`npm run build`；六个相关Go项目运行 `GOCACHE=/tmp/kid-workbench-go-cache go test ./...`。记录实际既有失败及监听权限限制。

通过标准：现有120题及学习历史能保持原样；后续视觉比较有明确App基准。

### Task 1：统一题型契约与能力查询

**创建**：上述contracts、生成脚本、TS/Go契约包及各自测试；**修改**：三个前端API类型、素材/题目/学习服务能力handler。

- [x] 先写契约测试：三个代码可识别；未知代码拒绝；听写不要求options；旧v1选择题仍可解析。
- [x] 实现JSON→TS/Go生成器，禁止调用网络；添加 `node scripts/generate-literacy-contracts.mjs --check`。
- [x] 增加题型能力查询，列出ready、material_missing、renderer_unavailable、evaluator_unavailable等明确原因；新题型未完成后续任务时保持不可发布。
- [x] 将题目后台表单的固定两种题型列表改为契约驱动，仍保留未接通状态，不提前生成书写题。
- [x] 运行契约生成检查、Go契约测试和三个前端类型检查。

断言示例：`validateResponse('write_char', {kind:'choice',selectedOptionId:'kp:1'})` 必须失败；`requiredMaterials('write_char')` 不含sense。

### Task 2：从App提取唯一练习播放器

**创建**：`packages/literacy-player/package.json`、`src/LiteracyPlayer.tsx`、`src/ChoiceQuestion.tsx`、`src/HandwritingQuestion.tsx`、`src/player.css`、`src/LiteracyPlayer.test.tsx`。

**修改**：App的 `PracticeStage.tsx`、`Tianzige.tsx`、`OptionTile.tsx`、`FreeDrawPad.tsx`、`ListenWritePad.tsx`、`styles/tokens.css` 及三个前端package/lock/Vite/TypeScript配置。

- [x] 先锁定App已有交互测试：选项顺序、图片完整显示、音频按钮不误提交、笔迹清空、提交锁定、失败后可重试。
- [x] 从App原样提取相关布局和设计变量。以 `.literacy-player` 作用域包装所有选择器，CSS变量统一加前缀；组件不读取body布局。
- [x] 提供 `mode='practice'|'preview'`，模式仅改变宿主提交处理，不改变题干、选项、字号或正确提示逻辑。
- [x] 图片统一用媒体resolver返回的素材URL，使用App基准的contain规则；缺资源显示错误与重试，正式题不能回退emoji、字体图或TTS。
- [ ] App先接共享组件，并与基准截图比较后再接后台；小屏按同样断点重排，不把桌面画布整体缩小导致笔迹坐标失真。
- [x] 运行player测试、App全量测试/构建；验证键盘选择、触屏落笔和pointer capture。

### Task 3：打通共享包在三个项目中的构建

**修改**：`literacy-app/Dockerfile`、`content-admin/Dockerfile`、`task-admin/Dockerfile`、根与独立compose配置、相关`.dockerignore`。

- [x] 使用私有本地包依赖，锁定构建输入；三个前端不分别复制player源码。共享包不启动独立服务。
- [x] content-admin/task-admin当前独立目录构建上下文不能读取根packages，改为kid-workbench根上下文，并同步Dockerfile COPY、Go shared-go本地replace及各独立compose相对路径。
- [x] 后台Go模块需要共享契约/评估包时，参照现有literacy-server使用shared-go的方式引入，不另复制算法。
- [x] 三个前端各自 `npm ci && npm run build` 成功；`docker compose build content-admin task-admin literacy-server literacy-app` 成功。

通过标准：本地Vite和Docker产物使用相同共享源，不能只在开发机路径别名下工作。

### Task 4：素材后台补齐书写数据和逐题型就绪状态

**创建**：`writing_templates.go`、`writing_templates_test.go`、HTTP模板导入handler；**修改**：`internal/literacy/materials.go`、`internal/db/db.go`、前端 `features/literacy/LiteracyPage.tsx`、`api/literacyTypes.ts`。

- [x] 先写失败用例：没有义图但有音频和模板的“的”可以听写；缺模板不能听写；无音频不可用；字符与模板不一致拒绝。
- [x] 新增素材拥有的模板表，支持单字导入、版本详情、错误原因及标准笔画预览；外部来源只在明确导入步骤访问，服务运行不依赖在线第三方字符数据。
- [x] 校验JSON结构、坐标/路径数量、有限数值、字符对应关系、最大2MiB模板大小；存储来源和许可信息。损坏数据不标为ready。
- [x] 冻结接口新增按题型/角色声明所需资源，书写只冻结音频与模板，不被缺义图阻断；兼容旧请求默认选择题素材组合。
- [x] 复用现有不可变对象摘要与并发协调机制。更新模板后旧revision不变，导入失败不替换旧可用模板。
- [x] 素材页分列三种题型的“可出题/缺什么”，增加使用共享player的试做。选择题试做需要同组有效干扰项，不能随机塞示例图；能力不足时明确显示。
- [x] 素材试做通过题目后台的只预览构建接口生成临时快照；接口只接受kpId/题型/已验证revision，不接受外部图片URL，不落question_tasks。
- [x] 运行素材Go测试与前端测试，验证更新素材前后旧题包仍使用旧字节。

### Task 5：实现有版本的书写评估与通用作答回执

**创建**：`shared-go/handwriting/evaluate.go`、`evaluate_test.go`、`literacy-server/internal/practice/handwriting.go`、`handwriting_test.go`；在parent-dashboard双数据库迁移目录新增下一未占用编号的迁移（执行时确认，禁止覆盖既有迁移）。

- [ ] 共同fixture覆盖正确字、缺笔、空白、乱涂、缩放平移、相近字、阈值边界；测试明确当前算法是模板覆盖匹配，不将相近字识别准确率虚报为已解决。
- [x] 实现Go评估函数 `Evaluate(strokes, template, policy) (EvaluationResult,error)`，纯函数不访问DB/网络；将现有JS阈值转为命名策略v1并保存到版本化配置。
- [x] 扩展新作答协议与回执表，保留旧optionIndex接口作为v1兼容分支；v2选择题按stable option ID判定，书写按strokes判定。
- [x] 先测事务：写入回执失败时attempt/计划计数/掌握更新均回滚；同key同笔迹返回原响应，同key不同笔迹409。
- [x] 保存写失败的事实，失败后允许重写；书写每题最多3次正式提交，最终未通过作为已作答结束，避免无限卡住。预览次数不受孩子计数限制。
- [x] 使用提示后通过记录hinted，练习可完成但不按无提示正确提升掌握；扩展学习输入表达独立掌握证据，保留其他学科原有默认语义并做回归。
- [x] 原子事务只进行已就绪本地数据评估与写入；领取时复制或缓存受校验的模板版本，作答事务内不下载对象或调用素材服务。
- [x] 给题目后台提供同一评估包的无记账预览入口，避免后台与App评分分叉。
- [x] 执行 shared-go、literacy-server、parent-dashboard 的Go测试；在PG验证真实幂等锁和SQLite迁移保留旧数据。

### Task 6：生成可配置的混合题包

**修改**：`task-admin/backend/internal/generation/literacy.go`（拆出choice/handwriting文件）、`internal/taskgen/materials.go`、`service.go`、HTTP handler及测试。

- [x] 先测“4看字选义＋4看义选字＋4听写”精确12题；只有1个字、没有义图时仍可生成1题听写。
- [x] 生成按注册题型分派，选择题继续4个选项；听写生成音频、模板引用、评估策略版本，禁止塞空选项冒充选择题。
- [x] 更新结构验证、替题、排序、发布媒体检查，使其按题型执行；模板JSON属于可校验资源，不套用图片MIME验证。
- [x] 指纹包含题意、素材及评估策略版本。相同字、音频、模板的听写重新排序不算新变式；无新组合明确返回，不虚构题量。
- [x] 新发布前确认App和学习服务支持所需契约；旧v1题包原样可读，不批量升级或重新生成120题。
- [x] 新快照遵守题干显示要求；看字选义优先素材字图，App不能忽略已冻结glyph改用系统字体。
- [x] 运行task-admin Go全量测试，重点回归v1生成、失败保留旧修订和复习来源不丢失。

### Task 7：三个端接入同一播放器，移除正式路径的示例替代

**修改**：题目后台 `GenerationPage.tsx`、素材后台 `QuizQuestionRow.tsx`/`quizPreview.ts`、App `HomePage.tsx`、`DemoPracticePage.tsx`、`PracticePage.tsx`、`TasksPage.tsx`、API与测试。

- [x] 题目后台的QuestionCard交给player，配比表按能力展示三种题型；题目来源、标准答案和参数仍放player外侧的折叠面板，不影响孩子画面。
- [x] 素材后台试做也使用player；先冻结当前选定素材再预览，更新后的预览显示新revision，旧任务预览保持旧revision。
- [x] 可选版本化任务页接入 player；首页直接练习保留 App 原组件及本地书写反馈，不提交学习记录。
- [x] 首页三张题型卡进入 `/practice/type/glyph|sense|write`，保留原有练习页面，App 独立组织题目；仅通过素材目录取得媒体，不查询或领取后台任务。
- [x] 固定示例只保留在 `/demo/:type`，旧 `/practice/type/:type` 映射到真实题型任务页。无已发布任务显示空状态，不自动生成或发布。
- [x] 通过第一次播放/用户手势处理音频策略；图片、音频或评估失败均可重试，笔迹与clientId保留。清空重写视为新提交内容。
- [ ] 前端行为测试覆盖三端相同题目渲染、预览不写账、触屏笔迹、混合题序、刷新恢复、发布失败不丢题。

### Task 8：复习策略与历史读取支持书写事实

**修改**：`task-admin/backend/internal/taskgen/review.go`、`worker.go`、review-evidence handler；parent-dashboard历史服务、diagnosis服务与相关测试。

- [x] 复习按response_kind分组：选择题读取实际错选，书写读取未通过/提示完成及评估结果，不再要求每条记录有selectedOptionId。
- [x] 原题重练直接引用旧听写快照；本轮未接描写模式，不能把它当可生成的补救题。
- [x] mixed模式如只能重练原听写，返回逐组容量缺口并建议original_only，不静默凑成“原题＋变式”。自动worker沿用这条规则，失败不影响孩子提交。
- [x] 后台复习依据显示原笔迹、标准字、提示状态与评估结论；不增加孩子整体薄弱点总览。
- [x] 进度历史保留书写题，知识库能看到书写失败事实；不得渲染成“选错了第0项”。
- [x] 测试选择＋书写混合历史、错后重写成功、提示完成、回执重放、重复复习去重及自动复习不递归。

### Task 9：防漂移检查、部署与最终验收

**创建**：`scripts/check-literacy-consistency.sh`、三端共享fixture的界面测试；**修改**：三端README、开发约定、验收文档及构建检查入口。

- [x] 一条检查命令包含契约生成无差异、共享player测试、三个前端测试/构建，以及Go生成/判题/回执回归。使用显式目录运行，不依赖个人软链接。
- [x] 静态检查正式player宿主不导入demoBanks、不自行合成媒体URL、不维护另一份题型常量；按允许的adapter路径检查，避免误拦合法测试fixture。
- [ ] 同一题目版本以同样1024×768练习视口在三端截图，比较内部画面；390×844检查窄屏，覆盖选择、听写、错误提示、音频失败和长提示语。
- [x] 隔离库生成12题混合任务，在后台逐题试做，在素材后台试做同一revision，确认图片/音频/模板摘要一致、提交同笔迹得到同评估结果。
- [x] 发布并由专用孩子完成；故意写错、提示后完成、网络中断后重放，验证各类回执与计数；从来源生成只重练原题的复习并完成。
- [x] 素材更新测试在独立对象前缀内进行；旧题、旧模板摘要不变，新任务用新素材；禁止覆盖生产共享素材作为测试手段。
- [x] 发布次序：备份→兼容数据库迁移→素材模板服务→学习解析/评估服务与App→题目后台启用新题型→素材试做入口。暂未就绪的能力保持禁用。
- [x] 正式服务原端口不变。验收孩子和作答不得写正式账本；保留现有120道题，另建混合题草稿供用户检查，未经发布动作不出现在孩子端。
- [x] 关闭新增能力即可停止新书写题发布，但保留新格式读取、模板及回执；已有书写计划继续可完成，不做删列降级。
- [x] 记录实际task/revision/plan/receipt、截图路径、测试退出码、兼容限制，再勾选完成。

## 7. 后续维护约定

每增加一种题型，必须同一变更中覆盖：契约→素材能力→生成器→共享player→服务端评估→回执与复习→三端验收。缺任一环节只显示“未就绪”，不能作为已支持题型发布。

App视觉调整只修改共享player及其基准，两个后台升级同一包；素材修改只产生新素材版本，不直接改历史题；评估规则调整产生新policyVersion，不重判既有回执。

## 8. 完成定义

- [x] App样式成为唯一练习样式，后台没有独立维护的识字答题界面。
- [x] 三端同revision引用同一素材，正式题无emoji/示例图/动态第三方模板替代。
- [x] 看字选义、看义选字、听音写字均能素材检查、生成、试做、发布、领取、判定、回执、复习。
- [x] 后台试做不写学习账；提示完成与独立完成可区分。
- [x] 原120题及已有学习历史兼容，合同漂移检查和三端构建通过。

仅新增“写一写”菜单、仅让后台画出田字格、或只共享接口而保留三份样式，都不算完成本计划。

## 2026-09-12 执行结果

核心功能与正式部署已完成，详见 `docs/verification/literacy-three-surface-unification.md`。未勾选项是尚未建立的更完整保障：改造前视觉基线、所有错误态/同题逐像素比对、相近字与阈值完整基准。当前已有三端两尺寸主要场景截图、同版本媒体摘要校验、真实PG作答复习与全量回归；不将这些替代完整基准后误标完成。

书写策略初版额外加入密集/稀疏乱画约束，属于首次部署的ink-match-v1；未来变更另起版本。提示协议已通，App暂无标准字提示按钮。旧非版本化听写因缺冻结模板显示新版任务入口，保留旧记录。
