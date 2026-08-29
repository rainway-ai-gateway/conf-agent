# conf-agent file_store 清理模块集成测试设计文档

## 1. 模块概述

`file_store` 是 `conf-agent` 中负责配置持久化与目录管理的模块。它的核心职责包括：

- 从 `ai-gateway-api` 拉取配置内容并写入临时目录；
- 通过 BFE `/reload/{module}` 接口触发热加载；
- 热加载成功后，将配置目录以版本号命名并切换符号链接 `mod_demo` 指向最新版本；
- 根据 `VersionKeepCount` 保留最近 N 个版本目录，清理历史版本。

本集成测试采用**进程内集成**方式，使用 `httptest.Server` 模拟上游服务，直接驱动 `conf_reload.Reloader` 运行，验证 `file_store` 的完整 reload 流程与清理行为。

## 2. 接口列表

| 编号 | 接口名称 | 方法 | 路径 | 说明 |
|------|----------|------|------|------|
| CA-FS-1 | 获取模块配置 | GET | `/configs/{module}` | 模拟 `ai-gateway-api` InnerAPI，返回模块配置 JSON |
| CA-FS-2 | 触发 BFE 热加载 | GET | `/reload/{module}` | 模拟 BFE reload 接口，按 path 参数加载配置 |

## 3. 测试用例统计

| 场景 | 测试用例数 |
|------|-----------|
| 正常 reload 与版本清理 | 1 |
| BFE reload 失败回滚 | 1 |
| **合计** | **2** |

## 4. 认证方式

本模块为内部调用，无认证要求。测试中使用 `httptest.Server`，无需 TLS 或 Token。

## 5. 目录结构

```
cleanup/
├── design.md
└── cleanup_test.go
```

## 6. 获取模块配置（CA-FS-1）

### 6.1 接口信息

| 项目 | 值 |
|------|-----|
| 模块 | file_store |
| 接口名称 | 获取模块配置 |
| 方法 | GET |
| 路径 | `/configs/{module}` |
| 说明 | 返回指定模块的配置数据与版本号 |

### 6.2 接口参数说明

#### 6.2.1 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| module | path | Y | 模块名，如 `mod_demo` |
| bfe_cluster | query | Y | BFE 集群名 |
| version | query | N | 当前本地版本号，为空表示首次拉取 |

#### 6.2.2 返回数据字段

| 参数名 | 类型 | 说明 |
|--------|------|------|
| Data | object | 配置内容，具体字段由模块决定 |
| Data.Rules | []string | 示例配置：规则列表 |
| Data.Version | string | 配置版本号，格式建议为 `yyyyMMddHHmmss` |
| ErrNum | int | 200 表示成功 |

## 7. 触发 BFE 热加载（CA-FS-2）

### 7.1 接口信息

| 项目 | 值 |
|------|-----|
| 模块 | file_store / trigger |
| 接口名称 | 触发 BFE 热加载 |
| 方法 | GET |
| 路径 | `/reload/{module}` |
| 说明 | BFE 按 `path` 参数指向的新配置目录进行热加载 |

### 7.2 接口参数说明

#### 7.2.1 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| module | path | Y | 模块名 |
| path | query | Y | 新配置版本目录的绝对路径 |

#### 7.2.2 返回数据字段

| 参数名 | 类型 | 说明 |
|--------|------|------|
| 状态码 | int | 200 表示热加载成功；非 200 表示失败 |
| body | string | 成功返回空或成功提示；失败返回错误信息 |

## 8. 测试场景总览

| 编号 | 场景 | 测试类型 | 简要说明 |
|------|------|---------|---------|
| CA-FS-001 | 正常 reload 与版本清理 | 正常流程 | 三次 reload 后，符号链接指向最新版本，历史版本按 `VersionKeepCount=2` 清理 |
| CA-FS-002 | BFE reload 失败保持当前版本 | 异常流程 | BFE reload 返回 500，符号链接不切换，当前版本目录保留 |

## 9. 测试场景详细设计

### 9.1 CA-FS-001：正常 reload 与版本清理

#### 设计思路

验证 `file_store` 在多次成功 reload 后：

1. 每次都会生成以版本号命名的配置目录；
2. `mod_demo` 符号链接始终指向最新版本目录；
3. 当历史版本数超过 `VersionKeepCount=2` 时，最旧的版本目录被删除。

#### 前提数据准备

- 创建临时测试根目录 `workDir`；
- 配置 `ReloaderConfig`：
  - `Name = "mod_demo"`
  - `RemoteConfUrl` 指向 mock `ai-gateway-api`；
  - `BfeReloadUrl` 指向 mock BFE；
  - `ConfDir = workDir + "/mod_demo"`
  - `VersionKeepCount = 2`
  - `Interval = 300ms`（缩短测试等待时间）

#### 执行步骤

1. 启动 mock `ai-gateway-api`，按请求顺序返回三个版本：
   - `20260101120000`（Rules: `["rule1"]`）
   - `20260101120001`（Rules: `["rule_20260101120001"]`）
   - `20260101120002`（Rules: `["rule_20260101120002"]`）
