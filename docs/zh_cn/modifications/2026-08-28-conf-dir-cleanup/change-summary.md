# conf-agent 配置目录过期清理机制优化摘要

## 1. 背景

当前 `conf-agent` 在每次配置变更时，会创建以版本号为后缀的临时配置目录（如 `mod_ai_token_auth_20260730144012`），并通过符号链接 `mod_ai_token_auth` 指向最新目录。但过期目录的清理机制不够健壮，长期运行后会在 `/home/work/bfe/conf` 下堆积大量历史版本目录（[rainway-ai-gateway/conf-agent#8](https://github.com/rainway-ai-gateway/conf-agent/issues/8) 示例中 `mod_ai_token_auth*` 相关目录已达 869 个）。

同样的问题也存在于 `tls_conf` 等其他配置目录，只是配置变更频率较低，问题不明显。

## 2. 目标

- 在每次成功切换正式配置后，主动扫描并清理过期的历史版本目录。
- 保留最近 N 个版本作为安全缓冲；N 通过配置项 `VersionKeepCount` 指定，默认 2。
- 增强符号链接异常场景（损坏、缺失、非链接）的容错能力。
- 修复相关日志误报问题。

## 3. 范围

| 范围 | 说明 |
|------|------|
| 涉及仓库 | `conf-agent` |
| 涉及模块 | `conf_reload/file_store`、`conf_reload`、`config` |
| 变更类型 | 配置落盘与清理机制优化 |
| 接口契约 | 无 |
| 数据迁移 | 无 |

## 4. 关键决策

| 决策 | 说明 |
|------|------|
| 扫描式清理 | 不再只依赖“切换时删除上一个目标目录”，改为扫描父目录，批量清理所有过期版本目录。 |
| 保留版本数可配置 | 通过 `VersionKeepCount` 指定保留数量，默认 2；Reloader 可覆盖 Basic 的默认值。 |
| 版本目录标识 | 在临时目录中写入 `.conf-agent-version` 标识文件，避免误删用户自定义目录。 |
| 清理时机 | 在 `UpdateDefaultConfDir` 成功建立新符号链接后执行，确保 BFE 已切换到新配置。 |
| 错误可观测 | 清理失败只记录错误，不中断主流程；新增清理数量统计日志。 |

## 5. 改动点

| 文件 | 修改内容 |
|------|----------|
| `conf_reload/file_store/file_store.go` | 新增 `cleanupOldVersions` 方法；在 `StoreFile2TmpDir` 中写入 `.conf-agent-version`；在 `UpdateDefaultConfDir` 成功后按 `VersionKeepCount` 调用清理。 |
| `conf_reload/reloader.go` | 修复 `UpdateDefaultConfDir` 失败时仍打印 `succ` 日志的问题；透传 `VersionKeepCount`。 |
| `config/config.go`、`config/config_file.go` | 新增 `VersionKeepCount` 配置项（Basic 默认值 2，Reloader 可覆盖）。 |
| `conf_reload/file_store/file_store_test.go`（新增） | 补充单元测试。 |

## 6. 影响面

| 项目 | 说明 |
|------|------|
| 磁盘占用 | 过期配置目录会被及时清理，避免无限堆积。 |
| 回滚能力 | 默认保留最近 2 个版本；可通过 `VersionKeepCount` 调整，便于按需回滚。 |
| BFE 稳定性 | 清理在符号链接切换成功后执行，不影响 BFE 读取当前配置。 |
| 兼容性 | 不修改配置格式、API 接口和 BFE 交互方式。 |
