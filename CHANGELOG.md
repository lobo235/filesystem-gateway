# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
