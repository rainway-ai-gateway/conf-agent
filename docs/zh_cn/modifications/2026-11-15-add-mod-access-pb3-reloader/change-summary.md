# 新增 mod_access_pb3 Reloader —— 变更摘要

日期：2026-11-15

## 背景

访问日志 Header 白名单采集（bfenetworks/bfe#1506）的控制面与数据面均已就绪：

- BFE 数据面：`mod_access_pb3` 支持动态配置文件 `mod_access_pb.data`（多段式
  JSON，当前含 `access_header` 白名单段），经 web monitor reload 接口
  `/reload/mod_access_pb3` 热加载（`bfe` 仓库 `v1.10.0-dev` 提交 `c3ddeb18`，
  SC49 集成测试 8/8 通过）；
- 控制面：`ai-gateway-api` 单例资源 `GET/PUT /open-api/v1/access-log/config`
  + 导出端点 `/inner-api/v1/configs/mod-access-pb`（topic `mod_access_pb`，
  数据签名版本、未变更返回 `Data: null`；`ai-gateway-api` 仓库 `v0.2.0-dev`
  提交 `44f676a`）。

conf-agent 样例配置 `conf/conf-agent.toml` 尚未登记对应 Reloader，
`mod_access_pb.data` 无法下发到 BFE——本变更是"控制台改白名单 → 全集群热生效"
链路的最后一环。

| 模块 | InnerAPI 端点 | 产物文件 | BFE reload API |
|------|---------------|----------|----------------|
| `mod_access_pb3` | `/inner-api/v1/configs/mod-access-pb` | `mod_access_pb.data` | `/reload/mod_access_pb3` |

## 变更内容

`conf/conf-agent.toml` 的 `[Reloaders]` 下新增一个段落，为标准
`NormalFileTasks` 形态，逐字段仿 `[Reloaders.mod_ai_route]` 先例：

```toml
# reloader for mod_access_pb3 (access log header whitelists, bfenetworks/bfe#1506)
[Reloaders.mod_access_pb3]
BFEReloadAPI    = "/reload/mod_access_pb3"
ReloadFile      = "mod_access_pb.data"
CopyFiles       = ["mod_access_pb.data", "mod_access_pb3.conf"]
[[Reloaders.mod_access_pb3.NormalFileTasks]]
ConfAPI         = "/inner-api/v1/configs/mod-access-pb"
ConfFileName    = "mod_access_pb.data"
```

- `BFEReloadAPI = "/reload/mod_access_pb3"` 显式声明（BFE 侧 handler 名即模块
  名，显式便于核对）；
- `ReloadFile = "mod_access_pb.data"`（数据文件，变更触发 BFE 热更；BFE 侧
  锁外校验、锁内整份原子替换，校验失败保留旧配置）；
- `CopyFiles = ["mod_access_pb.data", "mod_access_pb3.conf"]`：数据文件 + 模块
  静态 conf（INI 含 `[Log]` 与可选 `DataConfPath`，随版本目录下发，与控制面
  "静态配置不导出、经 CopyFiles 下发"的约定一致）。`DataConfPath` 为可选——
  未配置时 BFE 不启用动态配置（缺省运行），存量 `mod_access_pb3.conf` 无需
  修改即可继续加载；
- `NormalFileTasks`：`ConfAPI` + `ConfFileName` 一一映射；落点为
  `<BFEConfDir>/mod_access_pb3/mod_access_pb.data`，与 BFE 样例
  `DataConfPath = ../conf/mod_access_pb3/mod_access_pb.data` 一致。

**零代码改动**：Reloaders 为纯配置驱动（`config/config_file.go` 的 map 结构
+ 缺省约定），新增标准任务类型不需要改 `config/` 或 `conf_reload/`。

## 兼容性

- 纯增量：既有 Reloader 不受影响；
- 未升级 BFE（无 `/reload/mod_access_pb3` handler）：reload API 走 trigger
  既有错误处理路径（与既有模块一致），不影响其他 Reloader；
- 未配置 `DataConfPath` 的 BFE：不下发数据文件即维持缺省行为（不采集任何
  Header），无交叉影响；
- 版本流语义：白名单变更使导出数据签名变化 → 新 `Version` → conf-agent 拉取
  落盘并触发热加载；空白名单也导出（合法默认态，支持撤销已发布白名单）；
- 数据文件为**整份原子语义**：BFE 侧任一段校验失败则整份 reload 失败、保留
  旧配置（BFE SC49 TC-05 已覆盖），不存在半新半旧窗口。

## 测试

- TOML 语法校验通过（解析后 Reloader 键齐全）；
- `go test ./...` 全绿（无代码变更，验证无回归）。

## 关联

- BFE：`c3ddeb18`（`v1.10.0-dev`；见
  `docs/zh_cn/modifications/2026-11-15-issue-1506-mod-access-pb3-header-whitelist/design-changes.md`、
  `docs/zh_cn/sys_design/mod_access_pb_data_conf.md`，SC49 集成测试 8/8）；
- ai-gateway-api：`44f676a`（`v0.2.0-dev`；见
  `design-docs/modifications/2026-11-15-access-log-header-whitelist/`，
  api-define `access-log.md` / `mod-access-pb.md`，集成测试含导出契约与
  版本单调用例）；
- 后续登记点：`ai-gateway/kubernetes/deploy/bfe-configmap.yaml` 的
  conf-agent.toml 段同步本配置（部署侧变更，随下次部署带）。
