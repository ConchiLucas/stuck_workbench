# 科普儿童端

iPad 横屏优先的独立科普探索 PWA，默认开发端口 `19122`。应用只访问同源 `/api/v1`，生产环境由 Nginx 转发到 `science-server:19121`；不包含数据库、MinIO、素材后台或进度后台连接信息。

```bash
npm install
npm test -- --run
npm run build
npm run dev
```

默认孩子编号为 `1`。儿童端只展示素材后台审核发布后的知识卡和题目。
