# 识字手写效果核验（2026-09-12）

本核验直接执行 `literacy-app/src/lib/handwriting.ts` 与 `shared-go/handwriting/evaluate.go`，不复制算法。固定使用 shared-go 已提交的 一、山、水、的 四个模板，替换 App 的 HanziWriter 网络加载器以保证输入一致；不访问数据库、不生成作答事实、不要求任务同步。

复现（仓库根目录）：

```sh
node scripts/verify-handwriting-alignment.mjs > /private/tmp/handwriting-alignment.json
(cd shared-go && GOCACHE=/private/tmp/handwriting-go-cache go test ./handwriting)
(cd literacy-app && npm test -- --run src/lib/handwriting.test.ts src/lib/handwritingOrientation.test.ts)
```

需要现有 literacy-app/node_modules 的 TypeScript 和匹配 shared-go/go.mod 的 Go。脚本在临时目录生成 Go 调用入口，退出时清理，Go 构建缓存留在临时目录。Go 默认用户缓存不在本会话可写目录内，所以显式使用 /private/tmp 缓存。

## 修复证据与当前效果

原始 App 的 FreeDrawPad 返回屏幕像素（Y 向下），旧 looksLikeCharacter 直接与 HanziWriter medians（Y 向上）比较。后端已将屏幕归一化坐标翻成向上。此次主任务只在 App 模板读取处使用 `y: -y` 统一方向，归一化会消除平移，不改变 inkMatch 通用算法、界面入口、内容来源或阈值。

| 同一个屏幕输入 | 修复前 App | 修复后 App | Go |
| --- | --- | --- | --- |
| 标准水 | 通过，score 0.8 | 通过，score 1 | 通过，minStrokeCover 1 |
| 的缺最后一笔 | 通过，score 0.95 | 不通过，score 0.5 | 不通过，minStrokeCover 0.5 |
| 一/山/水/的标准字及平移缩小 | 均通过，但部分覆盖评分偏低 | 均通过，score 1 | 均通过，minStrokeCover 1 |
| 四字空白 | 均不通过 | 均不通过 | 均不通过 |
| 山缺最后一笔 | 不通过 | 不通过 | 不通过 |
| 水缺最后一笔 | 通过 | 通过 | 不通过 |
| 山/水/的四条横线或密集32条横线 | 通过 | 通过 | 不通过 |
| 一/山完整字形拆成多段抬笔 | 通过 | 通过 | 不通过 |

`handwriting-alignment-before.json` 是方向修复前的50个确定性样本结果；`handwriting-alignment-after.json` 是修复后的同一组结果。后者 `appCanvas` 执行当前 looksLikeCharacter；`legacyCanvas` 用相同 inkMatch 重现旧版未转换方向的输入；`appSameAxis` 是统一向上坐标的算法控制组。

修复后50个样本的 App 实际屏幕路径与同向控制组通过/失败结果一致。与 Go 仍有15项通过/失败差异，全部为 App 通过、Go 不通过。这是刻意构造的边界集，包含重复样本（如某字缺末笔和逐笔删除的最后一项），不是儿童真实书写准确率，也不能用35/50描述模型质量。

## 有意保留的策略差异

两侧共享64格归一化栅格、10%边距、用户/模板半径3/4、每笔20个采样点、临近半径5，以及最小笔画覆盖0.52、平均覆盖0.62、精度0.22的基础阈值。两侧都是模板几何覆盖判定，不能等同OCR、规范笔顺或人工书法评价。

Go `ink-match-v1` 在此基础上额外要求墨迹占格比例不超过0.65，且每个模板笔画至少有某一条用户笔画覆盖70%。因此它能拦截本样本中的乱画，也会拒绝某些 App 容许的分段书写与缺笔情况。这个要求并非笔顺检查或严格的一笔对应一笔：一条用户笔画可以匹配多个模板笔画。

App 的 looksLikeCharacter 先要求至少一条笔画包含两个点；Go 还验证点坐标/时间、笔画数和模板格式，输出带 evaluatorVersion 的结果及更多指标。两侧输入协议不同（App 像素且无时间戳、Go 0..1屏幕坐标和非递减时间戳）；不能把同一份数值数组未经坐标转换直接同时传入当作一致性测试。

当前不建议修改 Go policy，也不建议为了表面一致把 Go 防乱画阈值删掉。方向修复消除了明确的坐标错误；是否统一抬笔容忍度、缺笔与乱画判定，是后续需单独决定的产品策略。若调整 Go 阈值，遵守现有代码要求发布新 policy 版本，不重评历史回执。本次可以说明标准输入、空白和缩放效果已验证，不能说明所有手写判定完全一致。

## 核验范围

50例包含空白、真实模板标准字、平移缩放、垂直镜像、逐笔删除、仅首笔、稀疏/密集横线乱画、可满足后端64笔限制的分段完整字。分段样本保持几何线条基本不变，刻意暴露单笔覆盖限制。

这验证的是确定性评分函数及 App 坐标入口，不覆盖浏览器指针采样、音频、动画、联网 HanziWriter 模板漂移、真实儿童笔迹、服务端回执或任何数据写入。模版来源差异仍存在：App 运行时加载 HanziWriter，后端使用题目内固定模板。本测试故意用相同模板隔离算法，不代表生产两侧模板来源已经统一。

执行结果：比较脚本50例断言通过；Go `go test ./handwriting` 通过；App `handwriting.test.ts` 与 `handwritingOrientation.test.ts` 共2文件6测试通过。脚本明确断言修复后的屏幕输入与同向控制通过结果一致、标准与缩放样本 App score=1/Go通过、空白拒绝、Go拒绝乱画，并保留旧坐标差异和现有策略差异的断言，避免误报全面一致。
