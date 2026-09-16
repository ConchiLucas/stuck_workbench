# 单词乐园题型练习闭环设计

## 目标

把已经在跑的 `english-app`（`http://localhost:19132`，单词乐园）做成和其他四门课孩子端同一套体验：首页题型画廊、点进去一题一屏、立即对错、四题一组、结果页；其中「听音选词」「看图选词」改由 `english-server`（`http://localhost:19131`）按词库实时出题。

## 已锁定的默认（本次不问细节）

这些选择按刚打开的拼音 / 科普 / 英语 / 算数孩子端来定。之后按最终画面再改。

| 项 | 默认 |
| --- | --- |
| 产品 | 继续做儿童英语 `english-app` + `english-server`，不新开第五学科 |
| 首页 | 保持五张题型卡，不改回「今天学英语 / 单词地图」主入口 |
| 可玩题型 | 五种全部能练完；后三种仍用本地 demo |
| 接词库的题型 | 只接 `听音选词`、`看图选词` |
| 对错反馈 | 点选项后立刻对错，对了约 450ms 进下一题（跟拼音 demo） |
| 孩子 | 固定 `childId=1` |
| 每组题量 | 4 题 |
| 对战 / 完形 / 抢词 | 不做，那是 `english-word` |

## 现状

`english-server` 已经能列出 20 个主题、200 个单词，也能创建今日计划并判题。`english-app` 首页已经是五张卡，本地 demo 能点能选，但：

- 选择后没有对错提示（要到结果页才知道）
- 听音 / 看图用的是写死的 apple/cat，没有打到 200 词词库
- 首页没有入口去走已有的 `/practice/:planId`

内容后台 `http://localhost:19091/english` 继续负责素材，孩子端不调用它。

## 非目标

- 不改拼音、科普、算数、识字孩子端
- 不把组句子、写单词、读一读接到数据库（本轮保持本地四题 demo）
- 不新增键盘录音评分、对战、账号系统
- 不把首页改成识字那样的学习地图（那是后续一轮）
- 不把题型练习写入 `attempts` / 掌握度（那是正式题单的职责，现有 `/practice/:planId` 保留不动）

## 方案

沿用拼音题型卡的做法：画廊练习走 `POST /quiz/generate`；正式题单仍走现有 plan API。

```text
english-app :19132
  ├─ /                         五张题型卡
  ├─ /types/audio-choice       听音选词：API 出题，失败则本地 demo
  ├─ /types/image-text         看图选词：API 出题，失败则本地 demo
  ├─ /types/card-builder       组句子：本地 demo
  ├─ /types/input-gap          写单词：本地 demo
  ├─ /types/reading-qa         读一读：本地 demo
  └─ /types/:id/result         四题结果

english-server :19131
  └─ POST /api/v1/english/quiz/generate
```

## 实时出题

请求：

```json
{ "type": "listen", "excludeTargetIds": [101, 108] }
```

`type` 只允许 `listen`、`look`。前端映射：`audio-choice → listen`，`image-text → look`。

规则：

- 只查 `subjects.code = 'english'`
- 目标词和三个干扰项优先同一 `module_code`，不够再从其他主题补
- 四个选项的 `kpId` 和英文词互不重复
- `excludeTargetIds` 避免连出同一目标词
- `listen`：必须有中文释义；有音频则给 `speechUrl`，否则只给 `speechText`（前端 Web Speech 兜底）
- `look`：必须有中文释义；有义图则选项带 `imageUrl`，否则选项只显示中文
- 抽象问候语（`hello` / `please` / `sorry` 等）不作为 `look` 目标词
- 选项顺序每次打乱；响应里带 `answerIndex`（仅用于题型画廊，正式题单仍然不把答案提前给前端）

响应形状对齐拼音：

```json
{
  "instanceId": "…",
  "type": "listen",
  "stem": "听一听，选出你听到的单词",
  "targetId": 101,
  "speechText": "apple",
  "speechUrl": "/api/v1/english/words/101/speech.mp3",
  "visual": { "kind": "sound" },
  "options": [
    { "id": 101, "label": "苹果", "imageUrl": "/api/v1/english/words/101/sense.png" }
  ],
  "answerIndex": 0
}
```

`look` 的 `visual` 为 `{ "kind": "word", "text": "apple" }`，不自动播音频。

## 孩子端交互

- 横屏 `1024×768` 为主
- 顶栏：关闭、`当前 / 4` 进度条、上一题 / 下一题
- 选择题：点中后锁住；对了显示勾并自动前进；错了可再点一次，第二次仍错则揭晓再前进
- 组句子 / 写单词：拼对或写对后才能进下一题；结果页仍按对错统计
- API 出题失败：显示「服务返回了无法识别的内容」同类文案，并提供「用示例题」按钮，切回当前四题本地 demo，保证离线/502 时画廊仍能玩
- 同一时刻只播一段音频；切题即停

## 完成标准

1. 首页仍是五张卡，点进去都能做完 4 题并看到结果页。
2. 听音选词、看图选词在 `english-server` 健康时使用词库，连点「再练一次」不会马上重复同一目标词。
3. 点选项能立刻看出对错。
4. `english-app` 仍然只请求 `english-server`。
5. 现有 plan / practice 接口行为不变。
6. `npm test`、`npm run build`、`go test ./...` 通过；浏览器在 `19132` 把五张卡各走一遍。
