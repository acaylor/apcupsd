# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- `Status` no longer panics when the NIS sends a frame larger than 256 bytes.
- `Status` returns `io.ErrUnexpectedEOF` when the connection ends before the end-of-response terminator, instead of returning partial data without an error.
- A parse error no longer leaves the rest of the response unread, which made the next `Status` call return stale data.
- `Close` no longer blocks behind a pending `Status` call, so it can be used to abort a stalled request.
- `Status.Selftest` is now `true` for the `OK` result the NIS actually reports; previously it only matched `YES`, which the NIS never sends.

### Changed

- Concurrent `Status` calls on one `Client` are now serialized.
- Simplified NIS framing and status parsing.
- Removed the stale enumcheck references from the README and static analysis workflow.

## [0.1.0] - 2026-02-28

### Added

- `.gitignore` for Go build artifacts, coverage, IDE/OS files, and local config.
- **Development** section in README: prerequisites (Go 1.26), build/test/lint instructions.
- **Credits** section in README attributing the original project to [Matt Layher](https://github.com/mdlayher) ([github.com/mdlayher/apcupsd](https://github.com/mdlayher/apcupsd)).

### Changed

- Fork to [acaylor/apcupsd](https://github.com/acaylor/apcupsd); module path and git remote updated accordingly.
- Go 1.20 → **1.26** in `go.mod` and GitHub Actions (test + static analysis).
- `github.com/google/go-cmp` v0.5.9 → **v0.7.0**.
- LICENSE: retained original copyright (Matt Layher 2016–2023), added copyright for community-maintained fork (acaylor, 2026).
- GitHub Actions: `actions/setup-go@v2` → `@v5`, `actions/checkout@v2` → `@v4`.
- Static analysis workflow: enumcheck step disabled (dependency incompatible with Go 1.26); staticcheck and go vet retained.

[Unreleased]: https://github.com/acaylor/apcupsd/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/acaylor/apcupsd/releases/tag/v0.1.0
