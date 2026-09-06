# Metering — Album gRPC Service (Go)

A minimal Go gRPC microservice for managing **record album** metadata, with
Protocol Buffers code generation, gRPC server reflection, and a Gin HTTP
handler stub for future REST exposure.

> Module: `github.com/ravirajsubramanian/metering`
> Language: Go 1.27 · gRPC · Protocol Buffers · Gin

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture & Project Layout](#architecture--project-layout)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Regenerating Protobuf Code](#regenerating-protobuf-code)
- [Running the Server](#running-the-server)
- [API Reference](#api-reference)
  - [Proto Definition](#proto-definition)
  - [Messages](#messages)
  - [Service: `AlbumService`](#service-albumservice)
  - [Error Handling](#error-handling)
- [Client Examples](#client-examples)
  - [grpcurl](#grpcurl)
  - [grpcui / Postman](#grpcui--postman)
  - [Go Client](#go-client)
- [REST / Gin Component (`cmd/lease`)](#rest--gin-component-cmdeletelease)
- [Configuration](#configuration)
- [Development Guide](#development-guide)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Roadmap](#roadmap)
- [License](#license)

---

## Overview

This repository is a starter / learning project for a **metering-style backend
service** built around a classic "album store" domain (id, title, artist,
price). It currently exposes:

1. A **gRPC server** on port `:50051` implementing `album.AlbumService/Read`.
2. **Generated protobuf Go types** checked into `common/`.
3. A **Gin handler stub** in `cmd/lease/lease.go` (`GET /albums` style handler)
   intended for a future HTTP gateway / lease API.
4. An empty `internal/` directory reserved for private business logic
   (storage, metering, billing, auth).

Seed data (in both `main.go` and `cmd/lease/lease.go`):

| ID  | Title                            | Artist         | Price |
|-----|----------------------------------|----------------|-------|
| `1` | Blue Train                       | John Coltrane  | 56.99 |
| `2` | Jeru                             | Gerry Mulligan | 17.99 |
| `3` | Sarah Vaughan and Clifford Brown | Sarah Vaughan  | 39.99 |

---

## Features

- **gRPC unary RPC** `Read(ReadRequest) → ReadResponse` with `NotFound`
  semantics.
- **Thread-safe server struct** (`sync.RWMutex` + in-memory `map[string]*pb.Album`).
- **gRPC reflection** enabled in non-production mode for `grpcurl`, `grpcui`,
  Postman, and Evans without needing the `.proto` file.
- **Protobuf-first contract** in `proto/album.proto`; Go stubs generated with
  `protoc-gen-go` + `protoc-gen-go-grpc`.
- **Gin JSON handler** example (`GetAlbums`) returning the seed album list.
- Small, dependency-light footprint: `grpc`, `protobuf`, `gin`.

---

## Architecture & Project Layout

```text
metering/
├── main.go                  # gRPC server entrypoint (:50051), AlbumService impl
├── go.mod / go.sum          # module github.com/ravirajsubramanian/metering, Go 1.27
├── proto/
│   └── album.proto          # source of truth: Album, ReadRequest/Response, AlbumService
├── common/
│   ├── album.pb.go          # generated message types (protoc-gen-go v1.36.12)
│   └── album_grpc.pb.go     # generated client/server stubs (protoc-gen-go-grpc v1.6.2)
├── cmd/
│   └── lease/
│       └── lease.go         # Gin handler stub: Print(), GetAlbums(), seed data
├── internal/                # reserved for private app logic (currently empty)
└── README.md
```

Request flow (current):

```text
gRPC client ── Read(ReadRequest{id}) ──► :50051 ──► server.Read()
                                                        │
                                                        ▼
                                              in-memory lookup
                                              (seed slice + map)
                                                        │
                                                        ▼
                                              ReadResponse{album} / NotFound
```

Planned / stubbed flow:

```text
HTTP client ── GET /albums ──► gin.Context ──► cmd/lease.GetAlbums()
                                                 └── c.IndentedJSON(200, albums)
```

---

## Prerequisites

- **Go 1.27+** (`go version`)
- **protoc v7.x** (only needed to regenerate stubs; tested with `v7.36.0`)
  - plugins: `protoc-gen-go`, `protoc-gen-go-grpc`
- Optional for manual testing:
  - [`grpcurl`](https://github.com/fullstorydev/grpcurl)
  - [`grpcui`](https://github.com/fullstorydev/grpcui) or Postman (gRPC mode)
  - [`evans`](https://github.com/ktr0731/evans)

Install protoc plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

---

## Installation

```bash
# 1. Clone (adjust URL to your fork)
git clone https://github.com/ravirajsubramanian/metering.git
cd metering

# 2. Download dependencies
go mod download

# 3. Vet / build
go vet ./...
go build -o bin/metering .
```

Dependencies of note (`go.mod`):

- `google.golang.org/grpc v1.83.2`
- `google.golang.org/protobuf v1.36.12`
- `github.com/gin-gonic/gin v1.12.0`

---

## Regenerating Protobuf Code

The generated files in `common/` are checked in, but regenerate after any
`.proto` change.

From the repo root:

```bash
# Install plugins first (see Prerequisites)
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/album.proto
```

Notes:

- `proto/album.proto` currently declares `option go_package = "/common";`.
  For cleaner imports prefer a full import path, e.g.:
  `option go_package = "github.com/ravirajsubramanian/metering/common;common";`
  then move outputs with `--go_out=.` accordingly.
- The comment at the bottom of `proto/album.proto`
  (`// protoc --go_out=. --go-grpc_out=. proto/album.proto`) is the shorthand
  variant of the command above.
- Generated with `protoc-gen-go v1.36.12`, `protoc-gen-go-grpc v1.6.2`,
  `protoc v7.36.0` — keep versions close to avoid churn.

---

## Running the Server

```bash
go run .
# output: gRPC Server with reflection running on port :50051...
```

The server:

- Listens on TCP `:50051` (`net.Listen("tcp", ":50051")` in `main.go:82`).
- Registers `AlbumService` via `pb.RegisterAlbumServiceServer`.
- Enables **reflection** unless `ENV=production` (`main.go:92-94`).

```bash
# production mode (disables reflection)
ENV=production go run .
```

Verify it's listening:

```bash
lsof -i :50051        # macOS
# or
ss -ltnp | grep 50051 # Linux
```

---

## API Reference

### Proto Definition

Source: `proto/album.proto`

```proto
syntax = "proto3";

package album;

option go_package = "/common";

message Album {
  string id = 1;
  string title = 2;
  string artist = 3;
  double price = 4;
}

message ReadRequest {
  string id = 1;
}

message ReadResponse {
  Album album = 1;
}

service AlbumService {
  rpc Read(ReadRequest) returns (ReadResponse);
}
```

Full method name: **`/album.AlbumService/Read`**

### Messages

**`Album`**

| Field  | Type   | # | Description              |
|--------|--------|---|--------------------------|
| `id`   | string | 1 | Album identifier (`"1"`) |
| `title`| string | 2 | Album title              |
| `artist`| string| 3 | Artist name              |
| `price`| double | 4 | Price in dollars         |

**`ReadRequest`**

| Field | Type   | # | Description          |
|-------|--------|---|----------------------|
| `id`  | string | 1 | ID of album to fetch |

**`ReadResponse`**

| Field   | Type    | # | Description        |
|---------|---------|---|--------------------|
| `album` | `Album` | 1 | Matched album      |

### Service: `AlbumService`

| RPC  | Request       | Response       | Description                              |
|------|---------------|----------------|------------------------------------------|
| `Read` | `ReadRequest` | `ReadResponse` | Fetch one album by ID from seed data. |

Server implementation: `main.go:62-79` (`func (s *server) Read(...)`).
Lookup helper: `main.go:30-39` (`getAlbum(id string)`).

### Error Handling

- Unknown ID → gRPC status `NotFound` (`codes.NotFound`):
  `Album with ID <id> not found` (`main.go:68`).
- Unimplemented RPCs → `Unimplemented` (from
  `UnimplementedAlbumServiceServer` in `common/album_grpc.pb.go:63-67`).

---

## Client Examples

### grpcurl

Requires reflection (default when `ENV != production`).

```bash
# list services
grpcurl -plaintext localhost:50051 list
# → album.AlbumService
# → grpc.reflection.v1.ServerReflection

# list methods
grpcurl -plaintext localhost:50051 list album.AlbumService
# → album.AlbumService.Read

# describe types
grpcurl -plaintext localhost:50051 describe album.ReadRequest
grpcurl -plaintext localhost:50051 describe album.Album

# call Read (seed IDs "1", "2", "3")
grpcurl -plaintext -d '{"id": "1"}' \
  localhost:50051 album.AlbumService/Read
```

Expected response (pretty-printed):

```json
{
  "album": {
    "id": "1",
    "title": "Blue Train",
    "artist": "John Coltrane",
    "price": 56.99
  }
}
```

Error case:

```bash
grpcurl -plaintext -d '{"id": "999"}' \
  localhost:50051 album.AlbumService/Read
# ERROR: Code: NotFound, Message: Album with ID 999 not found
```

### grpcui / Postman

```bash
grpcui -plaintext localhost:50051
# open http://127.0.0.1:port, select album.AlbumService → Read
```

In Postman: New **gRPC request** → URL `localhost:50051` (plaintext, no TLS)
→ import via **server reflection** or point at `proto/album.proto` →
call `Read` with `{"id": "2"}`.

### Go Client

```go
package main

import (
    "context"
    "log"
    "time"

    pb "github.com/ravirajsubramanian/metering/common"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    conn, err := grpc.NewClient("localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := pb.NewAlbumServiceClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    res, err := client.Read(ctx, &pb.ReadRequest{Id: "1"})
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("album: %+v", res.GetAlbum())
}
```

---

## REST / Gin Component (`cmd/lease`)

`cmd/lease/lease.go` is **not yet wired into `main()`** — it's a stub for a
future HTTP lease/metering API.

Contents:

- `Print()` — prints `"Lease"` (smoke-test helper).
- `GetAlbums(c *gin.Context)` — `c.IndentedJSON(http.StatusOK, albums)`.
- Local `album` struct + same 3-record seed slice.

Typical future wiring (not yet implemented):

```go
router := gin.Default()
router.GET("/albums", lease.GetAlbums)
router.Run(":8080")
```

---

## Configuration

| Env var | Default | Effect                                    |
|---------|---------|-------------------------------------------|
| `ENV`   | `""`    | `ENV=production` disables gRPC reflection |

No config files, flags, database, or auth yet. Port `:50051` is hardcoded in
`main.go:82`.

---

## Development Guide

- Entry point: `main.go:81-100`.
- Server state: `type server struct` (`main.go:41-45`) holds
  `sync.RWMutex` + `map[string]*pb.Album` (map is initialized but the current
  `Read` path uses the package-level `albums` slice via `getAlbum`).
- Generated code: do not hand-edit `common/*.pb.go`; edit
  `proto/album.proto` and regenerate.
- `internal/` is empty — suggested homes:
  `internal/store/`, `internal/service/`, `internal/meter/`, `internal/api/`.
- `cmd/lease/` suggests a multi-binary layout (`cmd/<svc>/main.go`); consider
  moving the gRPC `main.go` to `cmd/album/main.go` and leaving repo-root
  `main.go` as a thin launcher.

Code style: `gofmt` + `go vet`:

```bash
gofmt -l .
go vet ./...
```

---

## Testing

No test files are checked in yet. Suggested starting points:

```bash
# once tests exist
go test ./...
go test -run TestRead -v ./...
```

Recommended first tests:

1. `getAlbum("1")` returns Blue Train; unknown ID returns not-found.
2. `server.Read` with `bufconn` returns expected `ReadResponse` and
   `codes.NotFound` on missing ID.
3. `lease.GetAlbums` returns HTTP 200 + 3-element JSON array (via
   `httptest.NewRecorder` + Gin test context).

---

## Troubleshooting

| Symptom | Cause / Fix |
|---------|-------------|
| `grpcurl ... list` shows nothing / connection refused | Server not running; check `go run .` output and `lsof -i :50051`. |
| `failed to listen: address already in use` | Another process on `:50051`; kill it or change the port in `main.go:82`. |
| Reflection methods missing in production | Expected: reflection is skipped when `ENV=production`. Unset `ENV` for local dev. |
| `protoc: command not found` / plugin errors | Install `protoc` + `protoc-gen-go` / `protoc-gen-go-grpc`, ensure `$(go env GOPATH)/bin` is on `PATH`. |
| `go_package` import warnings | `option go_package = "/common"` is a short path; switch to the full module path (see Regenerating section). |
| Build errors mentioning `uuid`, `CreateRequest`, `CreateResponse` | `main.go:47-60` contains a `Create` stub referencing types not in `proto/album.proto` and a missing `uuid` import. Either add a `Create` RPC to the proto + `github.com/google/uuid` dependency, or remove the stub until the API is designed. |

---

## Roadmap

- [ ] Fix or remove the non-compiling `Create` stub in `main.go:47-60`.
- [ ] Add `Create / List / Update / Delete` RPCs to `proto/album.proto` as needed.
- [ ] Replace slice lookup with the mutex-guarded map (and use `RLock` in `Read`).
- [ ] Return `exists=false` for unknown IDs instead of always `true`.
- [ ] Wire `cmd/lease` Gin handlers into an HTTP server / grpc-gateway.
- [ ] Add persistence (Postgres/SQLite) and proper metering/billing domain logic in `internal/`.
- [ ] Add `go test` coverage with `bufconn`, `httptest`, and CI (`go vet`, `gofmt`, `go test`).
- [ ] Add `Dockerfile`, `Makefile`, `.golangci.yml`, and GitHub Actions.
- [ ] Use full `go_package` import path andBuf / versioned API (`v1/`).

---

## License

No license file is currently present. If open-sourcing, add one (e.g. MIT,
Apache-2.0) as `LICENSE`, otherwise all rights reserved by default.
