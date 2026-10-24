# 新增 mod_ai_cache / mod_traffic_mirror / mod_ai_intent 三个 Reloader —— 变更摘要

日期：2026-09-26

## 背景

控制面 `ai-gateway-api` 已陆续导出三个模块的数据面配置，但 conf-agent 样例配置
`conf/conf-agent.toml` 未登记对应 Reloader，配置无法下发到 BFE：

| 模块 | InnerAPI 端点 | 产物文件 | BFE reload API |
|------|---------------|----------|----------------|
| `mod_ai_cache` | `/inner-api/v1/configs/ai-cache-rule` | `ai_cache.data` | `/reload/mod_ai_cache` |
| `mod_traffic_mirror` | `/inner-api/v1/configs/traffic-mirror-rule` | `mirror_rule.data` | `/reload/mod_traffic_mirror` |
| `mod_ai_intent` | `/inner-api/v1/configs/mod-ai-intent` | `intent_questions.data` | `/reload/mod_ai_intent` |

其中 `mod_ai_intent` 是本次语义路由新链路（BFE `7e482d90`+ / ai-gateway-api
`e56273e`）的最后一环。

## 变更内容

`conf/conf-agent.toml` 的 `[Reloaders]` 下新增三个段落，全部为标准
`NormalFileTasks` 形态，逐字段仿 `[Reloaders.mod_ai_route]` 先例：

- `BFEReloadAPI` 显式声明（缺省即 `/reload/{name}`，显式便于核对）；
- `ReloadFile` = 数据文件（变更触发 BFE 热更的文件）；
- `CopyFiles` = 数据文件 + 模块静态 conf（静态项随版本目录下发，与控制面
  "静态配置不导出、经 CopyFiles 下发"的约定一致）；
- `NormalFileTasks`：ConfAPI + ConfFileName 一一映射。

**零代码改动**：Reloaders 为纯配置驱动（`config/config_file.go` 的 map 结构 +
缺省约定），新增标准任务类型不需要改 `config/` 或 `conf_reload/`。

## 兼容性

- 纯增量：既有 Reloader 不受影响；
- 未部署对应 BFE 模块时：reload API 404 的行为与既有模块一致（trigger 按
  既有错误处理路径处理）；
- 无 `mod_ai_*` 目录的存量 BFE conf 目录：首次下发时由 file_store 创建版本
  目录与符号链接（与既有模块同路径）。

## 测试

- TOML 语法校验通过（tomllib 解析，10 个 Reloader 键齐全）；
- `go test ./...` 全绿（无代码变更，验证无回归）。

## 关联

- ai-gateway-api：`e56273e`（intent-config 资源与导出契约，见
  `design-docs/api-define/InnerAPI接口定义/mod-ai-intent.md`）
- BFE：`7e482d90` / `33d1d735`（mod_ai_intent 模块与空 questions 软开关）
