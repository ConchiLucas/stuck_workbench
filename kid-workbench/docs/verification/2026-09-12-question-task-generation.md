# 题目后台开发验收记录

日期：2026-09-12。状态：识字首版及默认关闭的自动复习已完成并部署。下列证据均来自实际测试。

## 范围与实现

题目后台负责识字两型普通任务、按真实错选生成复习、素材冻结消费和发布。孩子进度与掌握度仍走已有学习事务。本次不扩展其他学科或新增学习总览。

实际文件组织沿用既有服务结构：素材在 `content-admin/backend/internal/literacy/materials.go`；任务生命周期、素材客户端、复习与worker集中在 `task-admin/backend/internal/taskgen/`，纯生成在 `internal/generation/`；孩子领取在 `literacy-server/internal/plan/tasks.go`，快照判题在 `internal/practice/version.go`。前端入口为 `GenerationPage.tsx` 与孩子端 `TasksPage.tsx`。没有为了匹配计划文件名重复建立新服务层。

## 数据保护与迁移

- 部署前备份：`/tmp/kid-workbench-before-question-tasks-20260912.dump`，已成功用 pg_dump 创建并恢复。
- 独立验收数据库：`study_workbench_qtask_test_20260912`，位于现有 Docker PostgreSQL。
- 测试端口：素材29091、题目29201、识字API29151、历史后台29081。与正式数据库隔离。
- 真实PostgreSQL迁移013成功，旧 `question_id` 放宽可空，增加版本关联和作答回执；重复启动无迁移失败。
- 新增 `study_plan_task_claims` 绑定续练的所有claimKey；单个study_plans.task_claim_key无法保证完成后旧续练键仍回放原计划。
- 素材更新使用独立持久化的 `material_epoch/material_pending`，避免对象写入成功而数据库回滚导致旧sourceRevision仍可使用。

## 已通过的自动化检查

- content-admin：全量Go及race测试（含真实JPEG兼容）。
- task-admin：生成精确配比、100个种子不变量、幂等、版本冲突、换题/排序/发布、复习来源去重、旧题导入、worker退避/重试/租约回收。
- literacy-server：全量Go及race测试；严格快照、选项映射、定向领取、撤回、继续/再练、claim别名幂等、receipt失败整事务回滚。
- parent-dashboard：全量Go测试；013保留旧行、新旧题历史混合读取、新题不回退兼容字段。
- diagnosis-admin：全量Go测试；新回执补齐题型与实际错选。
- shared-go：修复3处测试固定日期过期的问题，仅使用相对当前UTC测试时钟；未改掌握度算法，全量通过。
- 前端题目后台与识字端测试、TypeScript检查和生产构建通过；最终复跑题目后台7项、识字端18项测试全部通过。

## 已发现并修复的问题

1. 复习草稿改标题不应清空题包；规格修改明确拒绝。
2. 复习排序必须保留逐题来源版本。
3. 重生成抽到旧组合后应有界重抽，不能误报组合耗尽；最终seed可重放。
4. 跨组复习截断80候选前保留目标与实际错选项。
5. 旧素材的PNG路径存在实际JPEG内容，冻结需按真实字节识别，媒体类型与字节一致。
6. 旧题后台reshuffle不能修改新素材任务。

## 跨服务实测证据

| 验证对象 | 实际结果 |
| --- | --- |
| 普通题包 | task3，revision1→2→3，8题4+4；换题、排序、发布成功 |
| 领取 | 测试child2；4个同key并发及4个不同key续练均返回plan87 |
| 作答 | plan87完成8题、10次作答；receipt1–10、attempt87–96；两个知识点先错后对 |
| 幂等 | 重放返回原响应；同client换题409，计数不增加；完成后旧续练键仍返回原计划 |
| 复习 | task4/revision4，4题（2原题+2变式），sourceVersion18/17；手动与自动返回同一任务 |
| 复习作答 | plan89完成；receipt11–14、attempt97–100；跨孩子及撤回后新领404；已领可继续 |
| 自动复习 | 2026-09-12 18:02扫描job1，仅创建草稿task4；人工发布；复习完成未递归建job |
| 并发编辑 | task5两个相同expectedRowVersion请求，一个成功到version2、另一个409 |
| 历史兼容 | parent Detail/Review各返回8题；diagnosis保留kp1/kp2实际错选与skill |
| 真实素材 | g1全部10字冻结；PNG/JPEG/MP3实际SHA与MIME一致，ETag304；并发冻结同结果；旧source409 |

浏览器实测：1440宽后台题包、390宽表单、1024×768孩子横屏均可操作；后台试答反馈正确；孩子端从任务列表继续plan88，答对后刷新自动进入下一题，图片保持冻结版本。所检查页面没有控制台error。后台界面未增加学习总览。

部署已更新 backend、content-admin、task-admin、literacy-server、literacy-app、diagnosis-admin；五个后端健康检查healthy，孩子前端running。原 `http://localhost:19201/` 已显示“题目后台”；正式g1候选10/10就绪。正式数据库迁移013已确认，未写验收作答。自动复习正式配置保持false。

最终全量命令：六个Go项目分别 `go test ./...` 全部退出0；两个前端 `npm test && npm run build` 全部退出0。content-admin既有httptest需要监听端口，受控权限复跑通过；初次沙箱拒绝监听不是业务失败。

独立验收库及备份保留；所有临时验收服务已停止，避免留下后台扫描。关键JSON结果保存在本目录 `question-task-generation-evidence/`。

## 回退

自动复习默认关闭。回退只关闭新入口/扫描，保留新列、修订、媒体、回执；已有新格式计划须继续使用支持快照的学习与历史读取服务。不会删除孩子账本或回滚数据库表结构。


## 与原实施步骤的调整

- 实际图片兼容PNG与JPEG并保留原字节，类型按内容判断；像素上限20MP、输入图片10MiB、音频20MiB。没有转换原素材或写回共享旧对象。
- 网络超时、媒体覆盖稳定、失败回滚等破坏性场景使用隔离fixture回归测试；没有覆盖实际共享素材来验证，以免影响其他服务。
- 浏览器采用实际人工操作验收，HTTP并发与复习闭环使用独立库脚本；未新增全套跨浏览器自动化套件或实际断网设备测试。
- 当前工作区存在大量用户原有暂存改动，代码保留在当前工作区供调整，未混合提交、推送或重置。
