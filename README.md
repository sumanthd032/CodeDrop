# CodeDrop

[![Go
Reference](https://pkg.go.dev/badge/github.com/sumanthd032/codedrop.svg)](https://pkg.go.dev/github.com/sumanthd032/codedrop)
[![License:
MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> CodeDrop enables developers to securely and temporarily hand off code
> artifacts via a CLI using client-side encryption, strict lifecycle
> policies, and content-addressed storage without permanent storage or
> UI surfaces.

CodeDrop is an **ephemeral transfer primitive**, not a repository.

------------------------------------------------------------------------

## The Problem

Developers constantly need to hand off build artifacts, logs, or quick
patches. Existing tools violate basic engineering hygiene:

-   **Google Drive / Slack:** Permanent storage for temporary needs, no
    enforced expiration and too slow.
-   **Pastebin / Public Links:** No strong access limits or
    cryptographic confidentiality.
-   **Server-Side Encryption:** The provider holds the keys and can read
    your data.

**CodeDrop solves handoff, not storage.**

------------------------------------------------------------------------

## Core Features

-   **Client-Side Convergent Encryption:** Files are chunked and
    encrypted locally via AES-256-GCM. The server never sees plaintext
    or keys.
-   **Content-Addressed Storage (CAS):** Encrypted chunks
    deduplicated via SHA-256 hashing.
-   **Atomic Lifecycle Enforcement:** Strict download limits enforced
    via Redis Lua scripts.
-   **Zero Data Retention:** Garbage Collector destroys chunks and
    metadata immediately upon expiration.
-   **Stream-First CLI UX:** Pipe-friendly and scriptable. No UI
    dashboards.

------------------------------------------------------------------------

## Architecture

**Client:** Go Cobra CLI (Chunking & AES-GCM Encryption)\
**API Server:** Go Chi Router (Stateless policy enforcement)\
**Metadata:** PostgreSQL\
**Counters:** Redis\
**Storage:** S3 / MinIO

------------------------------------------------------------------------

## Getting Started

### Prerequisites

-   Go 1.21+
-   Docker Desktop

### 1. Start Infrastructure

``` bash
docker compose up -d
```

### 2. Start API Server

``` bash
go run cmd/server/main.go
```

### 3. Build CLI

``` bash
go build -o codedrop cmd/cli/main.go
```

------------------------------------------------------------------------

## Usage

### Push

``` bash
./codedrop push secret_build.zip --expire 1h --max-views 2
```

### Pull

``` bash
./codedrop pull "http://localhost:8080/drop/a1b2c3d4#k=base64key..."
```

### Stats

``` bash
./codedrop stats
```
