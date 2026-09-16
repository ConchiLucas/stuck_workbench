# 逻辑孩子端题型试玩设计

## 家长看板里已经有什么

`http://localhost:19081/subjects/logic` 是掌握度看板，不是孩子答题页。它展示：

- 学科「逻辑」，62 个知识点，五个模块全部未开始。
- 模块：找规律 15、分类 12、排序 12、图形推理 12、找不同 11。
- 种子 payload + 出题器会写成 `questions.code = pick1/pick2`（各 62 道）。

出题规则（`parent-dashboard/backend/internal/quiz/logic.go`）：

| 模块 | payload.kind | 孩子看到 |
|------|----------------|----------|
| 找规律 `pattern` | `pattern` | 上方 emoji 序列，选出下一个 |
| 分类 `classify` | `classify` | 四个选项里找出和其他不一样的 |
| 排序 `order` | `order` | 上方乱序序列，选出正确文字顺序 |
| 图形推理 `shape_reason` | `pattern` | 同找规律，几何形状 |
| 找不同 `diff` | `classify` | 同分类，强调「哪一个不一样」 |

选项是 emoji 或短文字（含 `1️⃣`、`1 → 2 → 3`）。序列走 `visual.kind = seq`。

## 孩子端要做成什么

新建 `logic-server` `:19191` + `logic-app` `:19192`。交互对齐算数/科普题型试玩：

1. 首页五张卡，对应五个模块。
2. 点卡连续 `POST /api/v1/logic/quiz/generate` 四次，`excludeTargetIds` 为知识点 id。
3. 全屏 4 题，点选项约 450ms 后进下一题；最后一题进结果。
4. 结果列出对错；再练一次重新出题。
5. 出题失败显示孩子能懂的话，可「再试一次」或「用示例题」。

`type` 取值：`pattern` / `classify` / `order` / `shape_reason` / `diff`。服务端只读该模块的 `pick1`，不要求发布状态，不写掌握度、不建每日计划。

## 非目标

不改家长看板、内容后台、识字端。不把逻辑并进现有四个 App。
