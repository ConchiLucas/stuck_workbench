# 识字孩子端

iPad 横屏优先的识字学习 PWA，只访问同源 `/api/v1`，由 Nginx 转发到 `literacy-server:19151`。

```bash
npm install
npm run dev   # http://localhost:19152
```

默认孩子编号为 `1`。素材由素材后台维护，孩子端不调用素材后台或进度后台 API。

## 练习任务

首页新增「练习任务」，可领取题目后台已发布的普通练习与定向复习，支持继续与再练。新题严格使用领取时的完整快照和冻结媒体，选项顺序刷新后保持一致；答题仍交由 literacy-server 判定和记账。原识字卡、独立练习与手写入口保留。

### 隔离验收默认孩子

开发验收时可以显式配置独立测试孩子和后端，不需要修改浏览器存储或增加孩子切换界面：

```sh
VITE_CHILD_ID=2 VITE_API_PROXY=http://127.0.0.1:29151 npm run dev -- --host 127.0.0.1 --port 29152 --strictPort
```

`localStorage` 中已有的 `literacy-child-id` 优先于 `VITE_CHILD_ID`；两者都未设置时仍默认孩子 `1`。验收须使用独立数据库里的测试孩子，不能把这项配置指向真实孩子进行测试作答。


## 识字三端一致性（2026-09-12）

正式练习和后台试做共用 `packages/literacy-player`，练习区域沿用识字 App 样式；后台管理外壳独立。图片、音频、书写模板都引用素材后台冻结版本。支持看字选义、看义选字、听音写字，描写尚未接入。

题型唯一来源是 `contracts/literacy/question-types.json`，修改后在仓库根执行 `node scripts/generate-literacy-contracts.mjs`。新增题型需同时补齐素材、生成、播放器、服务端评估、回执和复习；发布检查实际部署的 App 和学习服务能力。

从 kid-workbench 根目录运行 `bash scripts/check-literacy-consistency.sh` 检查契约漂移、正式宿主、共享播放器、三个前端与六个 Go 项目。需先在三个前端运行 `npm ci`；Go HTTP 测试需要允许本地监听。

Docker 构建上下文为 kid-workbench 根目录，使用根 `docker compose build content-admin task-admin literacy-server literacy-app`，独立 compose 也已引用父目录。共享包无需单独运行服务。

听写由服务端 `ink-match-v1` 模板匹配评分，记录真实笔迹、提示状态和评估版本；不是 OCR 或专业笔顺评分。提示完成不会作为独立掌握证据。后台试做不写学习账本。

部署先备份并通过 parent-dashboard 执行兼容迁移 014，再升级素材、学习服务和 App，最后升级题目后台。保留历史 v1 题目及回执，不通过删列回滚。验收详情见 `docs/verification/literacy-three-surface-unification.md`（仓库根）。

### 旧听写兼容限制（2026-09-12）

未绑定冻结题目版本的旧 `write_char` 计划项缺少可追溯的书写模板，不能继续使用旧版前端自报正确的判题方式。App 对仍待作答的这类题目显示说明及新版听写任务入口，不展示可提交写字板，不自动跳过、完成或改写旧计划；旧选择题和已完成的学习记录保留。新版听写使用冻结模板与服务端评估。

当前练习界面没有提示按钮，`hintsUsed` 为 0；服务端与复习依据能够读取带提示的作答协议，但不能将其描述为 App 已提供提示交互。

### 2026-09-12 范围修正：先对齐题型效果

App 首页恢复直接进入各题型练习，保留原有布局与翻题、听写流程。选择题图片通过素材目录获取；不依赖题目后台发布、查询或领取任务。后台同步能力暂存为可选能力，当前优先逐题型对齐效果。原 App 本地听写与后台评分仍需比对。App 26 项测试与构建通过。
