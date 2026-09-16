# Task Plan: 素材页跳转孩子端

## Goal
识字到成语（含短句）每个素材页右上角加「孩子端」按钮，打开对应孩子端首页。

## Current Phase
Phase 2

## Phases

### Phase 1: Discovery
- [x] 对照截图端口与 page-heading
- **Status:** complete

### Phase 2: Shared link + pages
- [ ] KidAppLink 测试失败后实现
- [ ] 九个素材页右上角挂上
- **Status:** in_progress

### Phase 3: Verify
- [ ] vitest + 浏览器点开链接
- **Status:** pending

## Decisions Made
| Decision | Rationale |
|----------|-----------|
| `<a target=_blank>` 而不是 button | 外链；拼音/英语「题型」仍是唯一 heading button |
| 含短句 | 截图有 `:19172`，后台已有短句菜单 |
| href 用当前 hostname + 固定端口 | 本机 localhost / 局域网 IP 都能跳 |

## Errors Encountered
| Error | Attempt | Resolution |
|-------|---------|------------|
|       | 1       |            |
