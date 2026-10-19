# Revirathya - Finance Core

## Overview
Core services to managing financial data. Exposes gRPC and GraphQL as APIs backed by Postgres.

**Stack:**
- Go 1.26
- pgx
- gRPC
- gqlgen (GraphQL)
- sqlc
- golang-migrate

## Prerequisites
| Tool | Used by | Required? |
|:--|:--|:--|
| Go 1.26 | `run`, `build`, `test`, `vet`, `gql-generate` | Yes |
| GNU Make | all `make` commands | Yes |
| Docker with Compose v2 (`docker compose`) | `docker-*` | Yes |
| `migrate` (golang-migrate, with Postgres driver) | `migrate-*` | Yes, for local mode |
| `sqlc` | `sqlc-generate` | When editing SQL queries |
| `buf` + `protoc-gen-go` + `protoc-gen-go-grpc` | `proto-generate`, `proto-lint`, `check` | When editing `.proto` files, and for `make check` |
| `golangci-lint` v2 | `lint`, `check` | For `make lint` / `make check` (CI runs it on every PR) |
| `grpcurl` | `grpc-list`, `grpc-describe` | Optional |

`gqlgen` needs no install: it's declared as a Go tool in `go.mod` and runs through `go run`.

### Installing the Go-based tools

All of these install with `go install`, into `$(go env GOPATH)/bin` (usually `~/go/bin`). Versions match the
ones CI uses and the committed generated code.

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

5. Install `golangci-lint` (the project recommends the binary over `go install`):

   ```sh
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.15.0
   ```

6. Optional: install `grpcurl`:

   ```sh
   go install github.com/fullstorydev/grpcurl/cmd/grpcurl@v1.9.1
   ```

7. Check everything is found:

   ```sh
   buf --version && protoc-gen-go --version && protoc-gen-go-grpc --version \
     && sqlc version && migrate -version && golangci-lint version && grpcurl --version
   ```

Docker and Make come from your OS: Docker Desktop (macOS/Windows) or Docker Engine with the Compose plugin
(Linux); Make via `xcode-select --install` (macOS) or `sudo apt install make` (Debian/Ubuntu).

## Quickstart

```sh
cp .env.example .env # After copy, set the Required values (see Configuration)
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
make docker-build svc=migration   # Build the migration image (revirathya-finance-migration:$MIGRATION_VERSION)
make docker-up                    # Start db -> migration -> core
```

In Docker mode the app is reachable on the host ports (gRPC `:51051`, GraphQL `:8180/graphql` by default).

The Docker files live in `infra/docker/` (`Dockerfile`, `migration.Dockerfile`, `docker-compose.yaml`). The Makefile
runs compose with `--project-directory .`, so it reads the root `.env` and keeps the project name
`revirathya-finance-core` (and its `pgdata` volume). To run compose by hand, use the same flags:

```sh
docker compose --project-directory . -f infra/docker/docker-compose.yaml ps
```

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
|`svc`|`docker-build`|`core`|Image to build: `core` (`infra/docker/Dockerfile`) or `migration` (`infra/docker/migration.Dockerfile`)|
|`svc`|`docker-up`, `docker-logs`|all services|Service(s) to start or follow, e.g. `svc=db` or `svc="db migration"`|
|`VERSION`|`docker-build`, `docker-up`|from `.env`|Tag of the `core` image; override for one-off builds, e.g. `make docker-build VERSION=0.0.2`. Pass the same value to `docker-up` so compose runs the image you built.|
|`MIGRATION_VERSION`|`docker-build svc=migration`, `docker-up`|from `.env`|Tag of the `migration` image; override the same way, e.g. `make docker-build svc=migration MIGRATION_VERSION=0.0.2`.|
|`args`|`docker-down`|none|Extra flags for `docker compose down`, e.g. `args="-v --remove-orphans"`|

## Configuration

Copy `.env.example` to `.env`. Only the required variables must be set; prefer the defaults for everything else.

