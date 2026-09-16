# Task Plan: 单词乐园题型练习闭环

## Goal
把 `english-app` 做成和其他四门课孩子端一样可玩：五张题型卡都能练完，听音选词和看图选词打到英语词库。

## Current Phase
Phase 5

## Phases

### Phase 1: 设计与计划
- [x] 对照四门课孩子端锁定默认
- [x] 写入设计说明
- [x] 写入实施计划
- **Status:** complete

### Phase 2: 本地 demo 立刻对错
- [x] 选择题点完立刻显示对错
- [x] 错一次可再点
- [x] 前端测试通过
- **Status:** complete

### Phase 3: english-server 实时出题
- [x] listen / look 生成器
- [x] POST `/api/v1/english/quiz/generate`
- [x] Go 测试通过
- **Status:** complete

### Phase 4: 孩子端接词库
- [x] 客户端与 liveQuizStore
- [x] 听音选词 / 看图选词走 API
- [x] 502 时「用示例题」兜底
- **Status:** complete

### Phase 5: 浏览器验收
- [x] 19132 五张卡各走完 4 题
- [x] 词库题与本地兜底都看到
- **Status:** complete

## Decisions
- 继续做英语孩子端，不新开学科。
- 首页保持五张题型卡。
- 组句子 / 写单词 / 读一读本轮仍用本地 demo。
- 题型画廊不写掌握度；正式题单 API 不动。

## Errors Encountered
| Error | Attempt | Resolution |
|---|---:|---|
| Docker 里旧 english-server 没有 `/quiz/generate`，19131/19132 都 404 | 1 | `docker compose up -d --build english-server english-app` |
| 组句子连点词卡只记下最后一个词 | 1 | TilePlay 用 ref 累积已选词 |
| 词库图 `sense.png` / 语音 `speech.mp3` 返回 503 `asset_unavailable` | 1 | 未改；题干和中文选项仍可玩。MinIO 网络可达，像是对象键/桶配置问题 |

## Notes
- 设计：`docs/superpowers/specs/2026-09-06-english-type-practice-loop-design.md`
- 计划：`docs/superpowers/plans/2026-09-06-english-type-practice-loop.md`
- 孩子端：http://localhost:19132
- 出题 API：`POST http://127.0.0.1:19131/api/v1/english/quiz/generate`
