# 儿童学习工作台

进度后台 + 孩子答题端 + 素材后台，共用一套 PostgreSQL。

## Docker 启动（推荐）

在 `kid-workbench` 目录：

```bash
./scripts/docker-up.sh
```

或 `make up`（同样走这个脚本）。首次会构建镜像，之后可用 `./scripts/docker-up.sh --no-build` 只启动。停止：`./scripts/docker-up.sh --down`。

| 服务 | 地址 |
|------|------|
| 进度后台 + API | http://localhost:19081 |
| 识字 API | http://localhost:19151 |
| 识字孩子端 | http://localhost:19152 |
| 拼音 API | http://localhost:19111 |
| 拼音孩子端 | http://localhost:19112 |
| 科普 API | http://localhost:19121 |
| 科普孩子端 | http://localhost:19122 |
| 算数 API | http://localhost:19141 |
| 算数孩子端 | http://localhost:19142 |
| 古诗 API | http://localhost:19161 |
| 古诗孩子端 | http://localhost:19162 |
| 英语短句 API | http://localhost:19171 |
| 英语短句孩子端 | http://localhost:19172 |
| 成语 API | http://localhost:19181 |
| 成语孩子端 | http://localhost:19182 |
| 逻辑 API | http://localhost:19191 |
| 逻辑孩子端 | http://localhost:19192 |
| 素材后台 | http://localhost:19091 |
| 题目后台 | http://localhost:19201 |
| 诊断台 | http://localhost:19211 |
| PostgreSQL | localhost:15432 |

首次如需 60 天演示数据：

```bash
make seed-demo
```

其他命令：

```bash
make down    # 停止
make logs    # 查看日志
make seed    # 重新灌 catalog + 题库（幂等；短句 scene/reply 变更后必须再跑）
```

素材后台还需 Shared Config Center，以及 Docker 网络：

```bash
docker network create vibedeploy-shared   # 若尚无
```

## 本地开发（可选）

需本机 PostgreSQL 监听 `15432`，或使用 Docker 只起数据库：

```bash
docker compose up postgres -d
```

```bash
# 后端
cd parent-dashboard/backend
go run ./cmd/seed -mode=catalog
go run ./cmd/seed -mode=questions
make run                    # http://localhost:19081

# 进度后台前端（另开终端）
cd parent-dashboard/frontend
npm install && npm run dev    # http://localhost:19083

# 识字服务与孩子端（另开两个终端）
cd literacy-server
go run ./cmd/server            # http://localhost:19151
cd ../literacy-app
npm install && npm run dev     # http://localhost:19152

# 拼音服务与孩子端（另开两个终端）
cd pinyin-server
go run ./cmd/server            # http://localhost:19111
cd ../pinyin-app
npm install && npm run dev     # http://localhost:19112

# 科普服务与孩子端（另开两个终端）
cd ../science-server
go run ./cmd/server            # http://localhost:19121
cd ../science-app
npm install && npm run dev     # http://localhost:19122

# 算数服务与孩子端（另开两个终端）
cd ../math-server
go run ./cmd/server            # http://localhost:19141
cd ../math-app
npm install && npm run dev     # http://localhost:19142

# 古诗服务与孩子端（另开两个终端）
cd ../poem-server
go run ./cmd/server            # http://localhost:19161
cd ../poem-app
npm install && npm run dev     # http://localhost:19162

# 英语短句服务与孩子端（另开两个终端）
cd ../phrase-server
go run ./cmd/server            # http://localhost:19171
cd ../phrase-app
npm install && npm run dev     # http://localhost:19172

# 成语服务与孩子端（另开两个终端）
cd ../chengyu-server
go run ./cmd/server            # http://localhost:19181
cd ../chengyu-app
npm install && npm run dev     # http://localhost:19182

# 逻辑服务与孩子端（另开两个终端）
cd ../logic-server
go run ./cmd/server            # http://localhost:19191
cd ../logic-app
npm install && npm run dev     # http://localhost:19192

# 素材后台（另开终端）
cd content-admin/frontend
npm install && npm run dev    # http://localhost:19092（代理 API → 19091）

# 题目后台（另开两个终端）
cd task-admin/backend
go run ./cmd/server            # http://localhost:19201
cd ../frontend
npm install && npm run dev     # http://localhost:19202（代理 API → 19201）

# 诊断台（另开两个终端）
cd diagnosis-admin/backend
go run ./cmd/server            # http://localhost:19211
cd ../frontend
npm install && npm run dev     # http://localhost:19212（代理 API → 19211）
```

## 目录

```
kid-workbench/
├── docker-compose.yml
├── parent-dashboard/
│   ├── backend/     # Go API（Docker 内嵌进度后台静态资源）
│   └── frontend/    # 进度后台 React 源码
├── literacy-server/ # 识字孩子端专用 API（直读 PostgreSQL / MinIO）
├── literacy-app/    # iPad 横屏识字学习 PWA
├── pinyin-server/   # 拼音孩子端专用 API（直读 PostgreSQL / MinIO）
├── pinyin-app/      # iPad 横屏拼音学习 PWA
├── science-server/  # 只读已发布科普内容、记录科普学习的独立 API
├── science-app/     # iPad 横屏科普探索 PWA
├── math-server/     # 算数孩子端独立 API（直读 PostgreSQL / MinIO）
├── math-app/        # iPad 横屏算数练习 PWA
├── poem-server/     # 小儿诗园古诗线索 API
├── poem-app/        # iPad 横屏古诗练习 PWA
├── phrase-server/   # 英语短句孩子端独立 API（直读 PostgreSQL，无 MinIO）
├── phrase-app/      # iPad 横屏英语短句练习 PWA
├── chengyu-server/  # 成语孩子端独立 API（直读 PostgreSQL，无 MinIO）
├── chengyu-app/     # iPad 横屏成语练习 PWA
├── logic-server/    # 逻辑孩子端出题 API（直读 PostgreSQL pick1）
├── logic-app/       # 逻辑题型试玩 PWA（找规律/分类/排序/图形推理/找不同/比较）
├── content-admin/   # 素材后台
├── task-admin/      # 题目后台（识字题包组卷 / 发布）
├── diagnosis-admin/ # 诊断台（只读掌握度 / 错因）
└── shared-go/       # 跨学科共享的掌握度与原子学习写入规则
```

`shared-go` 只保存各学科共同成立的学习模型、掌握度状态机和原子学习写入事务；HTTP 路由、页面模型、素材生成和学科题目规则仍留在各自项目。

算数学习基础使用数据库迁移 `012_math_plan_snapshot.sql`：`study_plans` 保存计划类型、模块和阶段，`plan_items.question_snapshot` 保存不可变的完整题目快照。算数掌握度按模块选择技能：加减法为 `calc + story`，图形为 `find + name`；共享学习事务负责幂等作答、技能汇总、每日统计和掌握奖励。题目级 TTS 由素材后台预生成到 `math/questions/<questionId>.mp3`，孩子端服务只读 PostgreSQL 和 MinIO。

详细功能说明见 [parent-dashboard/README.md](parent-dashboard/README.md) 与 [content-admin/README.md](content-admin/README.md)。
