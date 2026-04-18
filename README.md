# grpc_auth

Minimal gRPC-based SSO/auth service with SQLite storage and JWT issuance.

## Overview

- gRPC service: register, login, and admin check
- SQLite storage with migrations
- JWT signed per app secret
- Self-contained integration tests

## Requirements

- Go 1.24+
- SQLite (via go-sqlite3)

## Quickstart (Taskfile)

1) Apply migrations:

```
 task migrate
```

2) Create an app (prints app_id and app_secret):

```
 task app-create
```

3) Run the server:

```
 task run
```

4) Run tests:

```
 task test
```

## Quickstart (manual)

1) Apply migrations:

```
 go run ./cmd/migrator --storage-path ./storage/sso.db --migrations-path ./migrations --migrations-table migrations
```

2) Create an app:

```
 go run ./cmd/app --config ./config/config_local.yaml --name dev-app
```

3) Run the server:

```
 go run ./cmd/sso --config ./config/config_local.yaml
```

## Configuration

Default local config: `config/config_local.yaml`.

You can pass config path via:

- CLI flag: `--config /path/to/config.yaml`
- Environment: `CONFIG_PATH=/path/to/config.yaml`

## gRPC API

Proto definition: `proto/sso/sso.proto`

Service `Auth`:

- `Register(RegisterRequest) -> RegisterResponse`
- `Login(LoginRequest) -> LoginResponse`
- `IsAdmin(IsAdminRequest) -> IsAdminResponse`

Generated code is in `gen/go/sso`.

## Migrations

Migrations live in `migrations/`. The migrator uses:

```
 go run ./cmd/migrator --storage-path <db> --migrations-path ./migrations --migrations-table migrations
```

## App management

Use the admin CLI to create apps and secrets:

```
 go run ./cmd/app --config ./config/config_local.yaml --name my-app
```

If `--secret` is omitted, a random secret is generated.

## Tests

Integration tests are self-contained and start a gRPC server in-process:

```
 go test ./...
```
