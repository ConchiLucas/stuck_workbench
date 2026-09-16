# 拼音 App 隔离浏览器测试

先运行 `npm run build`，再运行 `npm run e2e`。

Playwright 使用本地已安装的 WebKit。`fixture.ts` 将 `https://pinyin.test` 的页面、脚本、样式和图片请求全部从 `dist/` 读取并直接 fulfill，正式生成、实例 GET 和作答 POST 全部由内存 API 回答。配置没有 webServer，不启动监听端口，不访问 localhost:19112、正式拼音服务或数据库；未知 API 与外部请求被拒绝并导致测试失败，Service Worker 禁用。

`pinyin-flow.spec.ts` 保留首页、拼音图、旧计划题面和四题型布局检查；`pinyin-mastery-flow.spec.ts` 覆盖固定幂等重试、成功响应丢失后刷新恢复、音节素材 ID 与 KP ID 分离、跳过未提交、重新练习新实例和孩子隔离。

这些测试验证浏览器与接口契约、会话和展示行为。内存 ledger 只用于断言请求次数和 ID 路由，不代表数据库事务或掌握算法验证；后两者由 Go 测试与隔离数据库集成测试负责。
