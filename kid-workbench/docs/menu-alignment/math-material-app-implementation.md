# 算术素材与 App 本轮实施

2026-09-13；按五端标准实施，未替代中央档案的用户验收结论。

## 用户要求与改法

- 素材端沿用识字 `literacy-group / char-grid / char-card / mini-btn`。继续字图与读音，**不恢复**题型详情编辑、题目列表或试答。`MathDetailsEditor` 仍在仓库、路由未挂。
- 卡片标签：字图已备/缺少 · 读音已备/缺少 · 已启用/已停用。操作：生成字图、读音、重新生成、停用/审核启用。
- `math_assets.enabled` 新增列；同步不得把已停用项改回启用；新行默认启用。`PATCH /api/v1/math/items/:kpId/review` `{enabled}`。
- **诚实限制**：该启停只作用于素材后台列表，**没有**接到 math-server 出题。直播计划仍可能抽到已停用知识点。
- App 首页保持 5 张卡（equation/story/missing/judge/shape）。补充 `math-app/public/cards/*.svg` 示意图。正式题面奶油底、左题面右答案；≤760px 上下堆叠。
- 普通算式只显示 `3 + 5`（目录/生成不再写 `= ?`；KidPlayer 仍会剥掉历史 `= ?`）。无语音。补空/判断保留等号。听音图形必须有音频，播放≠作答；缺音频明示且不解锁。
- 数量图/图形没有位图时用 ★/🍎/🍓 或 `ShapeGlyph`，不再用「图片未准备」挡住可练题。

## 文件

- `content-admin/backend/internal/db/db.go`、`internal/math/service.go`、`internal/http/handler_math.go`、`router.go`：enabled 列、SetEnabled、审核接口。
- `content-admin/frontend/src/features/math/MathPage.tsx`、`MathPage.test.tsx`、`api/math.ts`。
- `packages/math-player/src/KidPlayer.tsx`、`player.css`：readOnly、字形回退、760 断点。
- `math-app/src/pages/mathDetail.css`、`public/cards/*.svg`、`QuestionTypeCard.tsx`、`App.test.tsx`。
- `shared-go/mathcontent/defaults.json`：加法 `3 + 5`、减法 `9 − 4`。

## 验证

- `cd content-admin/backend && go test ./internal/math ./internal/http` 通过。
- `cd content-admin/frontend && npx vitest run src/features/math`：6 tests 通过；素材页无「题型详情」、有「停用」。
- `cd math-app && npx vitest run`：86 tests 通过。
- 隔离测试未向真实孩子提交。

## 交接

生产变更：content-admin、math-app；共享 player 需各宿主重建。math-server 快照契约未改（读侧转换）。未宣称用户视觉验收。
