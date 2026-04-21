# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v2.0.1] - 2026-04-20

### Fixed
- `--version` (and `-version`, `-v`) now prints `filesystem-gateway version <semver> <os>/<arch>` and exits before any config loading. Previously the flag was ignored and the binary fell through to config validation, which aborted with an `NFS_BASE_PATH is required` error when env vars were absent.

## [v2.0.0] - 2026-04-20

### Added
- `GET /servers/{name}/files/find` — find entries under a server directory. Query params: `path` (relative, default root), `name_glob` (optional, matched against basename via `filepath.Match`), `type` (`f` or `d`, empty = both), `max_depth` (positive int = cap depth relative to `path`; zero or negative = unlimited), `modified_since` (RFC3339), `skip_exts` (comma list of extensions to skip), `max_entries` (default 1000, hard cap 10000). Returns `{"entries":[{"path","type","size","mtime"}], "truncated":bool}`. Symlinks are skipped (not followed, not reported). A malformed `name_glob` returns 400.
- `GET /servers/{name}/files/grep` gained `case_insensitive` (bool) and `skip_exts` (comma list) query params. Files whose extension matches `skip_exts` are passed to `grep` as `--exclude` patterns so they are never opened.

### Changed
- **BREAKING:** `GET /servers/{name}/files/grep` response shape. Previously `matches` was `[]string` in `path:line:text` form (ambiguous for filenames containing colons). Now `matches` is `[{"path","line","text"}]` with `path` relative to the server root. Gateway now invokes `grep -rnZIH` and parses NUL-separated output so filenames containing colons are handled correctly. No known external consumers of the old shape at time of change.

## [v1.2.0] - 2026-04-16

### Added
- `GET /servers/{name}/ls` — list entries under a server directory. Query params: `path` (relative, default root), `recursive` (bool, default false), `max_entries` (default 1000, hard cap 10000). Returns `{"entries":[{"path","type","size","mtime"}], "truncated":bool}`. Symlinks are not followed and not reported.
- `GET /servers/{name}/read` — read a file as raw bytes. Query params: `path` (required), `max_bytes` (default 1 MiB, hard cap 10 MiB — exceeding returns 413), `origin` (`start`|`end`, default `start`; `end` returns the last `max_bytes` for log tailing). Response body is the file content; Content-Type is sniffed (`text/plain; charset=utf-8` when UTF-8 validates, else `application/octet-stream`). Sets `Content-Length` and `X-Truncated: true` when the file exceeds `max_bytes`. Returns 415 for non-regular files (device/socket/FIFO).
- Per-server path scoping for the new endpoints: after symlink resolution the target must stay inside the specific server's directory, stricter than the existing basePath-level check.

## [v1.1.2] - 2026-04-16

### Changed
- Consolidated `.gitea/workflows/docker.yml` into a single `ci.yml` with lint, test, build, and docker jobs. The docker push is now gated behind lint+test+build passing, so broken code cannot reach the registry or Nomad.
- Pin `golangci-lint` version in `.golangci-version` file (`v2.11.3`) as the single source of truth for local and CI. The Makefile's `lint` target installs the pinned version into `./bin/` via the official `install.sh` script, and the CI workflow calls `make lint` rather than duplicating install logic.
- Docker build now uses Buildx with registry layer caching (`:buildcache` tag, `mode=max`) for faster rebuilds.
- Makefile `test` and `cover` targets now pass `-race`.

## [v1.1.1] - 2026-04-16

### Added
- `update` stanza with `auto_revert` in Nomad job spec
- `force_pull = true` in Nomad Docker config

### Changed
- Build output now goes to `bin/` directory instead of project root
- `make clean` removes `bin/` directory instead of a bare binary
- `.gitignore` uses `/bin/` pattern instead of bare binary name, and adds `.env.*`
- Registry hostname in deploy spec and CLAUDE.md replaced with `gitea.example.com` placeholder

## [v1.1.0] - 2026-04-16

### Added
- `GET /servers/{name}` — stat a single server directory; returns name, bytes, uid, gid, mode, mod_time. Returns 404 if the directory does not exist. Lets clients do "create if missing" in one round trip.
- `POST /servers/{name}/chmod` — set permission bits on the server root directory (non-recursive). Request body `{"mode": "0770"}` accepts a 3-digit octal string with optional leading zero (`^0?[0-7]{3}$`). Returns 404 if the directory does not exist.
- Server name validation widened to allow up to three path segments (e.g., `project/env/job`); previously capped at two. Each segment still constrained to `[a-z0-9][a-z0-9-]{0,47}`.

## [v1.0.1] - 2026-04-07

### Fixed
- Server name validation now accepts one level of subdirectory nesting (e.g., `minecraft/atm9`) — previously rejected names with `/`, breaking all Minecraft server creation via the MCP server which sends `minecraft/<name>` paths

## [v1.0.0] - 2026-03-28

### Added
- Initial release: filesystem operations extracted from minecraft-gateway
- Directory management: create, delete, rename (migrate), list, disk usage
- File operations: list, read, write, grep, move, delete
- Async file downloads with archive extraction (zip, tar.gz, tar.zst)
- Async backups with pzstd compression and restore
- Archive content listing
- Configurable download URL allowlist (ALLOWED_DOWNLOAD_HOSTS)
- Path traversal prevention via SafePath
- Bearer token authentication
- Structured JSON logging with X-Trace-ID propagation
