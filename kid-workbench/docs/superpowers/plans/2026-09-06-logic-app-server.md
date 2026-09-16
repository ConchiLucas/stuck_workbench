# 逻辑孩子端实施计划

**Goal:** 孩子打开逻辑首页点五种题型卡，都能从库里拉 4 道 `pick1` 做完并看到结果。

**Ports:** server `:19191`，app `:19192`

## 阶段

- [x] `logic-server`：`POST /api/v1/logic/quiz/generate`，按模块过滤 `pick1`
- [x] `logic-app`：五卡首页、4 题练习、结果、失败回退
- [x] Docker Compose 接入，curl + 浏览器验收
