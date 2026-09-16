# Task Plan: 识字素材完善

## Goal
实际检查并修复素材后台「识字」未达出题要求的素材（字图、义图、读音、书写模板、选项可区分性），用素材试做验证，记录技术校验与教学效果，不改 App/菜单、不写孩子账本、不发布题包。

## Current Phase
Phase 4 文档已写，本轮调查与可明确修复已完成。

## Phases

### Phase 1: Requirements & Discovery
- [x] 读 AGENTS.md、按菜单推进方案、01-识字.md
- [x] 对齐三题型契约、可出题校验、书写模板协议、冻结预览
- [x] 盘点 300 字素材与能力接口
- [x] 记录产品规则（不要义图、不强制配图）
- **Status:** complete

### Phase 2: Visual & audio audit
- [x] 第1组逐字看图、听音、试做三题型
- [x] 其余组义图质量抽查/全量看图，标问题清单
- **Status:** complete

### Phase 3: Closed-loop sample then batch
- [x] 优先修第1组问题字，冻结新版本并试做
- [x] 按既有流程补书写模板、替换不合格义图
- **Status:** complete（明确可修的已修；待确认未强行改规则）

### Phase 4: Documentation
- [x] 持续更新 docs/menu-alignment/识字素材完善记录.md
- [x] 在 01-识字.md 追加链接和进度，不覆盖既有记录
- **Status:** complete

## Key Questions
1. 第7组「不要义图」是否应保持不能出看字选义/看义选字？是，沿用 `noSenseImageChars`。
2. 写一写缺模板如何补？POST 既有 Hanzi Writer Data 包装 JSON，不放宽校验。
3. 义图不合格如何替换？调用现有 `POST /literacy/chars/:kpId/sense`，新修订冻结，不覆盖已发布题包。

## Decisions Made
| Decision | Rationale |
|----------|-----------|
| 官方记录写在 docs/menu-alignment/识字素材完善记录.md | 用户指定交付物 |
| 第7组不强行配义图 | 既有产品规则，用户要求不改教学规则 |
| 补模板用 hanzi-writer-data@2.0.1 包装后导入 | testdata/README 既有方式 |
| 只改同组会多解的义图 prompt | 不扩大教学规则 |

## Errors Encountered
| Error | Attempt | Resolution |
|-------|---------|------------|
| CDN 逐字导入模板卡住 | 1 | 改本地 npm pack 后批量 POST |
| 列表仍显示缺少书写模板 | 1 | 刷新 capabilities 查询，API 已是 300 就绪 |
