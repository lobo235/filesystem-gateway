# filesystem-gateway

Authenticated HTTP API for workload filesystem management (NFS volume): directory CRUD, file operations, downloads with archive extraction, and backups. Part of the [homelab-ai](https://github.com/lobo235/homelab-ai) platform.

## What it does

- Manages workload directories on NFS volumes (create, delete, rename)
- File operations: list, read, write, move, delete, grep
- Async downloads with archive extraction (zip, tar.gz, tar.zst)
- Async backups using pzstd compression with restore support
- Strict path traversal prevention
- Configurable download URL allowlist

## Quick Start

```bash
cp .env.example .env
# Fill in required values
go run ./cmd/server
```

## API

All routes except `/health` require `Authorization: Bearer <GATEWAY_API_KEY>`.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (unauthenticated) |
| GET | `/servers` | List server directories |
| POST | `/servers` | Create server directory |
| GET | `/servers/{name}` | Stat single directory (name, bytes, uid, gid, mode, mod_time); 404 if missing |
| DELETE | `/servers/{name}` | Delete server directory (requires `?confirm=true`) |
| POST | `/servers/{name}/chmod` | Set root directory mode (non-recursive); body `{"mode":"0770"}` |
| POST | `/servers/{name}/download` | Start async download |
| GET | `/servers/{name}/downloads/{downloadID}` | Download status |
| GET | `/servers/{name}/archive-contents` | List archive entries |
| GET | `/servers/{name}/disk-usage` | Disk usage in bytes |
| GET | `/servers/{name}/files` | List files |
| GET | `/servers/{name}/files/read` | Read file contents |
| GET | `/servers/{name}/files/grep` | Grep files |
| POST | `/servers/{name}/files/write` | Write file |
| POST | `/servers/{name}/files/move` | Move/rename file |
| DELETE | `/servers/{name}/files/delete` | Delete file or directory |
| GET | `/servers/{name}/backups` | List backups |
| POST | `/servers/{name}/backups` | Start async backup |
| GET | `/servers/{name}/backups/{backupID}` | Backup status |
| POST | `/servers/{name}/restore` | Restore from backup |
| POST | `/servers/{name}/migrate` | Rename server directory |

## Build

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linter
make cover    # Coverage report
```

## Docker

```bash
# Build (version defaults to "dev")
docker build -t filesystem-gateway .

# Build with explicit version
docker build --build-arg VERSION=v1.0.0 -t filesystem-gateway .

# Run
docker run --env-file .env -p 8080:8080 filesystem-gateway
```

## License

Private -- internal homelab use only.
