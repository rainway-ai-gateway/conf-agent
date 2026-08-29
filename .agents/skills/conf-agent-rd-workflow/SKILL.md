---
name: conf-agent-rd-workflow
description: 引导用户在 conf-agent 代码库中完成一次完整的功能研发流程，包括需求对齐、设计文档修改、代码实现、单元测试、集成测试与回归验证。
---

# conf-agent 代码库研发流程

本 Skill 适用于在 `conf-agent/` 目录中新增或修改配置拉取、持久化、BFE 热加载、目录版本管理等功能的完整研发流程。

## 触发语

当用户提出以下类型请求时启用本流程：

- “我要实现 xxx 功能”
- “请帮我完成 xxx 的代码与测试”
- “请在 conf-agent 中增加/修改 xxx”
- “请修改 conf-agent 的 xxx”

## 执行原则

1. **分阶段暂停确认**：本流程将研发拆分为多个 Phase。每个 Phase 执行完毕后，必须暂停并等待用户确认“可以继续”后，再进入下一个 Phase。不要在未获得用户确认的情况下自动推进到下一阶段。
2. **Git 提交前确认**：在任何 `git commit`、`git push` 或其他会改变 Git 仓库状态的操作之前，必须先向用户说明变更内容并取得明确同意。禁止自动执行 Git 提交或推送。
3. **Git push 默认目标为 origin**：如果获得用户授权执行 `git push`，默认推送到 `origin` 远程仓库，而不是 `upstream`。除非用户明确指定其他 remote，否则不使用 `upstream`。

## 研发阶段

### Phase 0. 需求澄清与范围界定

1. 让用户明确：
   - 功能目标（一句话描述）
   - 验收标准（必须通过的测试/行为）
   - 影响范围（是否改配置格式、是否改 prober 拉取、是否改 file_store 目录/清理行为、是否改 trigger/BFE reload）
2. 判断是否为“非平凡改动”（多文件、有架构选择、用户偏好影响实现）：
   - 是 → 调用 `EnterPlanMode`，先出设计文档/plan 再执行。
   - 否 → 直接进入 Phase 1。

完成本阶段后暂停，等待用户确认后再进入下一阶段。

### Phase 1. 修改 modifications 文档

`conf-agent` 侧的非平凡改动必须在 `docs/zh_cn/modifications/` 留下修改说明，便于后续维护与审计。

1. 在 `docs/zh_cn/modifications/` 下新建日期化目录，例如：
   ```
   docs/zh_cn/modifications/2026-08-28-conf-dir-cleanup/
   ```
2. 在该目录下创建：
   - `change-summary.md`：背景与目标、主要改动点、配置影响、兼容性说明、待实现清单
   - `design-changes.md`：配置/代码改动点、接口与行为变化
3. 如已有同主题目录，则更新其中的文档，不要重复创建。

完成本阶段后暂停，等待用户确认后再进入下一阶段。

### Phase 2. 更新 sys-design 文档

如果改动涉及系统设计、配置加载流程或目录管理，需要更新 `docs/zh_cn/sys-design/`：

1. 查找现有相关文档（例如 `总体设计文档.md`、`details/配置目录版本管理与清理.md`）。
2. 在文档中新增/修改对应章节，说明：
   - 配置结构变更（`config/config.go`、`config/config_file.go`）
   - `prober/file_store/trigger` 行为变化
   - `Reloader` 编排逻辑变化
   - 边界情况与默认值
3. 如需新增独立文档，直接创建 `.md` 文件，并在 `summary.md` 中建立链接。

完成本阶段后暂停，等待用户确认后再进入下一阶段。

### Phase 3. 更新 config 文档

如果改动新增、修改或废弃了配置项，必须同步更新 `docs/zh_cn/config/config.md`：

1. 在对应配置节新增/修改字段说明。
2. 说明字段类型、默认值、是否必填、合法性条件。
3. 如配置项可被 Reloader 覆盖，需说明继承规则。

完成本阶段后暂停，等待用户确认后再进入下一阶段。

### Phase 4. 代码实现

1. 阅读相关源码，确认修改范围：
   - 配置层：`config/config.go`、`config/config_file.go`
   - 编排层：`conf_reload/reloader.go`
   - 探测层：`conf_reload/prober/`
   - 存储层：`conf_reload/file_store/`
   - 触发层：`conf_reload/trigger/`
   - 生命周期：`agent/agent.go`
   - 工具层：`xfile/`、`xhttp/`、`xlog/`
2. 做最小改动，优先匹配现有代码风格：
   - 不引入未使用的依赖。
   - 不修改与本次需求无关的逻辑。
