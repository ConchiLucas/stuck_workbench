# Task Plan: 成语小园

## Goal
给 `chengyu` 学科做独立孩子端 + API：四张题型卡、4 题一套、立刻对错、写入共享掌握度。

## Current Phase
Phase 4

## Phases

### Phase 1: 对照现有学科锁定默认
- [x] 参考 phrase 前后端 + 已有 32 条成语种子
- [x] 写入设计说明
- **Status:** complete

### Phase 2: chengyu-server
- [x] quiz / plan / practice / HTTP
- [x] Go 测试通过
- **Status:** complete

### Phase 3: chengyu-app
- [x] 四卡画廊与练习页
- [x] 前端测试与 build
- **Status:** complete

### Phase 4: 接入 Docker 并浏览器验收
- [x] compose 19181 / 19182
- [x] 听释义走完 4 题到结果页；看拼音能出拼音题
- **Status:** complete

## Decisions
- 端口 19181 / 19182
- 题型 meaning / pick / pinyin / example
- pinyin / example 无库内题目时按知识点 payload 现场组题
- 不接 MinIO，浏览器朗读

## Errors Encountered
| Error | Attempt | Resolution |
|---|---:|---|
| 批量 rsync 被自动审核拦截 | 1 | 改为逐文件写入 |
| npm install 空 lock 报 edgesOut | 1 | 复用 phrase-app lockfile |
