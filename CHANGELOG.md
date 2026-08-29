<!--
This changelog should always be read on `master` branch. Its contents on other branches
does not necessarily reflect the changes.
-->

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).


## [v0.0.6] - 2026-08-29

### Added
- Add `VersionKeepCount` config option to control how many old config directories to keep.
- Add `.conf-agent-version` marker files in versioned config directories.
- Add `Stop()` methods to `Reloader` and `Agent` for graceful shutdown.
- Add integration tests for config directory cleanup.
- Add system design documentation and `AGENTS.md`.

### Changed
- Harden `UpdateDefaultConfDir` to handle missing/broken symlinks, plain directories, and Windows junctions.
- Migrate module path and copyright from `infinity-ai-gateway` to `rainway-ai-gateway`.

### Fixed
- Fix `Reloader` log bug on `UpdateDefaultConfDir` failure.


## [v0.0.5] - 2026-08-01

### Added
- Support for `mod_ai_route` configuration loading in conf-agent.

### Fixed
- On Windows, use directory junctions instead of symbolic links to avoid symlink permission issues.
- Fixed `FileLink` relative-path symlink creation on non-Windows platforms.
- Fixed related unit tests for the file-link implementation.


## [v0.0.2] - 2021-12-07

### Changed
- adjust auth header format to enhance authentication and authorization


## [v0.0.1] - 2021-10-19

### Added
- Initial released version

[v0.0.6]: https://github.com/bfenetworks/conf-agent/compare/v0.0.5...v0.0.6
[v0.0.5]: https://github.com/bfenetworks/conf-agent/compare/v0.0.2...v0.0.5
[v0.0.2]: https://github.com/bfenetworks/conf-agent/compare/v0.0.1...v0.0.2
[v0.0.1]: https://github.com/bfenetworks/conf-agent/releases/tag/v0.0.1
