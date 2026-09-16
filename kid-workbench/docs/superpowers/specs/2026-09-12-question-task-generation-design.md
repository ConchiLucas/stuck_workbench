# 题目后台：任务出题与复习出题设计

日期：2026-09-12  
状态：依据用户明确定位和自主决策授权制定，作为后续实现基线。  
实施计划：[详细开发计划](../plans/2026-09-12-question-task-generation.md)

## 1. 产品定义

题目后台以任务为单位，使用素材后台的内容生成若干题目，支持普通练习任务和复习任务。任务是出题要求及其产物，不是孩子的学习进度记录。

界面名称使用「题目后台」，目录保留 `task-admin/`，端口保留 `19201`（后端与 Docker 页面）和 `19202`（Vite）。API 继续使用 `question-tasks`，避免无收益的改名迁移。

| 系统 | 所有权 |
| --- | --- |
| content-admin | 知识内容、图片、音频、素材就绪检查和不可变素材修订 |
| task-admin | 出题要求、题型模板、生成题目、校验、题包发布、复习出题策略 |
| literacy-server 等学科服务 | 孩子领取题包、创建学习计划、判题、记录作答 |
| shared-go | 学习事务、掌握度和统计规则 |
| parent-dashboard | 按时间查看学习成果与历史作答 |
| diagnosis-admin | 知识积累、薄弱与错题的阅读和分析 |

题目后台读取必要的作答事实作为输入，不写掌握度，不做孩子学习总览、正确率趋势或错题分析看板。进度和知识库仅增加读取新题目格式的兼容代码，不改变产品分工。

## 2. 实施范围

首个完整版本必须跑通识字「素材 → 普通任务 → 发布 → 孩子作答 → 手动生成复习任务 → 再次作答」。支持 `glyph_sense` 看字选义和 `sense_char` 看义选字。独立题型模板注册表允许后续增加学科，但界面只展示真正接通的学科与题型。

第一版不增加大模型出题、不生成新图片和新音频、不改手写评分、不迁移全部学科、不重做三个其他后台。自动复习列为第一版之后的独立增量，默认关闭，完成手动流程再实现。

## 3. 从现状迁移的关键事实

- 当前后台仅从 `questions` 固定抽 10 题，`question_task_items` 保存题目引用。
- `questions` 对 `(kp_id, code)` 有唯一索引，不能承载同一知识点、同一题型的多道变式题。
- 素材地址带更新时间只解决缓存，原图片和音频仍可能被覆盖。
- `plan_items` 已有题目快照，但旧 `question_id` 非空，部分读取仍依赖 `INNER JOIN questions`。
- `attempts` 不保存所选选项；`plan_items.picks` 仅保存显示下标。
- `shared-go/learning.ApplyOne` 支持空 QuestionID 与显式 SkillCode，可直接复用。

因此新增独立题目版本、真实冻结媒体、学习计划来源关联和原子作答回执，保留旧题库和旧学习流程。

## 4. 出题要求

普通任务输入使用：`title`、`subjectCode`、`kind=practice`、`scope.moduleCodes`、`scope.kpIds`、`targetCount`、`typeCounts`、`distractorScope`。

- `targetCount` 默认 10，范围 1–20，与当前识字孩子端题量上限一致。
- `typeCounts` 必须为正整数且合计等于题量；默认两种题型均分，奇数多出的 1 题给 `glyph_sense`。
- 第一版每个任务限制一个识字组；可以从组内进一步选择目标字。
- 同一目标字可考两种题型；普通任务同一个 `(kpId, questionType)` 最多一题。目标字覆盖均衡，先轮流覆盖再分配第二种题型。
- 干扰项默认取同组；显式选择 `subject` 才允许扩展到识字学科其他可用素材，不静默扩大范围。
- 固定四选一，候选不足时阻止生成并说明缺口，不自动减题或重复凑数。
- 随机种子由服务端保存；同样的素材修订、模板版本、请求和种子得到相同题目。
- 不提供没有实际算法支撑的难度滑杆。第一版难度由四选一模板固定，后续有经过验证的干扰项规则才扩展。

示例（ID 仅用于说明，测试用 fixture 提供对应数据）：

```json
{
  "title": "第1组识字练习",
  "subjectCode": "literacy",
  "kind": "practice",
  "scope": {"moduleCodes": ["g1"], "kpIds": [1, 2, 3, 4]},
  "targetCount": 8,
  "typeCounts": {"glyph_sense": 4, "sense_char": 4},
  "distractorScope": "module"
}
```

## 5. 素材接口与不可变媒体

题目后台通过 content-admin 的专用 HTTP 契约取得素材；不继续在出题逻辑里 JOIN 素材后台私有表。共用数据库不等于共用业务所有权。

content-admin 新增：

- `GET /api/v1/generation-materials/literacy?moduleCode=g1`：素材候选、每种题型可用性、缺项原因、来源修订标识。
- `POST /api/v1/generation-materials/literacy/freeze`：传入候选 `kpIds` 及各自的来源修订，冻结内容及实际媒体，返回不可变 material revision。
- `GET /api/v1/material-revisions/:revisionId/media/:kind`：读取该修订的媒体，kind 仅允许 `glyph`、`sense`、`speech`。

