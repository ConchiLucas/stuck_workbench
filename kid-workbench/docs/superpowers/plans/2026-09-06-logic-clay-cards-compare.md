# 逻辑黏土卡与比较题型

**Goal:** 首页六张黏土插图卡，第六张「比较」能从库里出 4 题。

**Architecture:** 成语卡同款 `<img>` 全出血；种子加 `compare` 模块；logic-server 按模块码出 `pick1`。

**Tech Stack:** React + Vite, Go, PostgreSQL seed, PNG in `logic-app/public/cards/`

## 阶段

- [x] 六张 `/cards/{type}.png`，首页 2×3 不再留空
- [x] 种子 `compare` 12 KP；catalog 计数 1236→1248、logic 62→74
- [x] `logic-server` 接受 `compare`；孩子端练习/示例题/测试
- [x] 重建镜像，浏览器六卡 + 比较 4 题到结果
