# CodeDrop

A secure, self-destructing file sharing service built with Go. Upload encrypted files that automatically expire after a set duration or number of downloads.

## Features

- **Client-side encryption** — Files are encrypted before upload using a salt-based key derivation
- **Chunked uploads** — Large files are split into 5 MB chunks for reliable transfer
- **Auto-expiry** — Drops expire after a configurable duration (up to 24 hours)
- **Download limits** — Set a maximum number of downloads per drop
- **S3-compatible storage** — Uses MinIO locally, swappable with AWS S3 in production
- **Graceful shutdown** — Server handles `SIGINT`/`SIGTERM` for clean process termination

## Architecture

```
cmd/server/         → Application entry point
internal/
  api/              → HTTP handlers, router, and request/response models (chi)
  db/               → Postgres connection, migrations, and schema
  store/            → S3/MinIO object storage client
docker-compose.yml  → Local infrastructure (Postgres, Redis, MinIO)
```

### Tech Stack

| Component | Technology |
|-----------|------------|
| Language  | Go |
| Router    | [chi](https://github.com/go-chi/chi) |
| Database  | PostgreSQL (via [sqlx](https://github.com/jmoiron/sqlx)) |
| Storage   | S3-compatible (MinIO / AWS S3) |
| Cache     | Redis (provisioned, not yet integrated) |

## Getting Started

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)

### 1. Start infrastructure

```bash
docker-compose up -d
```

This starts:
- **Postgres** on port `5432`
- **Redis** on port `6379`
- **MinIO** on port `9000` (console on `9001`)

A `codedrop-bucket` is automatically created in MinIO.

### 2. Run the server

```bash
go run cmd/server/main.go
```

The API server starts on `http://localhost:8080`.

### Environment Variables

| Variable      | Default     | Description             |
|---------------|-------------|-------------------------|
| `DB_HOST`     | `localhost` | Postgres host           |
| `DB_PORT`     | `5432`      | Postgres port           |
| `DB_USER`     | `user`      | Postgres user           |
| `DB_PASSWORD` | `password`  | Postgres password       |
| `DB_NAME`     | `codedrop`  | Postgres database name  |

## API

### Health Check

```
GET /health
```

Returns `{"status": "healthy", "db": "connected"}`.

### Create a Drop

```
POST /api/v1/drop
Content-Type: application/json

{
  "file_name": "notes.txt",
  "file_size": 1048576,
  "encryption_salt": "random-salt-value",
  "expires_in": "1h",
  "max_downloads": 3
}
```

**Response** — `200 OK`

```json
{
  "drop_id": "uuid",
  "expires_at": "2025-01-01T01:00:00Z"
}
```

### Upload a Chunk

```
POST /api/v1/drop/{id}/chunk
X-Chunk-Index: 0
Content-Type: application/octet-stream

<binary data>
```

**Response** — `201 Created`

```json
{
  "status": "uploaded"
}
```

Chunks are limited to **5 MB** each.

## License

This project does not yet have a license. Contact the maintainer for usage terms.
