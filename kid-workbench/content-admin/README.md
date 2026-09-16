# 素材后台

算术详情管理支持十二种详情的文案、结构化例题、草稿保存、音频生成与发布，预览使用 `packages/math-player`。发布接口为 `/api/v1/math/details/published`；已发布修订与音频摘要地址不可变，更新文案会清除过期音频引用。2026-09-12 部署时 TTS 上游无可用账号，新题干音频尚待补齐，详见仓库根 `docs/verification/2026-09-12-math-content-unification.md`。

素材后台（位于 `kid-workbench/content-admin`）。当前能力：

1. **识字**：从同库 `study_workbench` 同步识字表，自动判定是否需要义图（可人工改）
2. **科普**：维护儿童端摘要、解释、趣闻，并执行审核与发布
3. **配置管理**：只读消费 Shared Config Center

## 前提

- Shared Config Center 已启动（配置菜单）
- `kid-workbench` Postgres 已启动（识字同步），本机常见端口 `15432`
- Docker 网络：`docker network create vibedeploy-shared`（若尚无）

## Docker 启动

推荐在 `kid-workbench` 根目录统一启动：

```bash
cd .. && make up
```

或仅启动本服务：

```bash
make up
```

打开 http://localhost:19091 （默认进入识字）

根目录 compose 下，素材后台通过 Docker 网络直连 `postgres` 服务。

## 本地开发

```bash
cd backend
SHARED_CONFIG_CENTER_BASE_URL=http://127.0.0.1:18783 \
  APP_DB_HOST=127.0.0.1 APP_DB_PORT=15432 \
  go run ./cmd/server

cd frontend
npm install && npm run dev   # http://localhost:19092
```

## 识字 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/literacy/sync` | 从题库同步 |
| GET | `/api/v1/literacy/chars` | `view=groups\|table`，可选 `needsSenseImage` |
| PATCH | `/api/v1/literacy/chars/:kpId` | `{ "needsSenseImageOverride": true\|false\|null }` |

## 科普发布流程

1. 点“从题库同步”。首次同步的新知识点进入“待审核”。
2. 编辑摘要、解释和趣闻并保存；已发布内容一旦修改也会退回“待审核”。
3. 点“提交审核”，确认儿童可读性和事实准确性。
4. 点“发布”。发布前必须已经生成该知识点的 `recognize` 题目。

为兼容已有线上内容，数据库升级时已有 `science_assets` 行默认标为已发布；之后新增内容一律从草稿开始。可用只读脚本 `backend/scripts/verify_science_publication.sql` 核对各状态数量与无题目的已发布内容。

## 算数题目音频

算数有两类音频，所有权和用途不同：

- `math_assets.speech_audio_url` 是知识点级后台预览音频，继续由现有“读音”按钮维护。
- `questions.media_url` 是孩子端练习必须使用的题目变体音频 object key，格式为 `math/questions/<questionId>.mp3`。

在算数页面按模块查看“题目音频 ready/total”，并在发布练习内容前执行“生成本组题目音频”。空值、旧的 HTTP URL 或非标准 object key 都会重新生成；孩子端不使用浏览器 TTS 兜底。

## 设计文档

- `docs/superpowers/specs/2026-08-29-study-content-admin-config-design.md`
- `docs/superpowers/specs/2026-08-29-literacy-assets-list-design.md`

## 题目生成素材接口

新增 `/api/v1/generation-materials/literacy` 提供候选就绪信息，POST `/api/v1/generation-materials/literacy/freeze` 冻结指定来源修订。媒体保存为内容摘要寻址的不可变对象，题目后台和孩子端通过修订媒体接口使用历史图片、音频。素材后台继续负责素材制作，不管理孩子学习进度。


## 识字三端一致性（2026-09-12）

正式练习和后台试做共用 `packages/literacy-player`，练习区域沿用识字 App 样式；后台管理外壳独立。图片、音频、书写模板都引用素材后台冻结版本。支持看字选义、看义选字、听音写字，描写尚未接入。

题型唯一来源是 `contracts/literacy/question-types.json`，修改后在仓库根执行 `node scripts/generate-literacy-contracts.mjs`。新增题型需同时补齐素材、生成、播放器、服务端评估、回执和复习；发布检查实际部署的 App 和学习服务能力。

从 kid-workbench 根目录运行 `bash scripts/check-literacy-consistency.sh` 检查契约漂移、正式宿主、共享播放器、三个前端与六个 Go 项目。需先在三个前端运行 `npm ci`；Go HTTP 测试需要允许本地监听。

Docker 构建上下文为 kid-workbench 根目录，使用根 `docker compose build content-admin task-admin literacy-server literacy-app`，独立 compose 也已引用父目录。共享包无需单独运行服务。

听写由服务端 `ink-match-v1` 模板匹配评分，记录真实笔迹、提示状态和评估版本；不是 OCR 或专业笔顺评分。提示完成不会作为独立掌握证据。后台试做不写学习账本。

部署先备份并通过 parent-dashboard 执行兼容迁移 014，再升级素材、学习服务和 App，最后升级题目后台。保留历史 v1 题目及回执，不通过删列回滚。验收详情见 `docs/verification/literacy-three-surface-unification.md`（仓库根）。

## 2026-09-13 素材界面职责调整

按用户最新要求，素材后台已移除题型入口、题型能力标签、详情编辑及试做预览。保留内容、图片、音频、书写模板和素材审核发布管理。供其他服务使用的素材数据与接口继续兼容；旧文档中的素材试做说明不再代表当前页面。执行记录见 `../docs/menu-alignment/素材端清理记录.md`。
