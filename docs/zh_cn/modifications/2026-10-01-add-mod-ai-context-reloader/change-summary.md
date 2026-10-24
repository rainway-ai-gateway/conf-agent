# 新增 mod_ai_context Reloader —— 变更摘要

日期：2026-10-01

## 背景

控制面 `ai-gateway-api` 已完成 `mod_ai_context` 上下文压缩与裁剪模块的控制面
（`ai_context_rules` 集合资源 + `ai_context_settings` 全局设置单例，导出端点
`/inner-api/v1/configs/ai-context-rule`，产物 `context_rule.data`，顶层
`Defaults` 块恒导出，见 ai-gateway-api `b19c619`）；BFE 数据面模块与 reload
端点 `/reload/mod_ai_context` 也已就绪（`bf441cb5`）。conf-agent 样例配置
`conf/conf-agent.toml` 尚未登记对应 Reloader，规则文件无法下发到 BFE——
本变更是该链路的最后一环。

| 模块 | InnerAPI 端点 | 产物文件 | BFE reload API |
|------|---------------|----------|----------------|
| `mod_ai_context` | `/inner-api/v1/configs/ai-context-rule` | `context_rule.data` | `/reload/mod_ai_context` |

## 变更内容

`conf/conf-agent.toml` 的 `[Reloaders]` 下新增一个段落，为标准
`NormalFileTasks` 形态，逐字段仿 `[Reloaders.mod_ai_cache]` 先例：

- `BFEReloadAPI = "/reload/mod_ai_context"` 显式声明（缺省即 `/reload/{name}`，显式便于核对）；
- `ReloadFile = "context_rule.data"`（数据文件，变更触发 BFE 热更；BFE 侧规则加载器对该文件热加载，顶层 `Defaults` 块随文件版本流更新）；
- `CopyFiles = ["context_rule.data", "mod_ai_context.conf"]`：数据文件 + 模块静态 conf（INI 仅含 `ProductRulePath`/`OpenDebug`，随版本目录下发，与控制面"静态配置不导出、经 CopyFiles 下发"的约定一致）；
- `NormalFileTasks`：`ConfAPI` + `ConfFileName` 一一映射。

**零代码改动**：Reloaders 为纯配置驱动（`config/config_file.go` 的 map 结构 +
缺省约定），新增标准任务类型不需要改 `config/` 或 `conf_reload/`。

## 兼容性

- 纯增量：既有 Reloader 不受影响；
- 未部署 `mod_ai_context` 的 BFE：reload API 404 走 trigger 既有错误处理路径（与既有模块一致）；BFE 侧 conf 样例默认规则 `mode="off"`，模块加载后无压缩行为，可先部署模块再灰度开规则；
- 无 `mod_ai_context` 目录的存量 BFE conf 目录：首次下发时由 file_store 创建版本目录与符号链接（与既有模块同路径）；
- 版本流语义：settings 变更与 rules 变更都使导出 MD5 变化 → 新 `Version` → conf-agent 正常拉取热加载，`Defaults` 块与规则数组同文件原子切换，无部分生效窗口。

## 测试

- TOML 语法校验通过（tomllib 解析，Reloader 键齐全）；
- `go test ./...` 全绿（无代码变更，验证无回归）。

## 关联

- ai-gateway-api：`b19c619`（ai-context 控制面，见
  `design-docs/api-define/InnerAPI接口定义/ai-context-rules.md`、
  `ai-context-settings.md`、`InnerAPI接口定义/ai-context-rule.md`）
- BFE：`bf441cb5`（mod_ai_context 模块一期，见
  `docs/zh_cn/modifications/2026-10-01-ai-context-compress/design-changes.md`）；
  SC24 集成测试（10 个 TC）含规则热加载用例
- 后续登记点：`ai-gateway/kubernetes/deploy/bfe-configmap.yaml` 的
  conf-agent.toml 段同步本配置；bfe.conf `Modules=` 加 `mod_ai_context`
