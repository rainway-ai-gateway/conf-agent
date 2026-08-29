# conf-agent 配置目录过期清理机制优化设计

## 1. 概述

### 1.1 变更背景

`conf-agent` 每次从 `ai-gateway-api` 拉取到新配置后，会：

1. 创建临时目录 `ConfDir_<version>`；
2. 将配置写入该目录；
3. 触发 BFE 热加载；
4. 将正式配置路径 `ConfDir` 的符号链接指向新的临时目录。

当前实现（`conf_reload/file_store/file_store.go:UpdateDefaultConfDir`）在切换符号链接时，只尝试删除上一个符号链接目标目录。该机制存在以下问题：

- 只能清理“上一个”版本目录，历史上因进程异常、切换失败等原因遗留的目录永远不会被清理；
- 对符号链接损坏、缺失、被替换为普通目录等异常场景缺乏容错；
- 没有显式标识哪些目录是 `conf-agent` 生成的，存在误删风险；
- `conf_reload/reloader.go` 在 `UpdateDefaultConfDir` 失败时仍打印成功日志，掩盖问题。

### 1.2 变更目标

1. 建立可识别的版本目录标记机制。
2. 每次成功切换配置后，扫描并清理所有过期版本目录。
3. 保留最近 N 个版本作为安全缓冲；N 通过配置项 `VersionKeepCount` 指定，默认值为 2，兼顾回滚与 BFE 句柄安全。
4. 提升符号链接异常场景的容错性。
5. 修复日志误报。

### 1.3 变更范围

- `conf_reload/file_store/file_store.go`
- `conf_reload/reloader.go`
- `config/config.go`、`config/config_file.go`
- 新增 `conf_reload/file_store/file_store_test.go`

---

## 2. 当前机制详细分析

### 2.1 目录创建

`StoreFile2TmpDir` 根据版本号创建临时目录：

```go
func (fileStore *FileStore) tmpDir(version string) string {
    return fileStore.ConfDir + "_" + version
}
```

例如 `ConfDir=/home/work/bfe/conf/mod_ai_token_auth`，`version=20260730144012`，则临时目录为：

```text
/home/work/bfe/conf/mod_ai_token_auth_20260730144012
```

### 2.2 符号链接切换

`UpdateDefaultConfDir` 当前逻辑：

```go
dest, err := filepath.EvalSymlinks(fileStore.ConfDir)
if err != nil {
    return err
}

// 删除符号链接本身
os.RemoveAll(fileStore.ConfDir)

// 如果原目标是链接，再删除链接指向的真实目录
if dest != fileStore.ConfDir {
    os.RemoveAll(dest)
}

// 建立新符号链接
xfile.FileLink(fileStore.tmpDir(version), fileStore.ConfDir)
```

理论上每次只会保留当前激活目录。但生产环境中观察到大量历史目录堆积，说明当前“单版本回退”清理策略无法覆盖所有异常场景。

### 2.3 堆积原因

| 场景 | 说明 |
|------|------|
| 进程异常退出 | 在 `StoreFile2TmpDir` 之后、`UpdateDefaultConfDir` 之前退出，临时目录未被清理，后续版本目录持续增加。 |
| 切换失败 | `UpdateDefaultConfDir` 失败（如符号链接被占用、权限问题）后未重试，历史目录持续堆积。 |
| 多版本并发 | 若存在多个 `conf-agent` 实例或手动干预，符号链接目标与目录集合可能不一致。 |
| 无标识过滤 | 无法区分 `conf-agent` 生成的目录与用户手动创建的同名目录，不敢做全量清理。 |

---

## 3. 优化方案

### 3.1 核心思路

在临时目录中写入 `.conf-agent-version` 标识文件，使过期目录可被安全识别；在符号链接切换成功后，扫描父目录中所有带标识的目录，保留最近 `VersionKeepCount` 个版本（默认 2），删除其余目录。

### 3.2 版本目录标识

在 `StoreFile2TmpDir` 创建临时目录后，写入 `.conf-agent-version`：

```go
markerFile := filepath.Join(tmpDir, ".conf-agent-version")
if err := os.WriteFile(markerFile, []byte(version), 0644); err != nil {
    return fmt.Errorf("write version marker fail, file: %s, err: %v", markerFile, err)
}
```

该文件仅用于识别目录归属，不影响 BFE 读取配置。

### 3.3 清理策略

```go
func (fileStore *FileStore) cleanupOldVersions(ctx context.Context, keep int) error
```

实现步骤：

1. 解析当前 `ConfDir` 的符号链接目标 `currentTarget`；
2. 遍历 `ConfDir` 的父目录；
3. 筛选出同时满足以下条件的目录：
   - 名称前缀为 `filepath.Base(ConfDir) + "_"`；
   - 目录内存在 `.conf-agent-version` 文件；
   - 不是当前符号链接目标；
4. 按目录修改时间倒序排序；
5. 保留前 `VersionKeepCount` 个，删除其余；
6. 记录删除数量和错误。

