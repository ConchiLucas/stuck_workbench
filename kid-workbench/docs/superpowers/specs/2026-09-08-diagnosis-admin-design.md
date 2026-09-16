# 诊断台设计

独立只读后台，给人看孩子「会什么、弱在哪」。不写掌握度、不生成任务、不接大模型。

## 边界

- 新项目 `diagnosis-admin/`（Go API 内嵌 React），共用 `study_workbench`。
- 只读 `children`、`subjects`、`modules`、`knowledge_points`、`questions`、`attempts`、`mastery_states`、`mastery_skills`、`study_plans`、`plan_items`。
- 进度后台继续管今日进度和任务列表；诊断台侧栏可链过去，不复制矩阵和加餐。

## 页面

1. 诊断总览：人话结论 + 各学科健康度 + 最急薄弱点
2. 学科诊断：模块与技能缺口
3. 知识点档案：技能、作答时间线、最近错选
4. 错因：技能不平衡、反复选错、超时答错

## 地址

- Docker / 内嵌：http://localhost:19211
- 本地 Vite：http://localhost:19212

## 接口

- `GET /healthz`
- `GET /api/v1/children/:cid/diagnosis/overview`
- `GET /api/v1/children/:cid/diagnosis/subjects/:code`
- `GET /api/v1/children/:cid/knowledge-points/:kpId`
- `GET /api/v1/children/:cid/error-patterns`
