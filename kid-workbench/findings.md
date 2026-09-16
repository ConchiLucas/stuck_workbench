# Findings: 识字素材完善

## Requirements
- 只完善识字素材及必要校验/预览；保持 App 首页、题型布局、后台菜单
- 三题型：看字选义、看义选字、写一写；必须实际看图、听音、试做
- 缺图不能用文字/emoji/无关图充数；抽象字不强行配图
- 试做不写学习账本；不发布题包；代码修复才定向重建服务

## Research Findings

### 数量与结构
- 300 字，30 组 × 10。内容后台 `http://localhost:19091/literacy`
- 字图、读音：300/300
- 义图：290/300；第7组 10 字全部无义图（规则）
- 书写模板：本轮后 300/300

### 可出题校验
- 看字选义/看义选字：290 可出题；10 个 `missing_sense_image` = 第7组
- 写一写：300 可出题
- 校验不检查义图含义或选项可区分性

### 产品规则
- `noSenseImageChars`：的了不在有是你他她们这那吗呢吧着过和与也很就都把让从向比为而以且或
- 第7组恰好是：你他她们的了不在有是
- 试做冻结当前修订，不写账本

## Technical Decisions
| Decision | Rationale |
|----------|-----------|
| 第7组不配义图，只补写一写 | 产品词表明确无稳定实物含义 |
| 同组多解义图改 prompt 后重生成 | 文件存在不等于教学合格 |
| 肩/今早明/油醋/雾烟尘不擅自定规则 | 列为待确认 |

## Issues Encountered
| Issue | Resolution |
|--------|------------|
| 本机无 psql | docker exec postgres |
| 列表 capabilities 缓存旧「缺模板」 | 刷新页面 |

## Visual/Browser Findings
正式记录见 docs/menu-alignment/识字素材完善记录.md。
