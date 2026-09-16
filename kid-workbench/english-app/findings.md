# Findings

- 2026-09-02：英语孩子端从 96 题型收成 7 个脚本，再收到现在的 5 张题型卡。
- 2026-09-06：四门课孩子端（拼音 19112、科普 19122、英语 19132、算数 19142）都是题型画廊。英语卡能点，但选择后没有对错，听音/看图是写死的 apple 组。
- `english-server` 已有 20 主题 × 10 词、home/plan/practice；没有 `/quiz/generate`。
- 拼音画廊用 `POST /api/v1/pinyin/quiz/generate`，响应带 `answerIndex`；经 Docker Nginx 的 19112 当前 502，直连 19111 正常。英语接线时必须让 19132 的 `/api` 回到 JSON，不能把 HTML 502 丢给 `response.json()`。
- 现有 `english-app` 的测试明确禁止练习页出现「答对啦」；本轮要改成立刻对错，并改测试。
- 正式 `/practice/:planId` 只支持 `listen` / `picture`，首页还没入口；本轮不把画廊改成今日题单。
- 2026-09-06 实现后：`POST /api/v1/english/quiz/generate` 在 19131 和经 19132 nginx 都返回 JSON。听音选词打到词库（如 blue/helicopter），看图选词打到词库（如 rain/egg）。词库配图 `sense.png` 和语音 `speech.mp3` 仍 503 `asset_unavailable`（MinIO 可达，对象未取到）。关停 english-server 后听音选词出现「用示例题」，点开是苹果 demo，错选香蕉显示「再试一次」，再选苹果显示「答对了」。

