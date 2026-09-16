# 2026-09-12 按菜单推进：本轮调研计划

本轮目标：阅读现有实现，向用户提出逐菜单统一四端及后续 AI 协作的建议。本轮不实施业务改造，不部署。

根目录 task_plan.md、findings.md、progress.md 属于历史任务，保留；本次调研记录集中于此。

- [x] 阅读用户最新边界、根 AGENTS.md、已有计划与验收记录。
- [x] 分别安排 App、素材/题目后台、进度后台的只读审阅。
- [x] 汇总实际菜单、已存在的复用基础和未统一之处。
- [x] 输出推进讨论稿，区分已明确要求、建议和待验收事实。
- [x] 检查文档范围、链接、历史规则冲突与协作可执行性。

本轮不设计新的视觉稿；讨论对象是菜单划分、实施次序和验收方法。具体页面效果随首个识字样板由用户确认。

错误记录：初次猜测的 parent-dashboard/backend/internal/seed/subjects.go 和 frontend/src/components/QuestionPreview.tsx 不存在；随后用 rg --files 定位目录文件，由只读审阅确认真实组件为 components/task/ReviewItemCard.tsx。错误查询未产生文件改动。