`VersionKeepCount` 的来源：

- 优先读取当前 `Reloader` 的 `VersionKeepCount`；
- 若 `Reloader` 未设置，则继承 `Basic.VersionKeepCount`；
- 若均未设置，使用默认值 `2`。

### 3.4 符号链接异常处理

增强 `UpdateDefaultConfDir`：

| 场景 | 处理 |
|------|------|
| `ConfDir` 是正常符号链接 | 按现有逻辑删除旧链接和目标，建立新链接。 |
| `ConfDir` 是普通目录 | 重命名为 `ConfDir_<timestamp>.backup` 并保留，建立新符号链接。 |
| `ConfDir` 不存在 | 直接建立新符号链接。 |
| `ConfDir` 是损坏的符号链接 | 删除损坏链接，不删除目标（不存在），建立新链接。 |

> 普通目录重命名备份而非直接删除，避免误删用户已有配置。

### 3.5 日志修复

`conf_reload/reloader.go` 当前：

```go
err = r.fileStore.UpdateDefaultConfDir(ctx, version)
if err != nil {
    xlog.Default.Error(xlog.ErrLogFormat(ctx, "UpdateDefaultConfDir fail", err))
}
xlog.Default.Info(xlog.InfoLogFormat(ctx, "UpdateDefaultConfDir succ"))
```

改为仅在无错误时打印成功日志，失败时不打印：

```go
err = r.fileStore.UpdateDefaultConfDir(ctx, version)
if err != nil {
    xlog.Default.Error(xlog.ErrLogFormat(ctx, "UpdateDefaultConfDir fail", err))
    return
}
xlog.Default.Info(xlog.InfoLogFormat(ctx, "UpdateDefaultConfDir succ"))
```

---

## 4. 接口与配置变更

### 4.1 新增配置项

| 配置项 | 所在位置 | 类型 | 必填 | 默认值 | 说明 |
|--------|----------|------|------|--------|------|
| `VersionKeepCount` | `Basic` | int | N | 2 | 全局默认保留的历史版本目录数量。 |
| `VersionKeepCount` | `Reloader` | int | N | 同 `Basic.VersionKeepCount` | 单个 Reloader 的保留数量，未设置时继承 Basic。 |

### 4.2 约束

- 最小值为 `1`，即至少保留当前激活版本；
- 配置校验失败（如小于 1）时，`conf-agent` 启动失败，避免清理逻辑异常。

---

## 5. 实现步骤

1. 在 `file_store.go` 中新增 `writeVersionMarker` 和 `cleanupOldVersions` 方法。
2. 修改 `StoreFile2TmpDir`，在创建临时目录后写入 `.conf-agent-version`。
3. 修改 `UpdateDefaultConfDir`，在建立新符号链接后调用 `cleanupOldVersions(ctx, versionKeepCount)`，`versionKeepCount` 取自 Reloader 配置或 Basic 默认值。
4. 增强 `UpdateDefaultConfDir` 对普通目录、缺失目录、损坏链接的处理。
5. 修复 `reloader.go` 的成功日志误报。
6. 新增 `file_store_test.go`，覆盖清理逻辑和异常场景。

---

## 6. 测试计划

### 6.1 单元测试

1. `cleanupOldVersions`：创建多个版本目录，按 `VersionKeepCount` 保留，删除其余。
2. `cleanupOldVersions`：忽略不带 `.conf-agent-version` 的目录。
4. `UpdateDefaultConfDir`：`ConfDir` 为符号链接时，旧目标被删除，新链接正确建立。
5. `UpdateDefaultConfDir`：`ConfDir` 为普通目录时，被重命名为备份目录。
6. `UpdateDefaultConfDir`：`ConfDir` 不存在时，直接建立新符号链接。
7. `UpdateDefaultConfDir`：损坏的符号链接被删除并重建。

### 6.2 集成测试

1. 模拟一次完整配置 reload，验证旧版本目录被清理。
2. 连续多次配置变更，验证按 `VersionKeepCount` 保留对应数量的版本目录（默认 2 个）。
3. 人为创建无标识目录，验证不会被误删。

---

## 7. 风险与缓解

| 风险 | 说明 | 缓解措施 |
|------|------|----------|
| 误删用户目录 | 父目录中可能存在用户手动创建的目录 | 通过 `.conf-agent-version` 标识和名称前缀双重校验 |
| 清理时 BFE 仍持有旧目录句柄 | BFE 可能尚未释放旧配置文件 | 通过 `VersionKeepCount` 保留多个版本，默认 2 个 |
| 权限不足 | 无法删除某些历史目录 | 记录错误日志，不中断主流程 |
| 符号链接操作原子性 | 切换符号链接非原子操作 | 先创建新链接到临时名，再原子 rename（可选优化） |

---

## 8. 兼容性

- 不修改配置文件的格式和内容。
- 不修改 `conf-agent` 与 `ai-gateway-api`、BFE 的交互协议。
- 新增的 `.conf-agent-version` 文件对 BFE 透明。
