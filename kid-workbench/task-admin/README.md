# 题目后台

算术入口 `/math` 从素材后台读取已发布详情，按 5/10/20 范围、题型与数量生成任务，并冻结来源修订和题面快照。预览共用 `packages/math-player`，后台试做不写孩子账本。本轮尚未提供算术 App 领取题包功能；App 详情示例直接来自素材发布目录。

访问 [题目后台](http://localhost:19201)。本服务按任务生成题目，消费素材后台的内容，依据真实错选生成复习题包。学习进度、掌握度和孩子知识账本仍由原学习服务、进度后台与孩子知识库负责。

## 当前能力

- 支持识字「看字选义 / 看义选字 / 听音写字」，指定识字组、目标字、1–20 题、题型配比和干扰项范围。
- 创建草稿后生成题目，试答、换题、排序、查看历史修订，检查后发布；撤回和归档阻止新领取，已有练习保持可读。
- 图片和音频从素材后台冻结为不可变修订。原素材后续更新不会改变旧题包。
- 孩子从识字端「练习任务」领取，支持继续和再练；真实作答才更新学习账本。
- 从一次已完成练习的实际错选生成复习草稿，可选「原题 + 变式」或「只重练原题」。重复请求复用同一复习任务。
- 旧抽题题包标记「旧题包」，原预览保留；按当前素材导入时另建草稿，记录旧题 ID，不伪造历史媒体。

主数据使用现有 PostgreSQL `study_workbench`。新题保存在 `question_versions`，不占用旧 `questions` 的知识点与题型唯一键。

## 部署与本地开发

根目录 Docker Compose 配置已包含素材服务地址。首次部署顺序为：备份数据库 → parent-dashboard 运行学习迁移 013 → content-admin 素材迁移 → task-admin → literacy-server、孩子前端与历史读取服务。不要运行 seed 代替迁移，以免意外重做课程种子。

```bash
# 在本目录 backend 中；使用既有数据库配置
APP_CONTENT_ADMIN_URL=http://127.0.0.1:19091 go run ./cmd/server
# 在本目录 frontend 中
npm ci
npm run dev
```

后台端口 19201；开发前端 19202。`APP_CONTENT_ADMIN_URL` Docker 默认配置为 `http://content-admin:19091`。新题媒体走同源代理。

## 自动复习

`APP_AUTO_REVIEW_ENABLED=false` 默认关闭。启用后每 60 秒检查首次启用之后完成的普通任务，只生成复习草稿。任务与租约持久化，失败按 1 分钟、5 分钟退避，总计 3 次后停止，可在来源任务详情查看错误并手动重试。不会自动发布，不会对复习任务递归生成，也不会重放所有历史学习。

关闭开关停止新扫描；保留题目修订、媒体、作答回执及新数据库列。已领取的新题计划仍需要新版学习服务，不能直接回退到只识别旧 question_id 的服务。

## 验证与实现记录

后端 `GOCACHE=/tmp/kid-workbench-go-cache go test ./...`；前端 `npm test` 与 `npm run build`。

详细设计、分阶段计划和实际验收记录见：

- [设计](../docs/superpowers/specs/2026-09-12-question-task-generation-design.md)
- [开发计划](../docs/superpowers/plans/2026-09-12-question-task-generation.md)
- [验收记录](../docs/verification/2026-09-12-question-task-generation.md)

其他学科、书写评分、开放题与复习日历不在本次首版范围内。


## 识字三端一致性（2026-09-12）

正式练习和后台试做共用 `packages/literacy-player`，练习区域沿用识字 App 样式；后台管理外壳独立。图片、音频、书写模板都引用素材后台冻结版本。支持看字选义、看义选字、听音写字，描写尚未接入。

题型唯一来源是 `contracts/literacy/question-types.json`，修改后在仓库根执行 `node scripts/generate-literacy-contracts.mjs`。新增题型需同时补齐素材、生成、播放器、服务端评估、回执和复习；发布检查实际部署的 App 和学习服务能力。

从 kid-workbench 根目录运行 `bash scripts/check-literacy-consistency.sh` 检查契约漂移、正式宿主、共享播放器、三个前端与六个 Go 项目。需先在三个前端运行 `npm ci`；Go HTTP 测试需要允许本地监听。

Docker 构建上下文为 kid-workbench 根目录，使用根 `docker compose build content-admin task-admin literacy-server literacy-app`，独立 compose 也已引用父目录。共享包无需单独运行服务。

听写由服务端 `ink-match-v1` 模板匹配评分，记录真实笔迹、提示状态和评估版本；不是 OCR 或专业笔顺评分。提示完成不会作为独立掌握证据。后台试做不写学习账本。

部署先备份并通过 parent-dashboard 执行兼容迁移 014，再升级素材、学习服务和 App，最后升级题目后台。保留历史 v1 题目及回执，不通过删列回滚。验收详情见 `docs/verification/literacy-three-surface-unification.md`（仓库根）。

可用 `APP_WRITING_ENABLED=false` 暂停发布新的听写题。该开关只作用于题目后台发布能力，不停止已有题目读取、孩子作答、回执或历史查询；选择题发布不受影响。默认开启。