### Required

|Var|Description|
|:--|:--|
|`VERSION`|Image tag for `core` (e.g. `0.0.1`). Used by `make docker-build` and docker-compose. Compose checks it for every command, even `make docker-up svc=db`.|
|`MIGRATION_VERSION`|Image tag for `migration` (e.g. `0.0.1`), versioned separately because the image only changes when `migrations/` does. Used by `make docker-build svc=migration` and docker-compose, which also checks it for every command.|
|`DB_PASSWORD`|Postgres password, used by the `db` container and the Docker-mode connection string|
|`DATABASE_URI`|Postgres DSN for **local mode** (`make run`, `make migrate-*`), e.g. `postgres://revirathya:<password>@localhost:5532/finance?sslmode=disable`. Its port must match `DB_HOST_PORT`. Docker mode ignores it and builds its own DSN pointing at `db:5432`.|
|`GRPC_API_KEY`|Shared secret the Telegram bot sends on every gRPC call (`authorization: Bearer <key>`). Generate with `openssl rand -base64 32`.|
|`GOOGLE_CLIENT_ID`|OAuth client ID of the Google Cloud OAuth client. Only Google ID tokens issued for this client are accepted by `loginWithGoogle`.|
|`JWT_SECRET`|Secret for signing session tokens, at least 32 bytes. Generate with `openssl rand -base64 32`. Changing it logs everyone out.|

### Optional (defaults shown)

|Var|Default|Description|
|:--|:--|:--|
|`DB_USER`|`revirathya`|Postgres user|
|`DB_NAME`|`finance`|Postgres database name|
|`GRPC_PORT`|`50051`|gRPC listen port (plain number, no `:`), local mode only|
|`GQL_PORT`|`8080`|GraphQL listen port (plain number, no `:`), local mode only|
|`ALLOWED_ORIGINS`|`http://localhost,http://127.0.0.1`|Comma-separated CORS origins for GraphQL. Must be full origins including scheme and port, e.g. `http://localhost:3000`|
|`JWT_TTL`|`24h`|Session token lifetime, as a Go duration (e.g. `30m`, `12h`)|
|`GRPC_ENABLE_REFLECTION`|`false`|Register gRPC server reflection, so `grpcurl` and Postman can discover services without proto files. Keep it off in production.|
|`LOG_FORMAT`|`json`|Log output format: `json` (one JSON object per line, for log platforms) or `text` (readable in a terminal; handy in your local `.env`)|
|`LOG_LEVEL`|`info`|Minimum level logged: `debug`, `info`, `warn` or `error`|
|`GQL_ENABLE_INTROSPECTION`|`false`|Allow GraphQL introspection (schema download) and "Did you mean …?" hints in validation errors. Keep it off in production.|

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
- Docker: built from `infra/docker/migration.Dockerfile` (`migrate/migrate` + bundled `migrations`); the `migration`
  service runs `up` automatically before `core` starts.

## Authentication

Every request is tied to an authenticated user, and all data (categories, expenses, Telegram context) is
scoped to that user. The user is never taken from a request argument; another user's records return
`NOT_FOUND`.

### GraphQL (Web Console)

1. The client signs in with Google and sends the Google ID token to `loginWithGoogle(idToken:)`.
   On first login the user is created from the Google profile.
2. The mutation returns a **session token** (JWT, valid for `JWT_TTL`).
3. Every other request sends `Authorization: Bearer <session token>`. Requests without a valid token get
   `UNAUTHENTICATED`.

To get a session token from Postman without the Web Console, see
[docs/postman-google-login.md](docs/postman-google-login.md).

### gRPC (Telegram bot)

Every call must send two metadata entries:

|Metadata|Value|
|:--|:--|
|`authorization`|`Bearer <GRPC_API_KEY>`: identifies the bot itself|
|`x-telegram-user-id`|The Telegram user ID of the message sender (`message.from.id`): identifies the user|

