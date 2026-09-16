# 拼音：进度与孩子知识库实施记录

2026-09-13。按用户最新五端授权，知识库纳入；历史文档的排除项已被替代。未改学习写入、掌握算法、日周月统计口径，未部署、提交或删除数据。

## 进度

专用 `/subjects/pinyin` 已使用 `PinyinMastery` 的字母听/找/认及音节 blend 状态、详情和冻结历史。保留这条实现，不复制错因库。旧 `MasteryMatrix` 的拼音现场试做入口已移除（正常拼音路由原本已绕过它），Math 分支保留。

- `frontend/src/components/mastery/MasteryMatrix.tsx`：移除潜在拼音现场试做分支。
- `frontend/src/components/question/PinyinHistoryQuestion.tsx`：补媒体版本真实性说明，冻结选项不等于冻结所有旧音频。
- `backend/internal/service/pinyin_review.go` + test：严格接受 `/api/v1/pinyin/syllables/{正整数}/speech.mp3`，可选且仅允许 `v={16位小写hex}`，保留版本查询；旧 items 限定不变。
- `backend/internal/http/pinyin_media.go` + test、`router.go`：原端口 `/api/pinyin/syllables/:assetId/speech.mp3` 代理到固定 CONTENT_ADMIN_URL；验证 ID/版本、不跟随跳转，不接受任意上游。

## 孩子知识库

既有能力档案、按知识点作答记录、错误筛选及保存分析继续使用识字同款宿主。无新增日历或矩阵。后端已有 `decodePinyin` 验证实例归属、稳定选项与回执；复用这些事实，未重新生成历史。

- `frontend/src/components/knowledge/PinyinEvidence.tsx`、CSS、test：接共享 `@kid-workbench/pinyin-player`，四题型 `readOnly` 且禁止 TTS，不传提交回调；按原顺序展示选择与正确答案说明。缺四选项、重复 ID、答案/所选项不在选项、缺字母文本、缺词或拼读视觉字段时明确无法还原；仍保留可核对的事实。媒体使用现有 child+attempt 代理，不直接信任快照任意 URL；旧无版本音频提示可能更新。
- `AnswerEvidence.tsx`：拼音专用分派；其他学科不变。`api/knowledgeTypes.ts` 加冻结 visual.text 字段。
- `package.json`、lock、`.npmrc`、Vite、`Dockerfile`：共享包构建与 React dedupe，Docker 先复制共享包；不在包内安装第二份 React。
- `WrongAnswersPage.tsx`：拼音仅四种实际题型，使用 App 名称；切学科清空旧 skill；全部学科仍保留原筛选。
- `backend/internal/knowledge/library.go`：拼音四能力文字为听音选字母、字中找拼音、看形认读、声韵拼读，不改技能编码。
- `ReviewSuggestionPage.tsx`：保存拼音分析后明确专属复习生成与归因链路未接入，隐藏生成/重试入口。候选页原有“可保存分析；这个学科的复习出题尚未接入”继续有效。
- `FunctionalNavigation.test.tsx`、`ReviewSuggestionPage.test.tsx`：覆盖过滤与生成限制。

## 验证

- 知识库 frontend `npm test`：11 files / 28 tests 通过。新增功能先跑红再实现。
- 知识库 frontend `npm run build`：通过。
- 知识库 backend `go test ./internal/knowledge ./internal/http`：通过（隔离夹具，不写真实孩子）。
- 进度 backend `go test ./internal/service ./internal/http -run 'PinyinReview|RewritePinyin|PinyinMedia|PinyinGlyph|PinyinSyllableMedia'`：通过。
- 进度 frontend `npm test && npm run build`：15 tests 通过，构建通过（既有大 chunk 提示）。

## 确切限制与交接

- 拼音普通题目生成由题目端代理完成；本记录不把它等同于“复习建议→原题/变式草稿→版本依据→后续作答归因”。后者仍不支持拼音，保存分析可用。
- 完整复习链最低需题目端支持 pinyin instance 来源验证、原题与变式生成、版本/题目来源保存，以及拼音回执和归因查询；不是解除 UI 禁用即可。
- 历史 listen 真实样本已有；其他三个历史场景使用组件/服务夹具验证，未伪造孩子历史。
- 浏览器窄屏/实际媒体、五端部署由主代理集成验收，本子任务无截图、无新预览服务，无用户视觉验收声明。
