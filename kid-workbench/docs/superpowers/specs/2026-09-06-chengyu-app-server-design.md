# 成语小园孩子端设计

## 目标

给已经种子在家长看板里的 `chengyu` 学科补一套独立孩子端：`chengyu-server` + `chengyu-app`。孩子打开首页就能点题型卡练成语，作答写入共享掌握度。

对照物是 `phrase-server` / `phrase-app`（新学科独立 API + 四卡画廊 + 真实题单），交互节奏对照四门课孩子端：4 题一套、可错一次、立刻对错。

## 默认

| 项 | 值 |
|----|----|
| 标题 | 成语小园 |
| API | `:19181` |
| 孩子端 | `:19182` |
| 孩子 | `childId=1` |
| 题量 | 点卡创建 `mode=type` 题单，默认 4 题 |
| 语音 | 浏览器 `speechSynthesis`，`zh-CN`，不接 MinIO |
| 题库 | 共享 PostgreSQL `subjects.code = chengyu`（32 条已有种子） |

## 四张卡

| 卡 | `questionCode` | 孩子看到 |
|----|----------------|----------|
| 听释义 | `meaning` | 听/看成语，选意思 |
| 选成语 | `pick` | 听成语，点出四个字 |
| 看拼音 | `pinyin` | 看拼音，选出成语 |
| 看句子 | `example` | 看例句，选出成语 |

`meaning` / `pick` 已在种子题库里。`pinyin` / `example` 由家长端出题器补上；孩子端若库里还没有这两类题，会按知识点 payload 现场组题并快照进 `plan_items`。

## 非目标

不改英语/短句 App，不加内容后台成语页，不画学习地图，不接抢词。