冻结由素材后台读出实际字图、义图、语音字节，分别计算 SHA-256，写入 `material-revisions/sha256/<hash>.<ext>`。摘要相同复用对象，禁止覆盖已存在但摘要不同的对象。数据库保存内容摘要、对象 key、类型和字节数。旧素材继续用原地址维护，不破坏现有 App。

来源修订用规范化内容字段摘要表示。冻结期间读取前后检查来源修订一致；任一不同返回 409，要求刷新素材。冻结读和原素材覆盖写使用同一知识点的跨进程锁，多个知识点按 kpId 升序加锁；图片/音频生成的外部调用放在锁外，只将最终对象写入和元数据更新放在锁内，避免读到正在替换的一半资源。冻结产物以实际字节摘要为准，数据库写入失败留下的未引用对象不删除已引用版本，可另做保留期清理。

识字当前没有完整审核状态：第一版明确返回 `readiness`，不虚构 `published` 标志。看字选义要求目标字可显示、四个选项义图可用；看义选字要求目标题干义图可用、四个字形可显示。字形允许用已有汉字文本渲染，音频在本版模板中必需。所有选项文字或图片摘要不得重复。发布前需要人工预览确认语义；自动检查不宣称能识别所有多义字和图义歧义。

后台及孩子端通过各自同源媒体代理访问素材修订，代理只接受修订 ID 和枚举 kind，不接受任意 URL。数据库不保存临时签名 URL 或容器内地址。服务之间的 base URL 通过环境配置传递。

## 6. 数据模型

| 实体 | 字段与约束 |
| --- | --- |
| `material_revisions` | 素材后台所有；revision ID、subject、kpId、规范化内容、摘要、媒体对象信息、创建时间；摘要唯一 |
| `question_tasks` | 扩展 kind、spec JSON、target_child_id（复习用）、parent_task_id、source_mode、active_revision_id、published_revision_id、row_version；原 ID 保留 |
| `question_task_revisions` | task_id、revision_no、spec、seed、template_version、material_manifest、validation、state、created_at；唯一 task_id + revision_no |
| `question_versions` | revision_id、seq、kp_id、question_type、skill_code、fingerprint、snapshot JSON、source_question_version_id（变式用）；写入后不可修改 |
| `question_task_generation_runs` | operation、请求幂等键、request_hash、state、expected_row_version、结果 revision_id、结构化错误；用于失败展示和重复请求恢复 |
| `question_task_review_sources` | task_id、source_receipt_id、source_question_version_id、reason；保存复习出题依据 |
| `study_plans` | 新增 source_question_task_id、source_question_task_revision_id、task_claim_key；学习状态仍由学科服务所有 |
| `plan_items` | 新增 question_version_id；旧 question_id 允许 NULL，至少一种来源非空；已有快照列保存兼容内容 |
| `question_attempt_receipts` | attempt_id 唯一、child_id + client_id 唯一、plan_id、plan_item_id、question_version_id、kp_id、skill_code、显示下标、原始下标、稳定选项 ID、正确性、耗时、答题响应 JSON、发生时间 |

生成题不写进旧 `questions`，不修改其唯一索引。旧题包继续走 `legacy_pool` 读取；首次修改时复制为新任务并标记「从旧题包导入」。旧题包没有冻结版本前不能作为新孩子端任务发布入口，仍可在后台查看。历史内容无法恢复的部分明确标记「按迁移时内容保存」，不能伪称当时快照。

数据库迁移按所有权拆分：素材表由 content-admin 管；出题表由 task-admin 管；学习计划列及回执由现有 parent-dashboard 迁移目录管。共享迁移不依赖另一个服务新建表的外键，跨服务 ID 由领取事务验证。迁移是加法式升级，不删除旧任务和作答。

## 7. 题目快照契约

```ts
type MediaRef = { revisionId: string; kind: 'glyph' | 'sense' | 'speech'; sha256: string }
type QuestionOption = {
  id: string; kpId: number; text: string; image?: MediaRef; audio: MediaRef
}
type QuestionSnapshot = {
  schemaVersion: 1
  subjectCode: 'literacy'
  kpId: number
  targetText: string
  questionType: 'glyph_sense' | 'sense_char'
  skillCode: 'glyph_sense' | 'sense_char'
  templateVersion: 'literacy-choice-v1'
  prompt: string
  stem: {text?: string; image?: MediaRef; audio?: MediaRef}
  options: QuestionOption[]
  answerOptionId: string
  explanation: string
  materialRevisionIds: string[]
}
```

选项 ID 用 `kp:<id>`，题目内唯一。题目版本 ID 独立于旧题库 ID。后台 DTO 包含答案用于试答；孩子领取时生成独立 DTO，移除答案和带答案的解释，服务端根据快照判题。显示乱序保存在 plan_items.option_order，刷新、重试和结果页都使用同一顺序。

