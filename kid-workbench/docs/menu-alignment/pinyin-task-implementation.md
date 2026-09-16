# 拼音题目后台实施记录（2026-09-13）

依据用户最新五端授权及 `03-五端菜单实施标准.md`，本轮补齐题目后台独立生成与保存。旧档案“生成缺口只记录不做”已被替代。部署与五端浏览器验收由主代理统一记录，本文不宣称已部署或用户验收。

## 用户要求与实现

- 本后台沿用识字任务卡片、暗色宿主表单及全屏题目弹窗，不显示发布/状态/删除/重新生成等操作。列表包含任务类型、名称、题型、素材分组、题数、时间和查看入口。
- 右上保留一个小型“生成拼音题目”入口。识字取消大标题和新建工具栏的简化方向继续保留；此小入口是本轮明确要求真实生成能力的必要入口，不恢复原完整任务编辑面板。弹窗选择四题型和题数、填写可选名称，生成后重新读取已保存题包。
- listen、inword、shape 只读 `pinyin_assets`；blend 只读已启用且有 `speech_url` 的 `pinyin_syllable_assets`。不调用 App 的实例生成接口；不读写孩子作答或掌握表。字中找拼音直接沿用素材映射 `kp_id → word_text` 及同素材 `word.mp3`，不自行推测汉字对应拼音。
- 每道题保存完整题干、四个不同选项的顺序/ID/标签、正确ID、目标素材ID/表/分组及题型。shape 使用四线格，blend 保留声母+带声调韵母，不套识字图片题尺寸。
- 生成前实际 GET 素材音频，验证 HTTP 200、MP3 帧或 ID3 签名和8MB大小上限，禁止上游重定向及任意URL。音节URL允许且只允许 `?v={16位小写hex}`。缺素材会明确失败，音频读取完后才开始事务。
- 真实 MP3 字节按 SHA256 存入专用 `pinyin_question_task_media`，题包引用其不可变地址；字母旧媒体可变也不会改变已生成题包。题目与媒体在同一事务写入，音频失败不留半包。新表仅有 `pinyin_question_tasks` 和 `pinyin_question_task_media`。
- 预览复用 `@kid-workbench/pinyin-player`，禁止合成音兜底，播放不会选择答案，选择反馈只存在 React 本地状态。没有预览 POST 作答请求。关闭和 Esc 都恢复触发按钮焦点。

## 文件

- `task-admin/backend/internal/pinyintask/service.go`、`service_test.go`：独立素材生成、快照与音频存储、输入/媒体校验。
- `task-admin/backend/internal/http/handler_pinyin.go`、`pinyin_tasks_test.go`、`router.go`：列表/生成/详情/不可变媒体API及HTTP回归。
- `task-admin/backend/cmd/server/main.go`：仅迁移题目后台专用表、接入服务。
- `task-admin/frontend/src/pages/PinyinTasksPage.tsx`、`PinyinTasksPage.test.tsx`、`pinyinTasks.css`：精简列表、小生成入口、同款预览弹窗。
- `task-admin/frontend/src/App.tsx`、`src/layout/AppShell.tsx`：`/pinyin`入口。
- 前端 `package.json`、lock、`.npmrc` 和 `task-admin/Dockerfile`：共享拼音包采用 `install-links`，Docker `npm ci --install-links`，现有包保持同React实例。

## API

- `GET /api/v1/pinyin/question-tasks`
- `POST /api/v1/pinyin/question-tasks`，如 `{"title":"拼音练习","types":["listen","inword","shape","blend"],"count":8}`；1至40题，题数不少于题型数。
- `GET /api/v1/pinyin/question-tasks/:id`
- `GET /api/v1/pinyin/task-media/:sha256.mp3`

## 验证

- `cd task-admin/backend && go test ./...`：全套通过，包含识字、算术、复习现有回归。
- `cd task-admin/frontend && npm test`：4文件14测试通过。
- `cd task-admin/frontend && npm run build`：TypeScript和Vite构建通过。
- 生成测试覆盖四题型、4不同选项与正确ID、素材修改后重载相同原题、版本化音频冻结、非法URL/HTML/重定向拒绝、缺音频零题包、无学习表写入。
- HTTP集成用隔离SQLite与真实HTTP测试服务器，验证生成→GET重载→冻结媒体读取→素材失败不追加题包，数据库仅出现素材测试表和两个任务表。
- 前端覆盖本地试答没有POST、生成后GET保存内容、Esc关闭/焦点恢复、列表失败与空态区分。

## 实际边界与后续验收

- 不切换 App 的正式取题来源；没有真实孩子作答，也未删除任务/素材/卷。
- 正式素材不足4份时对应题型会失败。真实音节录音导入与五端部署/窄屏媒体验证由主代理集成完成。
- 网络已保存但响应丢失时重试可能生成另一题包，目前没有持久幂等键；同页面生成按钮有并发锁。后续若增加自动重试，应先实现服务端幂等处理。
- 本轮不提供单题替换、出题范围筛选或发布状态操作；这些不是四题型生成保存及同款预览的必要条件。

## 浏览器回归修正：题包预览纵向裁切

主代理发现 `task-preview.png` 第一题下排选项被截断。根因是 App 播放器依赖固定可用高度，而任务是自然高度的题卡列表；共享 `height:100%`、`minmax(0,1fr)` 和 size-container 选项组合不能为宿主贡献完整高度。

仅 task 的 `Preview` 增加 `.pinyin-task-player` 容器及宿主覆盖：题面/题干/答案区采用自然高度、可见溢出；选项两行各至少180px（包含声音题播放区与52px选择按钮）；来源信息跟在题面正常文档流之后。宿主按容器宽度760切上下、1100调整拼音组间距；不改共享 App CSS，不强行把整个题包塞进单屏。

几何验收：原端口打开真实题包，1280×720、1024×768、390×844分别检查四题型。每题4个 `.option-button` 均满足 `option.bottom <= questionBody.bottom + 1`，选项自身 `scrollHeight <= clientHeight + 1`，`questionBody.bottom <= sourceDetails.top + 1`，弹窗 `scrollWidth <= clientWidth + 1`。窄屏允许纵向滚动，滚到第二排并确认音频选择按钮及来源完整；关闭/Esc和焦点恢复继续回归。该几何检查由主代理实际浏览器执行，不以jsdom测试代替。

此修正后前端14测试、TypeScript及Vite构建通过；部署重建task服务由主代理执行。
