# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Gitea Docker workflow: add Buildx setup and registry layer caching, gate build on lint+test job
- GitHub workflows: consolidate ci.yml and docker.yml into single ci.yml with Docker job gated on test job
- Nomad job spec: add `update` stanza with auto_revert, add `force_pull = true` to Docker config
- Makefile: build output now goes to `bin/` directory instead of project root
- .gitignore: use `/bin/` pattern instead of bare binary name, add `.env.*` pattern

### Fixed
- Replace real infrastructure hostname with `gitea.example.com` placeholder in workflows, deploy spec, and docs
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
