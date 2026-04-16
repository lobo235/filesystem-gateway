# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