2. 启动 mock BFE，对所有 reload 请求返回 200。
3. 启动 `Reloader` 并等待第一次 reload 完成。
4. 断言符号链接 `mod_demo` 指向 `mod_demo_20260101120000`。
5. 等待第二次 reload 完成，断言符号链接指向 `mod_demo_20260101120001`。
6. 等待第三次 reload 完成，断言符号链接指向 `mod_demo_20260101120002`。
7. 列出 `workDir` 下 `mod_demo_*` 目录，断言仅剩 2 个（`20260101120001`、`20260101120002`）。
8. 读取当前符号链接目标目录下的配置内容，断言 `Rules` 为最新版本。

#### 请求参数

无显式请求参数，由 `Reloader` 自动轮询生成：

```text
GET /configs/mod_demo?bfe_cluster=default&version=
GET /configs/mod_demo?bfe_cluster=default&version=20260101120000
GET /configs/mod_demo?bfe_cluster=default&version=20260101120001

GET /reload/mod_demo?path=<tmp>/mod_demo_20260101120000
GET /reload/mod_demo?path=<tmp>/mod_demo_20260101120001
GET /reload/mod_demo?path=<tmp>/mod_demo_20260101120002
```

#### 预期返回结果

- 三次 `ai-gateway-api` 均返回 `ErrNum=200`；
- 三次 BFE reload 均返回 200；
- 符号链接最终指向 `mod_demo_20260101120002`；
- `workDir` 下只保留 `mod_demo_20260101120001`、`mod_demo_20260101120002` 两个版本目录；
- 最新版本目录下的配置 `Rules` 为 `["rule_20260101120002"]`。

#### 断言校验

| 检查项 | 预期值 | 校验方式 |
|--------|--------|---------|
| 第一次符号链接目标 | `mod_demo_20260101120000` | Equals |
| 第二次符号链接目标 | `mod_demo_20260101120001` | Equals |
| 第三次符号链接目标 | `mod_demo_20260101120002` | Equals |
| 保留版本目录数 | 2 | Len=2 |
| 被清理版本 | `mod_demo_20260101120000` 不存在 | NotExists |
| 最新配置 Rules | `["rule_20260101120002"]` | Equals |

---

### 9.2 CA-FS-002：BFE reload 失败保持当前版本

#### 设计思路

验证当 BFE `/reload/{module}` 返回失败时，`file_store` 不会切换符号链接，当前生效版本保持不变。

#### 前提数据准备

- 创建临时测试根目录 `workDir`；
- 配置 `ReloaderConfig`，`VersionKeepCount = 2`，`Interval = 300ms`；
- mock BFE 对所有 reload 请求返回 500。

#### 执行步骤

1. 启动 mock `ai-gateway-api`，返回版本 `20260101120000`。
2. 启动 mock BFE，对所有 `/reload/mod_demo` 请求返回 500。
3. 启动 `Reloader`，等待至少 2 个轮询周期。
4. 断言符号链接 `mod_demo` 不存在或仍指向原始目标（因为 `UpdateDefaultConfDir` 未被调用）。
5. 断言没有生成新的 `mod_demo_*` 版本目录被符号链接引用。

#### 请求参数

```text
GET /configs/mod_demo?bfe_cluster=default&version=
GET /reload/mod_demo?path=<tmp>/mod_demo_20260101120000
```

#### 预期返回结果

- `ai-gateway-api` 返回 `ErrNum=200`；
- BFE reload 返回 500；
- `Reloader` 记录 `TriggerBFEReload fail`；
- 符号链接 `mod_demo` 未指向新版本目录。

#### 断言校验

| 检查项 | 预期值 | 校验方式 |
|--------|--------|---------|
| BFE reload 调用次数 | ≥1 | ≥ |
| 符号链接目标 | 不存在或不是 `mod_demo_20260101120000` | NotEquals |
| 版本目录是否被链接 | 无 | 符号链接不存在 |

---

## 10. 依赖与数据准备

1. 测试模块 `github.com/rainway-ai-gateway/conf-agent/integration` 通过 `replace` 引用本地 `conf-agent` 根模块。
2. 运行前执行 `go mod tidy` 确保依赖完整。
3. 每个测试用例使用 `t.TempDir()` 生成独立临时目录，测试结束后自动清理。

## 11. 注意事项

1. 测试在 Windows 与 Linux 下均需通过，符号链接检查应使用 `os.Readlink` 或平台无关封装。
2. `Reloader` 为轮询 goroutine，测试通过 `runner.WaitForReload` 监听日志中的 `reload succ` 或 `TriggerBFEReload fail` 来判断状态变化。
3. `Agent.Stop()` 与 `Reloader.Stop()` 应可重入，避免测试清理阶段阻塞。
4. mock BFE 的 500 响应应包含可读错误体，便于日志中定位问题。
