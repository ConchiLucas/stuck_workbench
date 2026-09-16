# 拼音音节真人录音

这 16 个文件取自 [hugolpz/audio-cmn](https://github.com/hugolpz/audio-cmn) 的 `64k/syllabs/`，录音作者 **Chen Wang**，上游标注许可 **CC-by-sa**。文件未经修改；每个文件的来源 URL 与 SHA-256 见 `sources.json`。派生或再分发须保留署名并遵循同一许可。

只包含当前音节素材所需的 16 个带调单音节。Docker 镜像保留此目录，素材页“导入音节真人包”将录音写入 MinIO 并保存带内容哈希的地址。不会调用语音合成，不会写入孩子学习记录。
