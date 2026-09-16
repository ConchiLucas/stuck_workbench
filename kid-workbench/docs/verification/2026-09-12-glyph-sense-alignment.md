# 2026-09-12 看字选义四端验收

范围：识字 glyph_sense。实施档案见 [01 识字](../menu-alignment/01-识字.md)。代码与运行检查通过，用户效果验收待反馈。

## 自动验证

| 项目 | 结果 |
| --- | --- |
| literacy-app `npm test` | 11 个文件、42 项通过；包含共享题面、历史只读、键盘焦点、音频独立启用、正式重试耗尽 |
| content-admin/frontend 测试与构建 | 28 项通过，构建通过 |
| task-admin/frontend 测试与构建 | 11 项通过，构建通过 |
| parent-dashboard/frontend 测试与构建 | 15 项通过，构建通过 |
| parent-dashboard/backend `GOCACHE=/private/tmp/kid-glyph-go-cache go test ./...` | 通过，覆盖回执关联、孩子隔离、原题快照、缺媒体、顺序和媒体代理 |
| 四服务最终 Docker 构建 | 全部通过 |

前端旧手写测试在 jsdom 中仍输出 HTMLMediaElement 未实现提示，测试通过。进度前端仍有现存大 chunk 构建提醒。本轮没有把这些问题算作修复。

## 实际页面检查

- App 原端口 19152：山题选水显示“再试一次”，重选山显示“答对啦”；下一题清空选择。点击选项读音不改变选择。1280×720 左字图右四义图；390×844 上下布局无横向溢出（scrollWidth=390，scrollHeight=844）；667×375 采用可纵向滚动的上下布局。
- 素材原端口 19091：识字第一字“一”点击试做，看字选义使用共用组件，五张图全部加载；读音按钮可操作。
- 题目原端口 19201：现有草稿任务 10 中四道 glyph_sense 都使用共享组件，20 张题面图片全部加载。未发布或重新生成题包。
- 进度原端口 19081：首页与历史接口返回 200，冻结字图代理返回 200 / image/png（7737 字节）。当前 question_attempt_receipts 为 0，没有可用于完整原题回放的真实识字回执。
- 完整历史展示使用现有题包第 123 项冻结素材构造隔离 UI 样例，明确写着“非孩子作答记录”。验证了已选项、当时答对、选项不可作答、音频可播放。样例没有写入数据库。真实历史关联和无效数据降级另由 Go 隔离集成测试验证。

[App 截图](glyph-sense-2026-09-12/app.png) · [素材试做截图](glyph-sense-2026-09-12/content-preview.png) · [隔离历史截图](glyph-sense-2026-09-12/history-isolated.png)

样例源文件：[isolated-history-sample.tsx.txt](glyph-sense-2026-09-12/isolated-history-sample.tsx.txt)。需要复验时复制到 parent-dashboard/frontend/glyph-check.tsx，创建带 root 元素并导入该脚本的 HTML，使用现有 Vite API 代理启动临时页面。标题必须保留隔离样例说明；检查后删除两临时文件并关闭服务。本轮临时端口 19383 已关闭，临时源码已移除。

## 部署与数据检查

在项目根执行：

```sh
docker compose build literacy-app content-admin task-admin backend
docker compose up -d --no-deps --no-build --wait --wait-timeout 60 literacy-app content-admin task-admin backend
```

四服务最终 healthy，19152 / 19091 / 19201 / 19081 均 HTTP 200。部署前后容器 ID 与启动时间比对：只有上述四服务变化，其余容器（含数据库、孩子知识库）保持不变。未删除数据卷。

学习账本数量部署前后均为：attempts 85、question_attempt_receipts 0、daily_stats 4、flower_ledger 1。本轮核实了计数不增；前后摘要序列化口径未能确认相同，因此不宣称通过内容哈希一致验证。浏览器验收只操作本地演示/素材试做/只读题包与样例，没有向孩子提交测试作答。

精确按本轮部署前和中间版的四服务镜像 ID 清理时，Docker 均返回 No such image；再查镜像清单，这些旧 ID 已不存在。保留当前镜像及其他已有历史标签，没有运行全局 prune。

## 审核与遗留

独立需求审核、代码审核通过。审核中补齐历史必需字图和四义图校验、正式重试耗尽文案、折叠区焦点陷阱；浏览器检查补齐只读音频 aria-disabled 覆盖，并做失败到通过的回归验证。

用户尚未确认视觉效果。旧记录缺冻结快照时显示不可还原，不能用今天素材伪造当时题面。看义选字、写一写、拼音、算数、总览以及 App 统一接题目后台，均未在本轮完成。
