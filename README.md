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

## Getting Started (Local Development)

### Prerequisites

* [Go 1.21+](https://go.dev/dl/)
* [Docker Desktop](https://www.docker.com/products/docker-desktop/)

### 1. Start Infrastructure

CodeDrop relies on Postgres, Redis, and MinIO (local S3). Start them using Docker Compose:

``` bash
docker compose up -d
```

### 2. Start API Server

``` bash
go run cmd/server/main.go
```

The server will automatically run database migrations and connect to Redis/MinIO on startup.

### 3. Build CLI

``` bash
go build -o codedrop cmd/cli/main.go
# Optional: Move to your PATH so you can use it anywhere
# mv codedrop /usr/local/bin/
```

------------------------------------------------------------------------

## Usage

### Push
Encrypt and upload a file with strict lifecycle policies.

``` bash
./codedrop push secret_build.zip --expire 1h --max-views 2
```

### Pull
Download, verify integrity, and decrypt locally. Note: Place the URL in quotes to prevent the shell from interpreting the # fragment.

``` bash
./codedrop pull "http://localhost:8080/drop/a1b2c3d4#k=base64key..."
```

### Stats
View real-time observability data, including storage saved by the CAS deduplication engine.
``` bash
./codedrop stats
```

## Security & Threat Model

**Honest-but-Curious Server**: CodeDrop assumes the server infrastructure is compromised. Because of Client-Side Encryption, the server only hosts mathematical garbage.

**URL Fragment Key Distribution**: The decryption key is appended to the URL as a fragment (#k=...). Browsers and HTTP clients never transmit fragments to the server. The key strictly remains on the sender and receiver's machines.

**Convergent Encryption Paradox**: Standard E2EE breaks deduplication (CAS). CodeDrop solves this by deriving the encryption key and AES-GCM nonce from the SHA-256 hash of the local file. Identical files produce identical ciphertext, allowing the server to deduplicate without ever knowing the plaintext.