指纹包括目标知识点、题型、题干及素材版本和按选项 ID 排序的选项内容，不包括显示顺序。仅打乱选项不算新变式。复习任务允许同一知识点同一题型多题，但指纹仍必须不同；原题重练可保留其原始指纹一次。

## 8. 任务生命周期

- 创建任务 → 无题草稿；主动生成 → 完整可预览修订；发布 → 固定 published revision。
- 生成、整包重生成、替换单题都创建新修订，成功后原子切换 active revision，失败保留原修订。
- 第一版仅草稿允许编辑；已发布修改先撤回。撤回只阻止新领取，已领取计划仍能完成。
- 已发布修订和被学习计划引用的题目版本不得覆盖或硬删除。删除仅适用于从未发布且未被领取的草稿；其他任务使用归档。
- 重生成和替题必须确实改变指纹；没有可用新组合则提示「没有新的可用题目组合」。
- 修改携带 expectedRowVersion；并发冲突返回 409，禁止最后一次写入静默覆盖。
- 生成先记录 run，外部素材调用不占长数据库事务；提交结果前锁任务并核对版本。进程中断的 running run 在启动恢复时标记 failed，允许新幂等键重试，不保留永久转圈状态。

## 9. 孩子端消费与作答

在识字首页保留原题型卡，增加「练习任务」入口。列表只展示已发布普通任务及分配给当前孩子的复习任务，展示标题、题数和继续/开始按钮，不展示后台术语。

领取通过 literacy-server，在同库事务中锁孩子和任务，检查发布及目标孩子，复制修订快照到学习计划，保存来源 ID；任务状态检查和计划创建在同一事务内。相同 child + claimKey 返回原计划，key 对应不同题包返回 409。同一个修订有未完成计划则优先继续；已完成后明确点「再练一次」才创建新 claimKey。

生成题判题走独立分支：严格使用 question_snapshot，不回退最新题库或 kpId 媒体。用显示下标映射稳定 option ID，再与答案比较。调用 ApplyOne 时 QuestionID=nil，SkillCode 来自快照；同一个事务保存学习更新与回执，任一步失败全部回滚。

重复 clientId 必须匹配同一个 planItem 和选项；匹配则返回已存响应，不重复计数；不匹配返回 409。保留现有每题最多两次尝试规则。回执记录每一次错误，即使第二次答对，也能作为复习依据。

## 10. 复习出题

第一版入口位于普通题包详情的「生成复习任务」，选择该题包的一次已完成练习记录。这里只提供识别出题来源必需的信息（计划编号、时间和可用错题数），不铺孩子分析页。后端限定该孩子、该来源计划，使用新回执中的真实错误。

默认策略 `wrong-answer-v1`：

1. 将错误按 `(kpId, questionType)` 合并，保留全部来源回执；排序为错误次数降序、最近错误优先、kpId 升序。
2. 默认目标题量为 `min(10, 不同错误组数 × 2)`；用户可改成 1–20。
3. 每组先保留一道原题，再尝试同目标同题型更换干扰项；变式优先保留孩子实际错选项作为干扰项。
4. 原题来自完整历史版本，变式使用当前可用素材的新修订，分别记录来源。不通过交换选项顺序冒充变式。
5. 没有新干扰项时返回实际可生成题量和原因；用户可明确选择「只重练原题」后生成，不自动降级或跨题型。
6. 无错题返回「这次练习没有可用于复习的错题」，不创建空任务。
7. 来源请求的规范化回执集合、孩子、策略版本和出题参数共同形成去重键；重复提交返回已有任务。新一轮练习产生新回执后可以生成新任务。

已有旧作答只在能恢复精确题目和选项时导入；缺失证据时不推断具体错选项。第一版验收以新流程产生的回执为准。

## 11. 自动复习增量

手动闭环完成后增加 `APP_AUTO_REVIEW_ENABLED=false` 默认关闭的后台 worker。开启后每 60 秒扫描已完成的普通任务来源学习计划，依据同一策略创建复习草稿，不自动发布。

持久化 job，唯一键 child_id + source_plan_id + policy_version；重启继续，每轮总共最多执行 3 次，前两次失败分别等待 1 分钟和 5 分钟，第 3 次失败后等待手动重试。任务详情展示生成错误。只处理 worker 启用时间之后完成的普通任务，不追扫历史，不从复习任务递归触发，不影响作答接口响应。没有错题的计划标记 skipped。worker 不负责决定掌握度或复习日历。

## 12. 验收原则

- 后台中所有新增主操作围绕出题任务，页面不加入学习总览。
- 指定数量与题型配比精确满足；素材不足、语义待人工检查、媒体失效都有明确状态。
- 素材更新后历史题目和已领取计划的文字、图片、音频、答案均不变。
- 题包和学习计划 ID 可追溯，选项乱序不影响判题或错选记录。
- 作答幂等、复习生成幂等；另一个孩子不能领取定向复习任务。
- 新旧题包共存，旧学习流程继续通过回归测试。
- 本设计的人工发布只是产品操作，不要求开发过程中逐项找用户审批实现细节。