| Situation | Result |
|:--|:--|
| Missing or wrong API key | `UNAUTHENTICATED`, before anything else runs |
| Missing or malformed `x-telegram-user-id` (must be a positive integer) | `UNAUTHENTICATED` |
| Telegram account not bound to a user | `UNAUTHENTICATED` for every RPC except `BindTelegramAccount` |
| Bound account | The call runs as the bound user |

### Telegram binding

A user links their Telegram account once, from the Web Console:

1. In the console (logged in), `createTelegramBindCode` returns a one-time code like `HE2H-KMDX`,
   valid for 10 minutes. Creating a new code replaces the previous one.
2. The user sends `/bind HE2H-KMDX` to the bot in a **private chat**.
3. The bot calls `BindTelegramAccount` with the code. The Telegram account is now bound, and the bot's
   conversation state is reset.

| `BindTelegramAccount` error | Meaning |
|:--|:--|
| `INVALID_ARGUMENT` | The code is malformed (must be exactly `XXXX-XXXX`, uppercase), unknown, expired or already used |
| `ALREADY_EXISTS` | This Telegram account is already bound to another user; unbind it there first |
| `RESOURCE_EXHAUSTED` | Too many attempts: at most 10 per Telegram account per 5 minutes |

Only a hash of each code is stored. `unbindTelegram` (GraphQL) removes the binding; binding again from a
different Telegram account replaces it. The attempt limit is kept in memory, so it is per process and resets
on restart.

## GraphQL server

`/graphql` accepts only `POST` requests with a JSON body. `GET`, websockets (subscriptions), multipart uploads and
automatic persisted queries are disabled.

|Limit|Value|When exceeded|
|:--|:--|:--|
|Request body|1 MB|HTTP `413`|
|Query size|5000 tokens|`GRAPHQL_PARSE_FAILED`|
|Query complexity|200 (each field counts 1)|`COMPLEXITY_LIMIT_EXCEEDED`|
|Request duration|10 seconds|HTTP `503` with `request timeout`|
|Malformed request (e.g. invalid JSON)|—|`BAD_REQUEST` / "invalid request"|

