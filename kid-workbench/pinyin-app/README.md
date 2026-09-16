# 拼音孩子端

iPad 横屏优先的拼音学习 PWA，只访问同源 `/api/v1`，由 Nginx 转发到 `pinyin-server:19111`。

```bash
npm install
npm run dev   # http://localhost:19112
```

默认孩子编号为 `1`。素材由素材后台维护，孩子端不调用素材后台或进度后台 API。
