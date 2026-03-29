# filesystem-gateway

Authenticated HTTP API for workload filesystem management (NFS volume): directory CRUD, file operations, downloads with archive extraction, and backups.
Part of the [homelab-ai](https://github.com/lobo235/homelab-ai) platform.

## Module

`github.com/lobo235/filesystem-gateway`

## Quick Start

```bash
cp .env.example .env
# Fill in required values
go run ./cmd/server
```

## Build, Test, Run

> Go is installed at `~/bin/go/bin/go` (also on `$PATH` via `.bashrc`).

```bash
# Build
make build

# Run tests
make test

# Run tests with verbose output
go test -v ./...

# Run linter
make lint

# Coverage report (opens in browser)
make cover

# Run the server (requires .env or env vars)
make run

# Build binary
go build -o filesystem-gateway ./cmd/server
```

## Project Layout

```
filesystem-gateway/
├── Dockerfile
├── Makefile
├── go.mod / go.sum
├── .env.example              # dev template — never commit real values
├── .gitignore
├── .golangci.yml             # strict linter config
├── .githooks/pre-commit      # runs lint + tests; activate with `make hooks`
├── CLAUDE.md                 # this file
├── README.md
├── CHANGELOG.md
├── cmd/
│   └── server/
│       └── main.go           # entry point
├── deploy/
│   ├── filesystem-gateway.hcl         # Nomad job spec (placeholders only)
│   └── filesystem-gateway.policy.hcl  # Nomad ACL policy
└── internal/
    ├── config/
    │   ├── config.go         # ENV var loading & validation
    │   └── config_test.go    # config tests
    ├── nfs/
    │   ├── client.go         # NFS filesystem operations (path traversal prevention)
    │   └── client_test.go    # unit tests (path traversal, CRUD, backup/restore)
    └── api/
        ├── server.go         # HTTP mux + Run()
        ├── server_test.go    # handler tests via httptest (all endpoints)
        ├── middleware.go     # Bearer auth + request logging + X-Trace-ID
        ├── handlers.go       # all route handlers
        ├── validate.go       # input validation (server names, download URLs)
        ├── validate_test.go  # validation tests
        ├── errors.go         # writeError / writeJSON helpers
        └── health.go         # GET /health (unauthenticated)
```

## Configuration

All config via ENV vars. Loaded from `.env` in development (via `godotenv`; missing file silently ignored). In production, secrets are injected by Nomad Vault Workload Identity — the app never talks to Vault directly.

| Var | Required | Default | Purpose |
|-----|----------|---------|---------|
| `NFS_BASE_PATH` | yes | — | Base path for server data on NFS (e.g. `/mnt/data/minecraft`) |
| `GATEWAY_API_KEY` | yes | — | Bearer token for callers of this API |
| `PORT` | no | `8080` | Listen port |
| `LOG_LEVEL` | no | `info` | Verbosity: `debug`, `info`, `warn`, `error` |
| `DATA_DIR` | no | `/data` | Directory for `.backup-status` and `.download-*.status` tracking files |
| `MAX_DOWNLOAD_SIZE` | no | `2147483648` | Max download size in bytes (2GB default) |
| `MAX_WRITE_FILE_SIZE` | no | `1048576` | Max file write size in bytes (1MB default) |
| `MAX_EXTRACT_SIZE` | no | `10737418240` | Max total extracted archive size in bytes (10GB default) |
| `ALLOWED_DOWNLOAD_HOSTS` | no | — | Additional download hosts beyond built-in defaults (comma-separated, `host` or `host/pathPrefix`) |

## Architecture

```
cmd/server/main.go              — entry point, wires deps, handles SIGINT/SIGTERM
internal/config/config.go       — ENV-based config with validation
internal/api/server.go          — HTTP server, route registration
internal/api/middleware.go      — bearerAuth + requestLogger + X-Trace-ID propagation
internal/api/handlers.go        — route handlers (servers, files, backups, downloads)
internal/api/validate.go        — input validation (server names, download URL allowlist)
internal/api/errors.go          — writeError / writeJSON helpers
internal/api/health.go          — GET /health handler (unauthenticated)
internal/nfs/client.go          — NFS filesystem client (path traversal prevention, CRUD, backups)
```

**Backup flow:** `POST /servers/{name}/backups` triggers an async backup using pzstd (parallel zstd). Returns immediately with a backup ID. Status tracked in `.backup-status` JSON files at `DATA_DIR/<server-name>.backup-status`. Files stored at `<NFS_BASE_PATH>/<server>/backups/<id>.tar.zst`.

**Download flow:** `POST /servers/{name}/download` triggers an async download. Returns immediately with a download ID and status "running" (201 Created). Status tracked in `.download-<id>.status` JSON files at `DATA_DIR/<server-name>.download-<id>.status`. Poll `GET /servers/{name}/downloads/{id}` for completion. Status includes result (files_count, total_bytes) on success, or error message on failure.

**Restore sequencing:** `POST /servers/{name}/restore` performs NO liveness check — it always attempts the restore. The orchestration layer is responsible for stopping the workload before restore and restarting it afterward.

## API Routes

All routes except `/health` require `Authorization: Bearer <GATEWAY_API_KEY>`.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | No | Returns `{"status":"ok","version":"..."}` |
| GET | `/servers` | Yes | List server directories on NFS volume |
| POST | `/servers` | Yes | Create server dir `{"name":"...","uid":N,"gid":N}` |
| DELETE | `/servers/{name}` | Yes | Delete server dir (requires `?confirm=true`) |
| POST | `/servers/{name}/download` | Yes | Start async download `{"url":"...","dest_path":"...","extract":bool,"uid":N,"gid":N,"mode":"overwrite\|skip_existing\|clean_first"}` returns 201 with `{"id":"...","status":"running"}` |
| GET | `/servers/{name}/downloads/{downloadID}` | Yes | Download status/details |
| GET | `/servers/{name}/archive-contents` | Yes | List archive entries (`?path=mods.zip`) supports .zip, .tar.gz, .tar.zst |
| GET | `/servers/{name}/disk-usage` | Yes | Disk usage in bytes |
| GET | `/servers/{name}/files` | Yes | List files (`?path=subdir`) |
| GET | `/servers/{name}/files/read` | Yes | Read file (`?path=logs/latest.log`) max 1MB |
| GET | `/servers/{name}/files/grep` | Yes | Grep (`?path=...&pattern=...`) max 10k lines/5MB |
| POST | `/servers/{name}/files/write` | Yes | Write file `{"path":"...","content":"...","uid":N,"gid":N}` max configurable (default 1MB) |
| POST | `/servers/{name}/files/move` | Yes | Move/rename file `{"src_path":"...","dst_path":"...","uid":N,"gid":N}` |
| DELETE | `/servers/{name}/files/delete` | Yes | Delete file or directory (`?path=...`) |
| GET | `/servers/{name}/backups` | Yes | List available `.tar.zst` backups |
| POST | `/servers/{name}/backups` | Yes | Trigger async backup `{"uid":N,"gid":N}` (optional); returns backup ID |
| GET | `/servers/{name}/backups/{backupID}` | Yes | Backup status/details |
| POST | `/servers/{name}/restore` | Yes | Restore from backup `{"backup_id":"...","uid":N,"gid":N}` |
| POST | `/servers/{name}/migrate` | Yes | Rename server `{"new_name":"..."}` |

### Input Validation

- **Server names:** `^[a-z0-9][a-z0-9-]{0,47}$`

### Path Traversal Prevention

All path parameters are resolved via `filepath.Abs()` and verified to have `NFS_BASE_PATH` as a prefix before any filesystem operation. Requests with `../` sequences, absolute paths outside the base, or URL-encoded traversal sequences are rejected with HTTP 400 and code `path_traversal`.

### Download URL Allowlist

Downloads are restricted to HTTPS URLs on allowed hosts. Built-in defaults cover common Minecraft mod CDNs and trusted GitHub paths. Additional hosts can be configured via `ALLOWED_DOWNLOAD_HOSTS` env var. The `validDownloadURL` function takes the combined list as a parameter.

## Testing Approach

```
internal/nfs/client_test.go     — path traversal prevention, filesystem CRUD, backup operations
internal/api/server_test.go     — handler tests via httptest (all endpoints)
internal/api/validate_test.go   — download URL validation, server name validation
internal/config/config_test.go  — config loading and validation
```

Key patterns:
- Table-driven tests for input validation (server names, path traversal)
- Test both success paths and error paths
- **Path traversal tests are mandatory:** `../`, `../../`, absolute paths, URL-encoded `%2e%2e%2f`

## Coding Conventions

- No external router, ORM, or framework — minimal dependency footprint
- Error responses always use `writeError(w, status, code, message)` with machine-readable `code`
- Route handlers return `http.HandlerFunc`
- All upstream errors wrapped with `fmt.Errorf("context: %w", err)`
- `X-Trace-ID` header propagated from request context to all upstream calls and log lines
- Structured JSON logging via `log/slog`; version logged on startup; every request access-logged

## Security Rules

> **Claude must enforce all rules below on every commit and push without exception.**

1. **Never commit secrets:** No `.env`, tokens, API keys, passwords, or credentials of any kind.
2. **Never commit infrastructure identifiers:** No real hostnames, IP addresses, datacenter names, node pool names, Consul service names, Vault paths with real values, Traefik routing rules with real domains, or any value that reveals homelab architecture. Use generic placeholders (`dc1`, `default`, `example.com`, `your-node-pool`, `your-service`).
3. **Unknown files:** If `git status` shows a file Claude didn't create, ask the operator before staging it.
4. **Pre-commit checks (must all pass before committing):**
   - `go test ./...` — all tests must pass
   - `golangci-lint run` — no lint errors
5. **Docs accuracy:** Review all changed `.md` files before committing — documentation must reflect the current state of the code in the same commit.
6. **Version bump:** Before any `git commit`, review the changes and determine the appropriate SemVer bump (MAJOR/MINOR/PATCH). Present the rationale and proposed new version to the operator and wait for confirmation before tagging or referencing the new version.
7. **Push confirmation:** Before any `git push`, show the operator a summary of what will be pushed (commits, branch, remote) and wait for explicit confirmation.
8. **Commit messages:** Must not contain real hostnames, IPs, or infrastructure identifiers.

## Versioning & Releases

SemVer (`MAJOR.MINOR.PATCH`). Git tags are the source of truth.

```bash
git tag v1.0.0 && git push origin v1.0.0
```

This triggers the Docker workflow which publishes:
- `gitea.big.netlobo.com/netlobo/filesystem-gateway:v1.0.0`
- `gitea.big.netlobo.com/netlobo/filesystem-gateway:v1.0`
- `gitea.big.netlobo.com/netlobo/filesystem-gateway:latest`
- `gitea.big.netlobo.com/netlobo/filesystem-gateway:<short-sha>`

Version is embedded at build time: `-ldflags "-X main.version=v1.0.0"` — defaults to `"dev"` for local builds. Exposed in `GET /health` response and logged on startup.

## Docker

```bash
# Build (version defaults to "dev")
docker build -t filesystem-gateway .

# Build with explicit version
docker build --build-arg VERSION=v1.0.0 -t filesystem-gateway .

# Run
docker run --env-file .env -p 8080:8080 filesystem-gateway
```

Multi-stage build: `golang:1.24-alpine` -> `alpine:3.21`. Statically compiled (`CGO_ENABLED=0`). Runtime image includes `zstd` package for pzstd backup compression.

## Known Limitations

- **Restore requires workload stop:** `POST /servers/{name}/restore` does not check workload liveness. If the workload is running during restore, it may overwrite restored data, causing silent data loss. The orchestration layer must stop the workload before calling restore.
- **Single backup status per server:** Only the most recent backup status is tracked per server in the `.backup-status` file. Starting a new backup overwrites the previous status.
- **Backup timestamp IDs:** Backup IDs use second-precision UTC timestamps (`2006-01-02T15-04-05`). Starting two backups for the same server within the same second will collide.
- **pzstd required at runtime:** The `pzstd` binary must be available in `PATH` for backup/restore operations. The Dockerfile includes it via `apk add zstd`.
- **Download timestamp IDs:** Download IDs use second-precision UTC timestamps (`2006-01-02T15-04-05`). Starting two downloads for the same server within the same second will collide.
- **Download status files are per-download:** Each download creates its own status file (`<server>.download-<id>.status`), unlike backups which overwrite a single status file per server.