Introspection is off unless `GQL_ENABLE_INTROSPECTION=true`. With it off, tools can't download the schema from
the server (e.g. Postman's "fetch schema", GraphQL codegen); point them at the committed schema in
`graphql/schema/*.graphql` instead, or enable it in development.

## gRPC server

|Limit|Value|When exceeded|
|:--|:--|:--|
|Request message|1 MB|`RESOURCE_EXHAUSTED`|
|Call duration|10 seconds, when the client sets no deadline (a client deadline is kept as is)|`DEADLINE_EXCEEDED` / "request timeout"|
|Concurrent calls per connection|100|Further calls wait|
|Connection handshake|10 seconds|Connection closed|
|Idle connection|5 minutes without calls|Connection closed (clients reconnect automatically)|
|Client keepalive pings|At most one every 30 seconds|Connection closed with `GOAWAY` (`too_many_pings`)|

If the Telegram bot enables client keepalive, its ping interval must be at least 30 seconds.

## Database connections

The connection pool is configured in code (`internal/platform/database/db.go`), and overrides `pool_*` parameters
in `DATABASE_URI`: at most 10 connections (1 kept open), each replaced after 1 hour or 30 minutes idle, 5-second
connect timeout. Every session runs with `statement_timeout = 10s`, so Postgres cancels any statement running
longer, and `application_name = financecore` (visible in `pg_stat_activity`).

## Input validation

Both APIs validate input the same way, after trimming surrounding whitespace from text:

|Field|Rule|
|:--|:--|
|Category name, first name|Required, at most 100 characters|
|Last name|Optional, at most 100 characters|
|Expense amount|Greater than 0, at most 2 decimal places, below 1,000,000,000,000|
|Expense date|From 2000-01-01 to tomorrow (Asia/Jakarta)|
|Expense description|Optional, at most 500 characters. Never `null` in responses (`""` when there is none). On update, omit it to keep the current one, or send `""` to clear it|
|Telegram context fields|Optional, at most 100 characters each|

Invalid input returns `INVALID_ARGUMENT` (gRPC) or `BAD_USER_INPUT` (GraphQL) with a message naming the field,
e.g. `amount: must be greater than 0`. Lengths count characters, not bytes.

## Request IDs

Every request gets a request ID, added to the logs for that request so they can be found together. Send
your own to correlate with your logs, or let the service generate one:

|API|Request|Response|
|:--|:--|:--|
|GraphQL / HTTP|`X-Request-ID` header (optional)|`X-Request-ID` header|
|gRPC|`x-request-id` metadata (optional)|`x-request-id` response header|

Each request also produces one access log line (method, path or gRPC method, status, duration, request ID).
Query strings, headers and bodies are never logged.

An incoming ID is kept only if it is at most 64 characters of letters, digits, `-`, `_` and `.`; anything else is
replaced with a new UUID. Browsers can send and read `X-Request-ID` through CORS.

## Health checks

|Endpoint|Meaning|
|:--|:--|
|`GET /healthz` (HTTP)|Liveness: always `200` while the process runs|
|`GET /readyz` (HTTP)|Readiness: `200` when the database answers within 2 seconds, `503` otherwise|
|`grpc.health.v1.Health/Check` (gRPC)|`SERVING` while the process runs, `NOT_SERVING` during shutdown|

None of them require authentication. In Docker mode, `core` reports its health through `/readyz`
(`docker compose ps` shows `healthy`). The runtime image has no shell or `curl`, so the check runs the binary
itself: `financecore healthcheck` exits `0` when `/readyz` returns `200`, and `1` otherwise.

```sh
curl -i localhost:8080/readyz
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check   # needs GRPC_ENABLE_REFLECTION=true
```

## gRPC tools

`make grpc-list` and `make grpc-describe` read the local proto files, so they work without a running server:

```sh
make grpc-list
make grpc-describe symbol=financecore.v1.CategoryService
```

To call RPCs, set `GRPC_ENABLE_REFLECTION=true` and send both metadata entries, e.g. (after
`set -a; source .env; set +a` to load `.env` into your shell):

```sh
grpcurl -plaintext \
  -H "authorization: Bearer $GRPC_API_KEY" \
  -H "x-telegram-user-id: 123456789" \
  localhost:50051 financecore.v1.CategoryService/ListCategories   # Docker mode: localhost:51051
```

## Development

```sh
make run               # run the app locally (go run ./cmd)
make build             # build the binary to bin/financecore
make proto-generate    # buf: gRPC stubs        -> gen/go
make sqlc-generate     # sqlc: query code       -> gen/sqlc  (queries in sqlc/queries)
make gql-generate      # gqlgen: GraphQL code   -> gen/graphql
make lint              # golangci-lint (config: .golangci.yaml)
make check             # vet + lint + test + proto-lint
```

All generated code lives in `gen/` and is committed; CI fails if it's out of date.

### Tests

```sh
make test              # unit tests (go test ./test/...)
make test-integration  # all tests, including integration tests against a real Postgres (needs Docker)
```

Tests live in `test/`, mirroring the `internal/` package they cover (e.g. `test/internal/usecase/category` tests
`internal/usecase/category`); shared helpers are in `testutil/`. CI also runs the unit tests with `-race`
(`go test -race ./test/...`). Integration tests carry the `integration` build tag; each package's `TestMain` calls `testutil.RunWithPostgres`, which starts a throwaway Postgres container,
applies `migrations/` and provides `testutil.TestPool`. They never touch your dev database.

To get editor support in integration test files, add the tag to gopls (e.g. `.vscode/settings.json`):
`"gopls": { "buildFlags": ["-tags=integration"] }`.

## Changelog

See [CHANGELOG.md](CHANGELOG.md).
