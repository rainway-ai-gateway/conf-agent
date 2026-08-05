<!--
This changelog should always be read on `master` branch. Its contents on other branches
does not necessarily reflect the changes.
-->

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).


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

[v0.0.2]: https://github.com/bfenetworks/conf-agent/compare/v0.0.1...v0.0.2
[v0.0.1]: https://github.com/bfenetworks/conf-agent/releases/tag/v0.0.1
