# 成语小园

iPad 横屏优先的成语孩子端，只访问同源 `/api/v1`，由 Nginx 转发到 `chengyu-server:19181`。

```bash
npm install
npm run dev   # http://localhost:19182
```

首页四张卡片会创建真实题单并进入练习。听释义播放已保存的成语读音 MP3，不使用浏览器朗读。默认孩子编号为 `1`。
