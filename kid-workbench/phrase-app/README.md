# 短句小舞台

iPad 横屏优先的英语短句孩子端，只访问同源 `/api/v1`，由 Nginx 转发到 `phrase-server:19171`。

```bash
npm install
npm run dev   # http://localhost:19172
```

首页四张卡片会创建真实题单并进入练习。有整句 MP3 时播放冻结或素材音频；没有读音时提示暂不可用，不使用浏览器朗读冒充。默认孩子编号为 `1`。

题库变更后请先 `make seed`，再打开孩子端。
