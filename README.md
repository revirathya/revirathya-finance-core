# Revirathya - Finance Core

## Overview
Core services to managing financial data. Exposes gRPC and GraphQL as APIs backed by Postgres.

**Stack:**
- Go 1.25
- pgx
- gRPC
- gqlgen (GraphQL)
- sqlc
- golang-migrate

## Prerequisites
| Tool | Used by | Required? |
|:--|:--|:--|
| Go 1.25 | `run`, `build`, `test`, `vet`, `gql-generate` | Yes |
| GNU Make | all `make` commands | Yes |
| Docker with Compose v2 (`docker compose`) | `docker-*` | Yes |
| `migrate` (golang-migrate, with Postgres driver) | `migrate-*` | Yes, for local mode |
| `sqlc` | `sqlc-generate` | When editing SQL queries |
| `buf` + `protoc-gen-go` + `protoc-gen-go-grpc` | `proto-generate`, `proto-lint`, `check` | When editing `.proto` files, and for `make check` |
| `grpcurl` | `grpc-list`, `grpc-describe` | Optional |

`gqlgen` needs no install: it's declared as a Go tool in `go.mod` and runs through `go run`.

### Installing the Go-based tools

All of these install with `go install`, into `$(go env GOPATH)/bin` (usually `~/go/bin`). Versions match the
ones used in the `Dockerfile` and in the committed generated code.

1. Make sure the Go bin directory is on your `PATH` (add this line to `~/.bashrc` or `~/.zshrc`):

   ```sh
   export PATH="$PATH:$(go env GOPATH)/bin"
   ```

2. Install `buf` and the two protoc plugins it calls (`buf.gen.yaml` uses them as local plugins):

   ```sh
   go install github.com/bufbuild/buf/cmd/buf@v1.72.0
   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
   ```

3. Install `sqlc`:

   ```sh
   go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
   ```

4. Install `migrate` with the Postgres driver (the `postgres` build tag is required, or it can't connect):

   ```sh
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1
   ```

5. Optional: install `grpcurl`:

   ```sh
   go install github.com/fullstorydev/grpcurl/cmd/grpcurl@v1.9.1
   ```

6. Check everything is found:

   ```sh
   buf --version && protoc-gen-go --version && protoc-gen-go-grpc --version \
     && sqlc version && migrate -version && grpcurl --version
   ```

Docker and Make come from your OS: Docker Desktop (macOS/Windows) or Docker Engine with the Compose plugin
(Linux); Make via `xcode-select --install` (macOS) or `sudo apt install make` (Debian/Ubuntu).

## Quickstart

```sh
cp .env.example .env # After copy, set VERSION, DB_PASSWORD and DATABASE_URI
```

There are two ways to run locally:

**Local mode:** Postgres in Docker, app on your machine (`go run`).

```sh
make docker-up svc=db   # Start Postgres only
make migrate-up         # Apply migrations
make run                # Run the app (gRPC :50051, GraphQL :8080/graphql)
```

**Docker mode:** everything in Docker (Postgres, migrations, app).

```sh
make docker-build                 # Build the app image (revirathya-finance-core:$VERSION)
make docker-build svc=migration   # Build the migration image (revirathya-finance-migration:$VERSION)
make docker-up                    # Start db -> migration -> core
```

In Docker mode the app is reachable on the host ports (gRPC `:51051`, GraphQL `:8180/graphql` by default).

Common commands for both modes:

```sh
make docker-logs                          # Follow logs of all services
make docker-logs svc=core                 # Follow logs of one service
make docker-down                          # Stop services
make docker-down args="-v"                # Stop services and delete the Postgres volume
make docker-down args="--remove-orphans"  # Any extra `docker compose down` flags go in args
```

Make treats anything starting with `-` as its own option, so `make docker-down -v` does not work; pass flags
through `args="..."` instead.

### Make variables for Docker targets

|Variable|Used by|Default|Description|
|:--|:--|:--|:--|
|`svc`|`docker-build`|`core`|Image to build: `core` (`Dockerfile`) or `migration` (`migration.Dockerfile`)|
|`svc`|`docker-up`, `docker-logs`|all services|Service(s) to start or follow, e.g. `svc=db` or `svc="db migration"`|
|`VERSION`|`docker-build`, `docker-up`|from `.env`|Image tag; override for one-off builds, e.g. `make docker-build VERSION=0.0.2`. Pass the same value to `docker-up` so compose runs the image you built.|
|`args`|`docker-down`|none|Extra flags for `docker compose down`, e.g. `args="-v --remove-orphans"`|

## Configuration

Copy `.env.example` to `.env`. Only the required variables must be set; prefer the defaults for everything else.

### Required

|Var|Description|
|:--|:--|
|`VERSION`|Image tag for `core` and `migration` (e.g. `0.0.1`). Used by `make docker-build` and docker-compose. Compose checks it for every command, even `make docker-up svc=db`.|
|`DB_PASSWORD`|Postgres password, used by the `db` container and the Docker-mode connection string|
|`DATABASE_URI`|Postgres DSN for **local mode** (`make run`, `make migrate-*`), e.g. `postgres://revirathya:<password>@localhost:5532/finance?sslmode=disable`. Its port must match `DB_HOST_PORT`. Docker mode ignores it and builds its own DSN pointing at `db:5432`.|

### Optional (defaults shown)

|Var|Default|Description|
|:--|:--|:--|
|`DB_USER`|`revirathya`|Postgres user|
|`DB_NAME`|`finance`|Postgres database name|
|`GRPC_PORT`|`50051`|gRPC listen port (plain number, no `:`), local mode only|
|`GQL_PORT`|`8080`|GraphQL listen port (plain number, no `:`), local mode only|
|`ALLOWED_ORIGINS`|`http://localhost,http://127.0.0.1`|Comma-separated CORS origins for GraphQL. Must be full origins including scheme and port, e.g. `http://localhost:3000`|

### Optional: host ports for Docker mode

These control which ports on your machine map to the containers. The ports inside the containers are fixed
(Postgres `5432`, gRPC `50051`, GraphQL `8080`), so `GRPC_PORT` / `GQL_PORT` have no effect in Docker mode.

|Var|Default|Description|
|:--|:--|:--|
|`DB_HOST_PORT`|`5532`|Host port for Postgres|
|`GRPC_HOST_PORT`|`51051`|Host port for gRPC|
|`GQL_HOST_PORT`|`8180`|Host port for GraphQL|

## Migrations

- Local: `make migrate-up` / `make migrate-down` / `make migrate-create name=<X>` / `make migrate-version`
- Docker: built from `migration.Dockerfile` (`migrate/migrate` + bundled `migrations`); the `migration`
  service runs `up` automatically before `core` starts.

## gRPC tools

Server reflection isn't registered, so `grpcurl` is called with the proto files explicitly:

```sh
make grpc-list                                   # Local mode (localhost:50051)
make grpc-list GRPC_ADDR=localhost:51051         # Docker mode
make grpc-describe symbol=financecore.v1.CategoryService
```

## GraphQL authentication

GraphQL requests are authenticated with a session token (`Authorization: Bearer <token>`), obtained from the
`loginWithGoogle` mutation with a Google ID token. To get one from Postman without the Web Console, see
[docs/postman-google-login.md](docs/postman-google-login.md).

## Development

```sh
make proto-generate # buf: gRPC stubs
make sqlc-generate  # sqlc: query code
make gql-generate   # gqlgen: GraphQL resolvers
make check          # vet + test + proto-lint
```

## Changelog

See [CHANGELOG.md](CHANGELOG.md).
