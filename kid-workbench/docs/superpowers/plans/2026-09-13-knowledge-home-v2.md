# 孩子知识库图形首页实施计划

用户已批准 V2 设计并要求修改代码。使用 subagent-driven-development 分离后端统计与前端布局，最后统一检查与部署。

目标：在原首页上线九学科积累入口、能力状态、错题分布、复习状态；明细保留在菜单中。

技术：React 19 / TypeScript / CSS，Go / GORM，共享只读学习库。沿用 Summary 增加各科汇总；复习阶段使用独立只读统计接口，失败不影响知识图。当前目录有大量既有未提交实现，直接在用户指定目录修改，保留基线备份 /tmp/knowledge-home-v2-baseline/diagnosis-admin.tgz，不切换分支或覆盖其他服务。

- [x] 后端：先写真实数据测试，再扩展 knowledge.Subject 为 wrongCount、abilities（mastered/learning/shaky/unpracticed/unknown），仅统计已练知识点的适用能力，未知不伪装为未练。没有技能拆分的学科明确 unknown。review_due 归待巩固；保留历史事实与原 Summary 字段语义。
- [x] 复习统计：新增 GET /knowledge/review-status-summary，全量按本孩子未归档建议去重。字段 pending/draft/awaiting/answered/total；优先真实关联作答，其次已发布，其次已生成题包，其他待生成。查询失败返回错误，表未安装返回不可用，不伪造零值；匹配建议链接的任务版本与本孩子实际计划/回执。不能更新学习表。
- [x] 前端：新增 KnowledgeOverviewPage.tsx、overview.css、图表与菜单测试；首页调用 Summary 和独立复习统计，四种能力状态外兼容未知；真实零值与请求错误分开。九学科入口和图表行进入 /subjects/:code 与 /wrongs?subject=code；复习图进入 /reviews。图表无题目明细或分析文字。
- [x] 导航：AppShell 改为左侧四菜单；新增 /library 作为原知识档案，保留 /subjects、/knowledge-points、/attempts 和 /reviews 链接；修复所有返回知识档案的链接。首页采用紫色工作台、绿/蓝/橙/灰能力色，详细页保留现有可读内容。移动端菜单横排、图表单列。
- [x] 验证：知识统计包含未练、未知、旧账本、错误去重；复习包含多任务、版本不符、其他孩子、归档、零记录。前端验证首页无明细、链接正确、错误不是0；运行后端 go test ./... 与前端 npm test、npm run build。
- [x] 发布：记录旧镜像，只执行 docker compose build diagnosis-admin 与 docker compose up -d --no-deps diagnosis-admin；浏览器检查桌面与手机、原接口与新接口。健康后只清理该服务已替换的镜像和本轮预览进程，更新设计文档与完成记录。

全量统计来自服务器，不在浏览器抓取分页近似。四阶段代表建议当前最远可核验状态，不表示全部题目已完成或已经掌握。阶段混合时优先 answered > awaiting > draft > pending，全部样例数据不得进入应用。
