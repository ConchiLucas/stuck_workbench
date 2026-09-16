# 拼音素材与 App 本轮实施

2026-09-13；按五端标准实施，未替代中央档案的用户验收结论。

## 用户要求与改法

- 素材端沿用识字的 `literacy-group / char-grid / char-card / mini-btn`，继续保留字母及例字媒体，不恢复题型、题目或试做入口。
- 补齐原先不可管理的音节：列表显示声母、带调韵母、例字、录音有无、启用状态；可编辑例字、停用/审核启用、试听和导入真人包。知识点 ID、发音和声调保持稳定，避免把旧 ID 改成其他音节。
- 为当前 16 音节随镜像保留真实录音；作者 Chen Wang、上游 CC-by-sa，来源/原文件 SHA 在 `content-admin/backend/assets/pinyin-human-pack/sources.json`。未用合成音冒充真人音。
- 读取录音不产生 TTS，也不写学习表。导入写 MinIO、音节素材 speech_url；音节审核仅更新素材 enabled。
- App 首页和正式提交逻辑保留，关闭两练习宿主浏览器合成音兜底。共享组件只有 `Audio.play()` 成功才解锁选项，拒绝或 error 显示错误并取消可选状态。切播放与卸载会停止前一音频。公共组件 API 没变。

## 新接口与媒体约定

- `GET /api/v1/pinyin/syllables` → `{items,total}`，字段 camelCase。
- `PATCH /api/v1/pinyin/syllables/:id` → `{speechText,enabled}`；例字 1–20 字；不存在 404。
- `POST /api/v1/pinyin/syllables/human-pack` → `{generated,skipped,failed,errors}`。
- 导入目录：`PINYIN_SYLLABLE_PACK_DIR`，镜像设 `/app/assets/pinyin-human-pack`，本地默认 `assets/pinyin-human-pack`；未改变原有字母导入目录。
- 音节 speech_url 保存相对地址 `/api/v1/pinyin/syllables/{id}/speech.mp3?v={16位小写hex}`。content-admin 与 pinyin-server 均提供读取。
- MinIO `pinyin/syllables/{id}/speech-{hash}.mp3` 永久保留旧字节；无 v 为稳定最新别名 `speech.mp3`。历史代理须保留合法 v；不合法版本拒绝。
- 两个 blend 生成器仅选择 enabled 且 speech_url 非空素材；不足 4 条明确失败。题目后台另行校验实际字节。

## 本代理验证

- `content-admin/backend: go test ./internal/pinyin ./internal/http` 通过，含导入、缺录音、编辑审核、哈希旧版本重读和无效版本测试。
- `pinyin-server: go test ./internal/asset ./internal/http ./internal/quiz` 通过，含正式作答既有回归及无音节录音拒绝生成。
- `content-admin/frontend: npx vitest run src/features/pinyin/PinyinPage.test.tsx src/features/pinyin/SyllableMaterials.test.tsx` 3 tests 通过；`npm run build` 通过。
- `pinyin-app: npx vitest run` 50 tests 通过；`npm run build` 通过；另补 Audio.onerror 启播后失败测试后，共享组件专项 8 tests 通过（全套此前50条＋新增1条）。
- 测试使用内存数据库/模拟播放，未给真实孩子提交答案。

## 集成交接

生产服务变更：content-admin、pinyin-server、pinyin-app；共享 player 改动需所有宿主更新 node_modules 拷贝后构建。未部署、未删除数据、未改中央档案、未操作 git。

主代理部署后调用一次音节真人包导入，再核验16个录音、四题型真实媒体、五端窄屏。浏览器与部署验收由主代理完成，用户效果验收仍待反馈。

## 主代理审核修正：取消编辑后误保存草稿

发现“编辑例字→取消→停用”会把未保存草稿带入状态更新。已把 mutation 参数改为显式 `{speechText,enabled}`：编辑保存传草稿，状态按钮传 `item.speechText`，取消同时恢复已保存文本。回归先复现失败（收到取消的草稿），修复后素材4测试通过并重新构建；需重新构建 content-admin 镜像。

## 原端口验收修正：旧缓存题无音频

关闭TTS后，旧localStorage恢复的blend快照无speechUrl，无法继续。新页面在当前题无result/pending/submitting且缺必需录音时显示“这份旧题未保存读音，请重新练习”。重练通过restart+ensure，新增保留已确认entries的可选模式，保留原question/result并补齐未完成题量。session任意题仍有pending/submitting则拒绝此重启，只提供返回待确认题入口。当前题已有回执/待确认提交仍走原流程，不补写旧快照URL，不替孩子作答。

回归先复现无提示失败；修复后全套53测试与build通过；另补跨题pending保护测试。受影响部署仅pinyin-app。
