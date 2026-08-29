# AGENTS.md — conf-agent

This file guides AI coding agents working on the `conf-agent/` codebase.

## Project overview

`conf-agent` is the **configuration delivery sidecar** of AI Gateway. It polls `ai-gateway-api` for the latest data-plane configuration, persists it locally, triggers BFE hot reload, and keeps a bounded set of versioned config directories.

AI Gateway system context (for orientation only):
- **AI Gateway API** (`rainway-ai-gateway/ai-gateway-api`): control plane, exposes Open/Inner APIs.
- **BFE** (`bfenetworks/bfe`): data plane, forwards traffic and consumes the configuration.
- **conf-agent** (this repo): fetches configuration and triggers BFE hot reload.
- **Dashboard** (`rainway-ai-gateway/ai-gateway-web`): Web UI for visual management.
- **Service Controller** (`bfenetworks/service-controller`): discovers and syncs Kubernetes backend services.

## High-level architecture

Entry point: `main.go`
- Parses flags (`-c conf_dir`, `-cf conf-agent.toml`).
- Loads config via `config.Init`.
- Initializes logging via `xlog.Init`.
- Creates and starts `agent.Agent`, which owns one or more `conf_reload.Reloader` goroutines.

Request flow per `Reloader`:
1. **prober** (`conf_reload/prober/`): polls `ai-gateway-api` InnerAPI endpoints and returns new/updated config files.
2. **file_store** (`conf_reload/file_store/`): writes files into a temporary version directory, then switches the symlink `mod_{name}` to the new version and cleans up old versions according to `VersionKeepCount`.
3. **trigger** (`conf_reload/trigger/`): calls BFE `/reload/{module}` monitor port endpoint to perform hot reload.

## Directory structure and module relationships

| Directory | Responsibility |
|-----------|----------------|
| `agent/` | Lifecycle container that starts/stops all `Reloaders`. |
| `conf_reload/` | Core reload orchestration: `Reloader`, plus sub-packages for prober, file_store, and trigger. |
| `conf_reload/prober/` | Fetches configuration from `ai-gateway-api`. Supports normal, multi-key JSON, and extra-file tasks. |
| `conf_reload/file_store/` | Persists config to disk, manages versioned directories, symlinks, and cleanup. |
| `conf_reload/trigger/` | Calls BFE monitor-port reload endpoint. |
| `config/` | TOML config loading and `ReloaderConfig` construction. |
| `xfile/` | File-system utilities: copy, symlink, junction helpers. |
| `xhttp/` | HTTP client helpers. |
| `xlog/` | Structured logging. |
| `version/` | Version constant. |
| `conf/` | Runtime TOML configuration samples. |
| `docs/` | User and design documentation (`zh_cn/`, `en_us/`). |
| `test/` | Integration tests using in-process `httptest` servers. |

## Build/test conventions

- **Go version**: 1.22 (`go.mod`).
- **Module**: `github.com/rainway-ai-gateway/conf-agent`.
- **Build**: `make` or `make build` produces `./conf-agent`.
- **Static build**: `make build-static`.
- **Cross-compile release**: `make release` builds `linux/amd64` and `linux/arm64` tarballs in `dist/`.
- **Test**:
  - `go test ./...` runs all unit tests.
  - `cd test/integration && go test -v -count=1 ./tests/...` runs integration tests.
- **License headers**: `make license-check` / `make license-fix` use `license-eye`.
- **Start locally**: `./conf-agent -c ./conf -cf conf-agent.toml`

## Common modification patterns

### Add a new reloader behavior or config task type
1. Update `config/config.go` and `config/config_file.go` if new config fields are needed.
2. Implement logic in the relevant `conf_reload/` sub-package (`prober`, `file_store`, or `trigger`).
3. Wire the new behavior into `conf_reload/reloader.go`.
4. Add unit tests in the sub-package (`*_test.go`) and integration tests in `test/integration/` if a full reload flow is affected.
5. Update `docs/zh_cn/sys-design/` and `docs/zh_cn/config/config.md` if behavior changes are user-visible.

### Change config directory or symlink handling
- Primary code: `conf_reload/file_store/file_store.go`.
- Cross-platform concerns: Windows junctions vs. Unix symlinks (`xfile/` helpers).
- Integration tests should cover both successful switch and cleanup, and failure rollback.

### Change BFE reload triggering
- Primary code: `conf_reload/trigger/`.
- Verify timeout, URL construction, and status-code handling.

### Change prober / remote config fetching
- Primary code: `conf_reload/prober/`.
- Verify query parameters (`version`, `bfe_cluster`), response parsing, and extra-file handling.

### Design-first workflow (recommended for non-trivial changes)
Follow the six-step methodology in `docs/zh_cn/README.md`:
1. Create `docs/zh_cn/modifications/YYYYMMDD-<summary>/` with `change-summary.md` (and `design-changes.md` if needed).
2. Update `docs/zh_cn/sys-design/` and `docs/zh_cn/sys-design/summary.md`.
3. Update `docs/zh_cn/config/config.md` if config semantics change.
4. Implement code: config → conf_reload sub-package → reloader orchestration.
5. Add/update unit tests and integration tests.
6. Summarize and decide whether to add a long-lived `docs/zh_cn/sys-design/details/` document.

## Agent guidelines

- **Follow `docs/zh_cn/README.md`** for non-trivial features; keep design docs and code in sync.
- **Keep `Reloader.Stop()` and `Agent.Stop()` safe to call multiple times**; they use `sync.Once` to close the stop channel.
- **Prefer platform-agnostic file operations**; use `xfile/` helpers for symlinks/junctions and test on both Windows and Linux when possible.
- **Unit-test file_store behaviors** directly in `conf_reload/file_store/file_store_test.go`; integration tests should exercise the full `prober → file_store → trigger` flow.
- **Do not commit generated files** such as `conf-agent`, `coverage.out`, or `dist/` artifacts.
- **Run tests**:
  - `go test ./...` after any production change.
  - `cd test/integration && go test -v -count=1 ./tests/...` after integration-affecting changes.
- **License headers**: all new source files need the Apache 2.0 / Rainway AI Gateway header. Use `make license-fix` if unsure.
- **Coordinate with `ai-gateway-api/`** when changes affect InnerAPI contract or config export formats.

## Useful references

- `README.md` / `docs/zh_cn/README.md` — project overview and system design index.
- `CONTRIBUTING.md` — workflow and code style.
- `docs/zh_cn/sys-design/summary.md` — system design index.
- `docs/zh_cn/config/config.md` — configuration reference.
- `test/integration/tests/cleanup/design.md` — integration test scenario design.
- `Makefile` — build, test, release, and license targets.