3. 关键实现完成后，先跑相关单元测试：
   ```bash
   cd conf-agent
   go test ./conf_reload/... ./config/... ./agent/... ./xfile/...
   ```

完成本阶段后暂停，等待用户确认后再进入下一阶段。

### Phase 5. 补充单元测试

1. 如果修改了 `file_store`、`prober`、`trigger` 等子模块，在对应 `_test.go` 中补充单元测试。
2. 如果修改了配置解析或归一化逻辑，在 `config/` 中补充用例。
3. 如果修改了 `xfile` 工具函数，补充跨平台（Windows/Linux）用例。
4. 运行单元测试：
   ```bash
   cd conf-agent
   go test ./...
   ```

完成本阶段后暂停，等待用户确认后再进入下一阶段。

### Phase 6. 补充集成测试设计文档

集成测试设计文档位于 `test/integration/tests/<module>/design.md`。在写测试代码之前，先补充对应场景的设计文档。

1. 在 `test/integration/tests/<module>/design.md` 中：
   - 模块概述补充新字段/新行为说明
   - mock 接口参数表补充新增字段
   - 测试场景总览表新增用例编号与简要说明
   - 详细设计章节新增每个测试例（设计思路、前置数据、执行步骤、请求参数、预期结果）
2. 如果是新的测试模块，在 `test/integration/tests/` 下新建目录并编写 `design.md`。

完成本阶段后暂停，等待用户确认后再进入下一阶段。

### Phase 7. 补充集成测试代码

集成测试代码位于 `test/integration/tests/<module>/`。

1. 根据设计文档在对应 `_test.go` 中实现用例。
2. 使用 `testutil/mock_server.go` 模拟上游服务，`testutil/runner.go` 启动/停止 `Reloader`。
3. 运行相关集成测试：
   ```bash
   cd conf-agent/test/integration
   go test -v -count=1 -timeout 60s ./tests/<module>/...
   ```

完成本阶段后暂停，等待用户确认后再进入下一阶段。

### Phase 8. 回归验证

1. 运行本次新增/修改模块的集成测试。
2. 运行全量单元测试与编译：
   ```bash
   cd conf-agent
   go test ./... && go build ./...
   ```
3. 如有失败，优先修复；修复后再次回归，直到全部通过。

完成本阶段后暂停，等待用户确认后再进入下一阶段。

### Phase 9. 收尾与总结

1. 检查是否有注释/文档描述的是旧行为，及时同步更新。
2. 向用户汇报：
   - 改动了哪些文件
   - 新增/修改了哪些测试
   - 验证结果
   - 仍存在的风险或待决策点（如有）

3. **Git 提交前必须人工确认**：如果用户要求或流程需要执行 `git commit`、`git push` 等 Git 操作，必须先向用户清晰说明本次提交内容（包含文件清单与主要变更摘要），并取得明确同意后再执行。禁止在未获授权的情况下自动提交或推送。获得授权后，`git push` 默认推送到 `origin`，不要推送到 `upstream`，除非用户明确指定。

## 常见陷阱

- **版本目录跨平台差异**：Windows 使用 junction，Linux 使用 symlink；`file_store` 中的路径与链接操作应通过 `xfile/` 封装并在两个平台都验证。
- **Stop() 不可重入导致测试超时**：`Agent.Stop()` 与 `Reloader.Stop()` 必须使用 `sync.Once` 关闭 stop 通道，否则重复调用会阻塞。
- **Reloader 轮询间隔影响测试时长**：集成测试中应缩短 `ReloadInterval`，并通过日志关键字判断 reload 完成，避免硬编码 sleep。
- **VersionKeepCount 继承关系**：`Basic.VersionKeepCount` 是全局默认值，每个 Reloader 可独立覆盖；实现与文档中都要说明优先级。
- **BFE reload 失败后不能切换符号链接**：`UpdateDefaultConfDir` 必须在 `TriggerBFEReload` 成功后调用，否则会导致 BFE 指向未生效配置。
- **mock URL 中的绝对路径编码**：Windows 路径含反斜杠与冒号，拼接到 BFE reload URL 时可能被错误编码；测试断言时应关注最终行为而非 URL 字符串。

## 推荐命令速查

```bash
# 构建二进制
cd conf-agent
go build -o conf-agent.exe .

# 全量单元测试
cd conf-agent
go test ./...

# 相关单元测试
go test ./conf_reload/file_store/... ./conf_reload/prober/... ./conf_reload/trigger/... ./config/... ./agent/... ./xfile/...

# 集成测试（指定模块）
cd conf-agent/test/integration
go test -v -count=1 -timeout 60s ./tests/cleanup/...

# 许可证头检查与修复
cd conf-agent
make license-check
make license-fix
```
