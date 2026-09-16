# 拼音掌握改造验收记录

2026-09-12；对应方案：`2026-09-12-pinyin-mastery.md`。

## 已交付

- 进度后台原地址：<http://localhost:19081/subjects/pinyin>。
- 四种题型进度全部展开；字母按「听、找、认」三项判断，标记只有亮暗；音节独立按拼读判断。
- 字母与带调音节分组地图、题型视图、拼音搜索、悬浮信息、只读详情；默认不显示统计数字。
- App 四题型正式出题、冻结快照、服务端判分、事务回执、重试和刷新恢复；孩子之间的会话及题号隔离。
- 音节素材稳定映射、停用目录处理、模块技能校验、首次掌握里程碑、一次性奖励、旧规则升级审计。
- 总览与 App 使用同一孩子的实际进度；旧计划练习保持兼容。

## 实现范围与调整

- `shared-go`：mastery、model、learning、pinyincontract、pinyincatalog。
- `pinyin-server`：quiz、http、progress、home、catalog 及启动依赖。
- `pinyin-app`：API、正式练习会话、练习页与结果页、隔离测试。
- `parent-dashboard/backend`：015 双数据库迁移、拼音 Matrix/详情/汇总、升级命令、镜像打包与测试。
- `parent-dashboard/frontend`：拼音展示模型、专属组件、SubjectDetail 分支与相关类型。
- 历史测试合并在 progress/upgrade/quiz 测试文件中，没有为每项计划单独创建测试文件。日历继续读取统一日统计；题型、实际选择和快照事实由知识点历史接口提供，不新增另一套逐日详情页面。
- API JSON 沿用各服务已有字段风格，后台使用 snake_case，App 正式契约使用 camelCase。
- 旧未分类拼音作答保留事实，不能作为三技能掌握证据。旧 demo 重算器保留拼音派生状态；遇到本版回执、里程碑或升级审计时，在删除任何统计之前拒绝执行，正式升级使用 `pinyin-upgrade`。
- 正式选项 ID 为实例内 `option-N`；公开拼读题面不携带答案音节。旧演示接口继续兼容，正式 App 不再使用其前端答案判分。

## 验证

| 范围 | 验证与结果 |
|---|---|
| shared-go | `GOCACHE=/tmp/kid-pinyin-go-cache go test ./...` 通过 |
| parent backend | 全量 `go test ./...` 通过；最后新增 demo 重算保护后，`go test ./internal/seed -count=1` 通过 |
| pinyin-server | 全量 `go test ./...`、`go vet ./...` 通过 |
| PostgreSQL | 独立 `kid_pinyin_verify_20260912` 数据库：015迁移、映射、首次同步分母、拼读掌握、重复升级保留日期、并发幂等提交、外键及奖励事务通过 |
| App | `npm test -- --run`：39项通过；`npm run build` 通过；隔离 WebKit E2E：9项通过 |
| 进度前端 | 11项测试、lint、build通过；1280/768/390px地图、搜索和抽屉检查通过 |
| 部署后 | 三端口 HTTP 200；真实拼音页及 bā 详情正常；后台与 App 的61个知识点及其适用技能状态逐一一致 |

浏览器自动作答采用内存路由模拟；真实数据库事务通过独立 PostgreSQL 验证。正式孩子没有被用于测试作答。

## 升级与数据核对

数据库备份：`/tmp/pinyin-before-upgrade-20260912.dump`（318407字节）。

`pinyin-upgrade --dry-run` → `--apply` → 再次 `--dry-run` 均成功：

- 旧拼音派生记录：0；需要改为部分掌握：0；历史奖励受影响：0。
- 当前目录：45个字母、16个带调音节；目录可用。
- 本次没有旧拼音记录需要审计降级，因此规则生效日期为空，页面不虚构日期。

正式账本部署前后数量及所有行 JSON 摘要完全一致：

| 表 | 行数 | 前后相同的 MD5 |
|---|---:|---|
| attempts | 84 | 11f4e9556ed4ab440fc88bd06ae1cf3c |
| daily_stats | 3 | d725dbdcdbb41caf9753955ae98bbee1 |
| flower_ledger | 1 | 9eb5967d9b5835ec502a10ad0167ee0d |

## 部署范围

仅执行指定服务构建、停止及 `up --no-deps --no-build --force-recreate --wait`：

| 服务 | 原端口 | 结果 |
|---|---:|---|
| backend | 19081 | 新镜像替换，健康 |
| pinyin-server | 19111 | 新镜像替换，健康 |
| pinyin-app | 19112 | 新镜像替换，健康 |

全部运行容器部署前后的 ID、镜像和启动时间对比，变化集合严格为以上三项。数据库、识字、素材后台、题目后台、孩子知识库等均未被本轮重启。后台镜像包含同工作区并行识字任务已验证的共享改动，已通知对方。

旧三个容器已替换；按原容器镜像 ID 定向清理时 Docker 均报告镜像不存在，随后检查没有 dangling 镜像，未做全局 prune 或强制删除。原始核验记录保存在 `/tmp/pinyin-containers-before.txt`、`/tmp/pinyin-containers-after.txt`。

没有启动额外预览端口。独立测试数据库已删除；正式数据库及数据卷保留。
