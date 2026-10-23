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
make docker-up tracing=1          # Same, plus Jaeger, with core sending traces to it
```

In Docker mode the app is reachable on the host ports (gRPC `:51051`, GraphQL `:8180/graphql`, metrics
`:9900/metrics` by default). The metrics port is published on `127.0.0.1` only, so other machines can't reach it; gRPC and HTTP are published on
all interfaces.

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
|`args`|`docker-up`, `docker-down`|none|Extra flags for `docker compose up` / `down`, e.g. `args="-v --remove-orphans"`. `docker-down` always stops every profile's services, Jaeger included.|
|`tracing`|`docker-up`|off|Any value (`tracing=1`) also starts Jaeger (compose profile `tracing`) and points core at it ([tracing](#tracing))|

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
|`APP_ENV`|`prod`|`dev` or `prod`. Sets the defaults for dev-only features: in `dev`, gRPC reflection, GraphQL introspection and the GraphQL playground (`/playground`) are on. Any other value is a startup error.|
|`DB_USER`|`revirathya`|Postgres user|
|`DB_NAME`|`finance`|Postgres database name|
|`GRPC_PORT`|`50051`|gRPC listen port (plain number, no `:`), local mode only|
|`HTTP_PORT`|`8080`|HTTP listen port for GraphQL (`/graphql`) and health checks (plain number, no `:`), local mode only.|
|`METRICS_PORT`|`9090`|Listen port for Prometheus metrics (`/metrics`), local mode only. Keep it internal.|
|`ALLOWED_ORIGINS`|`http://localhost,http://127.0.0.1`|Comma-separated CORS origins for GraphQL. Must be full origins including scheme and port, e.g. `http://localhost:3000`|
|`JWT_TTL`|`24h`|Session token lifetime, as a Go duration (e.g. `30m`, `12h`)|
|`GRPC_ENABLE_REFLECTION`|`false` (`true` in `dev`)|Register gRPC server reflection, so `grpcurl` and Postman can discover services without proto files. Keep it off in production.|
|`LOG_FORMAT`|`json`|Log output format: `json` (one JSON object per line, for log platforms) or `text` (readable in a terminal; handy in your local `.env`)|
|`LOG_LEVEL`|`info`|Minimum level logged: `debug`, `info`, `warn` or `error`|
|`GQL_ENABLE_INTROSPECTION`|`false` (`true` in `dev`)|Allow GraphQL introspection (schema download) and "Did you mean …?" hints in validation errors. Keep it off in production.|
|`WRITE_RATE_LIMIT`|`15`|Writes allowed per user per `WRITE_RATE_WINDOW`, gRPC and GraphQL together ([rate limiting](docs/rate-limiting.md)). Positive integer.|
|`WRITE_RATE_WINDOW`|`1m`|Length of the write-limit window, as a Go duration. Positive.|
|`OTEL_EXPORTER_OTLP_ENDPOINT`|empty (tracing off)|OTLP gRPC collector to send traces to, e.g. `http://localhost:4317` (local mode) or `http://jaeger:4317` (Docker mode). `http://` means no TLS. See [Tracing](#tracing).|
|`OTEL_TRACES_SAMPLER`|`parentbased_always_on`|Which traces are kept, e.g. `parentbased_traceidratio` to keep a fraction (read by the OpenTelemetry SDK).|
|`OTEL_TRACES_SAMPLER_ARG`|none|Argument for the ratio samplers: the fraction to keep, e.g. `0.1`.|
|`OTEL_SERVICE_NAME`|`financecore`|Service name on every span.|
|`OTEL_RESOURCE_ATTRIBUTES`|none|Extra span resource attributes, e.g. `service.version=0.0.1`.|

### Optional: host ports for Docker mode

These control which ports on your machine map to the containers. The ports inside the containers are fixed
(Postgres `5432`, gRPC `50051`, HTTP `8080`, metrics `9090`), so `GRPC_PORT` / `HTTP_PORT` / `METRICS_PORT` have no
effect in Docker mode.

|Var|Default|Description|
|:--|:--|:--|
|`DB_HOST_PORT`|`5532`|Host port for Postgres|
|`GRPC_HOST_PORT`|`51051`|Host port for gRPC|
|`HTTP_HOST_PORT`|`8180`|Host port for HTTP (GraphQL, health checks).|
|`METRICS_HOST_PORT`|`9900`|Host port for metrics|

## Migrations

- Local: `make migrate-up` / `make migrate-down` / `make migrate-create name=<X>` / `make migrate-version`
- Docker: built from `infra/docker/migration.Dockerfile` (`migrate/migrate` + bundled `migrations`); the `migration`
  service runs `up` automatically before `core` starts.

## Integrating a new service

A checklist for a new client. Each point links to the section with the details.

|Topic|gRPC (`:50051`)|GraphQL (`POST /graphql` on `:8080`)|
|:--|:--|:--|
|**Auth**|`authorization: Bearer <GRPC_API_KEY>` plus `x-telegram-user-id` of a bound Telegram account on every call ([gRPC auth](#grpc-telegram-bot), [binding](#telegram-binding))|`Authorization: Bearer <session token>` from `loginWithGoogle` ([GraphQL auth](#graphql-web-console))|
|**Contract**|`proto/financecore/v1/*.proto`; Go clients import `gen/grpc/financecore/v1` ([messages](#grpc-messages), [tools](#grpc-tools))|`graphql/schema/*.graphql`; explore with the [playground](#playground-app_envdev-only) in dev|
|**Types**|`google.type.Date`, `google.type.Decimal`, IDs as UUID strings|`Date`, `Decimal`, `UUID` scalars, all JSON strings ([scalars](#custom-scalars))|
|**Errors**|Status codes; validation is `INVALID_ARGUMENT` with `field: reason`; another user's record is `NOT_FOUND`; over the write limit is `RESOURCE_EXHAUSTED`|`extensions.code`: `BAD_USER_INPUT` (`field: reason`), `NOT_FOUND`, `UNAUTHENTICATED`, `CONFLICT`, `RATE_LIMITED`, `INTERNAL`|
|**Lists**|`page_size` / `page_token` → `next_page_token` ([pagination](#pagination))|`first` / `after` → `pageInfo` ([pagination](#pagination))|
|**Updates**|Patch: unset fields are kept ([updates](#updates))|Patch: omitted or `null` fields are kept ([updates](#updates))|
|**Limits**|1 MB messages, 10 s default deadline, keepalive pings ≥ 30 s ([gRPC server](#grpc-server)); 15 writes per user per minute, shared with GraphQL: `CreateExpense`, `UpdateExpense`, `DeleteExpense` ([rate limiting](docs/rate-limiting.md)). Over it: wait the `retry-after` header (seconds)|1 MB body, 10 s, 5000 tokens, complexity 200 ([GraphQL server](#graphql-server)); 15 writes per user per minute, shared with gRPC: every mutation except `loginWithGoogle`, each field counted ([rate limiting](docs/rate-limiting.md)). Over it: wait `extensions.retryAfter` (seconds)|
|**Tracing**|Send `x-request-id` to correlate with this service's logs ([request IDs](#request-ids)); send W3C `traceparent` metadata to join your trace ([tracing](#tracing))|Send `X-Request-ID` ([request IDs](#request-ids)); send a W3C `traceparent` header to join your trace ([tracing](#tracing))|
|**Health**|`grpc.health.v1.Health/Check` ([health checks](#health-checks))|`GET /healthz`, `GET /readyz`|

Retries: reads, updates and deletes are safe to retry (a retried delete that already succeeded returns `NOT_FOUND`).
For creating expenses, send a UUID generated **once per user
action** and reused on every retry: `request_id` on gRPC `CreateExpense`, `requestId` on GraphQL `createExpense`. A
retry with the same key returns the expense the first call created instead of creating another; reusing a key for a
different expense is `INVALID_ARGUMENT` / `BAD_USER_INPUT` (`request_id: was already used for a different expense`).
Keys are per user. Derive the key from something stable (e.g. `uuid.NewSHA1(namespace, chatID + ":" + messageID)`
for a Telegram message) so it survives a client restart.

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

### Custom scalars

|Scalar|Format|Used for|
|:--|:--|:--|
|`Date`|String, `YYYY-MM-DD` (e.g. `"2026-10-17"`)|`transactionDt`, and the `expenses(transactionDt:)` filter|
|`Decimal`|String, e.g. `"12500.50"`; never a JSON number, so no precision is lost|`amount`|
|`UUID`|String, e.g. `"9b2f6c1e-4a7d-4c3e-8f1a-2d5e6b7c8a90"`|Every `id`, `userId`, `categoryId`, the `id` arguments, and what `delete*` returns|

All three travel as JSON strings. Variables must be declared with the scalar type (`$dt: Date`, not `$dt: String`;
`$id: UUID!`, not `$id: ID!`), or the query fails with `GRAPHQL_VALIDATION_FAILED`. A value in the wrong format, or a JSON number for a `Decimal`, returns
`BAD_USER_INPUT` with a message naming the scalar.

Introspection is off unless `GQL_ENABLE_INTROSPECTION=true` (the default with `APP_ENV=dev`). With it off, tools
can't download the schema from the server (e.g. Postman's "fetch schema", GraphQL codegen); point them at the
committed schema in `graphql/schema/*.graphql` instead, or enable it in development.

### Playground (`APP_ENV=dev` only)

With `APP_ENV=dev`, `GET /playground` serves GraphiQL against `/graphql`, with the schema's descriptions in its docs
panel. To call authenticated operations, put a session token in its headers panel:
`{"Authorization": "Bearer <token>"}` (from `loginWithGoogle`, or see [docs/postman-google-login.md](docs/postman-google-login.md)).
The page loads its scripts from a CDN, and it is never served with `APP_ENV=prod`.

## gRPC server

|Limit|Value|When exceeded|
|:--|:--|:--|
|Request message|1 MB|`RESOURCE_EXHAUSTED`|
|Call duration|10 seconds, when the client sets no deadline (a client deadline is kept as is)|`DEADLINE_EXCEEDED` / "request timeout"|
|Concurrent calls per connection|100|Further calls wait|
|Connection handshake|10 seconds|Connection closed|
|Idle connection|5 minutes without calls|Connection closed (clients reconnect automatically)|
|Client keepalive pings|At most one every 30 seconds|Connection closed with `GOAWAY` (`too_many_pings`)|

Clients that enable keepalive must ping at most every 30 seconds (gRPC client option `keepalive.ClientParameters.Time`
≥ 30s).

## Database connections

The connection pool is configured in code (`internal/platform/database/db.go`), and overrides `pool_*` parameters
in `DATABASE_URI`: at most 10 connections (1 kept open), each replaced after 1 hour or 30 minutes idle, 5-second
connect timeout. Every session runs with `statement_timeout = 10s`, so Postgres cancels any statement running
longer, and `application_name = financecore` (visible in `pg_stat_activity`).

## Metrics

Prometheus metrics are served at `GET /metrics` on their own port (`METRICS_PORT`, default `9090`), not on the
public HTTP port, so they can stay internal. Only `/metrics` is served there.

|Metric|Type|Labels|What it measures|
|:--|:--|:--|:--|
|`grpc_server_started_total`, `grpc_server_handled_total`|counter|`grpc_service`, `grpc_method`, `grpc_type` (+ `grpc_code` on handled)|gRPC calls; every method is exported at zero from startup|
|`grpc_server_handling_seconds`|histogram|`grpc_service`, `grpc_method`, `grpc_type`|gRPC latency|
|`http_requests_total`|counter|`handler` (`graphql`), `code`|`/graphql` requests by HTTP status, including middleware rejections (413, 503)|
|`http_request_duration_seconds`|histogram|`handler`|`/graphql` latency|
|`graphql_errors_total`|counter|`code`|GraphQL errors by `extensions.code` (`BAD_USER_INPUT`, `INTERNAL`, ...)|
|`rate_limited_total`|counter|`api` (`grpc`, `graphql`)|Writes rejected by the per-user write limit|
|`db_pool_connections_max` / `_total` / `_acquired` / `_idle`|gauge|—|Connection pool size and use|
|`db_pool_acquires_total`, `db_pool_acquire_wait_total`, `db_pool_acquire_wait_seconds_total`|counter|—|Connection acquires; a growing wait count means requests queue for a connection|
|`go_*`, `process_*`|various|—|Go runtime and process (memory, goroutines, CPU, open files)|

Labels are bounded on purpose (no GraphQL operation names, no user IDs).

```sh
curl -s localhost:9090/metrics | grep -E '^(grpc_server_handled_total|http_requests_total|db_pool_connections)'
```

## Tracing

OpenTelemetry traces are exported over OTLP gRPC when `OTEL_EXPORTER_OTLP_ENDPOINT` is set; without it, tracing is
off and costs almost nothing. One trace per request:

|Span|From|Name|
|:--|:--|:--|
|gRPC call|`otelgrpc` stats handler|`financecore.v1.ExpenseService/CreateExpense`; attributes `rpc.system.name=grpc`, `rpc.method`, `rpc.response.status_code` (e.g. `NOT_FOUND`)|
|HTTP request|`otelhttp`, outermost handler|the route, `POST /graphql` (unknown paths: just the method); attributes `url.path`, `http.response.status_code`|
|GraphQL|in-repo extension (`graphql/tracing`)|an operation span (`mutation CreateExpense`, or `mutation` without a name) and one span per resolver (`Mutation.createExpense`), marked as errors when they fail; never the query text, variables or argument values|
|SQL query|`otelpgx`|`SELECT`, `INSERT`, ...; the SQL text (`db.query.text`), never the parameters. Only queries inside a request are traced.|

Health checks (`grpc.health.v1`, `/healthz`, `/readyz`) aren't traced. Incoming W3C `traceparent` headers or metadata
are honoured, so a caller's trace continues here. Log lines written during a traced request carry `trace_id` and
`span_id`, so a log entry leads straight to its trace.

Locally, use [Jaeger](https://www.jaegertracing.io/) (compose profile `tracing`). Its UI is at
<http://localhost:16686> and it receives OTLP on `localhost:4317`; both are bound to `127.0.0.1`, and traces are kept
in memory only.

- **Docker mode:** `make docker-up tracing=1` starts Jaeger and points core at `http://jaeger:4317`. Without
  `tracing=1`, Docker mode doesn't trace; it ignores `OTEL_EXPORTER_OTLP_ENDPOINT` from `.env`.
- **Local mode:** `make docker-up svc=jaeger` (naming a service starts it whatever its profile) and set
  `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317` in `.env` for `make run`. In the UI, pick service `financecore`, then filter by operation (span name) or by
tags such as `rpc.system.name=grpc`, `rpc.response.status_code=INTERNAL`, `url.path=/graphql` or `error=true`, or paste
a `trace_id` from a log line into the search box.

## Pagination

Lists are paged and cursor-based: pass the token from one page to get the next. Tokens are opaque; don't build or
parse them. Expenses come newest first (`transaction_dt`, then `id`, descending); categories by name.

|API|Page size|Next page|Last page|
|:--|:--|:--|:--|
|GraphQL `expenses(transactionDt, first, after)`|`first`|`after: pageInfo.endCursor`|`hasNextPage: false`, `endCursor: null`|
|gRPC `ListExpenses`|`page_size`|`page_token: next_page_token`|`next_page_token` empty|
|GraphQL `categories(first, after)`|`first`|`after: pageInfo.endCursor`|`hasNextPage: false`, `endCursor: null`|
|gRPC `ListCategories`|`page_size`|`page_token: next_page_token`|`next_page_token` empty|

The page size defaults to 20 and is capped at 50; a negative size or an invalid token is invalid input. Stop when
there is no next token, not when a page is short. To read every category (e.g. for a selection keyboard), loop with
`page_size: 50` until `next_page_token` is empty.

```graphql
query { expenses(first: 20, after: "MjAyNi0xMC0xN3w5YjJm...") { nodes { id transactionDt amount } pageInfo { hasNextPage endCursor } } }
```

```sh
grpcurl -plaintext -H "authorization: Bearer $GRPC_API_KEY" -H "x-telegram-user-id: 123456789" \
  -d '{"pageSize": 20, "pageToken": "MjAyNi0xMC0xN3w5YjJm..."}' localhost:50051 financecore.v1.ExpenseService/ListExpenses
```

## Updates

Every update is a **patch**: send only the fields to change. A field that is omitted (or `null` in GraphQL) keeps its
current value; `""` clears `description` and `lastName`. An update with no fields returns the object unchanged.

|API|Mutation / RPC|How "omitted" is expressed|
|:--|:--|:--|
|GraphQL|`updateCategory(id, patch)`, `updateExpense(id, patch)`, `updateUser(patch)`|Leave the input field out, or send `null`|
|gRPC|`UpdateExpense`, `UpdateUserTelegramContext`|Leave the field unset (`optional` fields and message fields like `transaction_dt`)|

Fields that are sent are validated the same way as on create.

## Input validation

Both APIs validate input the same way, after trimming surrounding whitespace from text:

|Field|Rule|
|:--|:--|
|Category name, first name|Required, at most 100 characters|
|Last name|Optional, at most 100 characters|
|Expense amount|Greater than 0, at most 2 decimal places, below 1,000,000,000,000|
|Expense date|From 2000-01-01 to tomorrow (Asia/Jakarta)|
|Expense description|Optional, at most 500 characters. Never `null` in responses (`""` when there is none). On update, omit it to keep the current one, or send `""` to clear it (see Updates)|
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
itself: `financecore healthcheck` exits `0` when `/readyz` on `HTTP_PORT` returns `200`, and `1` otherwise. It loads the
same config as the server, so it needs the same environment.

```sh
curl -i localhost:8080/readyz
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check   # needs GRPC_ENABLE_REFLECTION=true
```

## gRPC tools

`make grpc-list` and `make grpc-describe` build the protos with buf (`bin/financecore.binpb`, including the
`google/type` imports) and read that, so they need no running server and no reflection. `describe` shows the doc
comments:

```sh
make grpc-list
make grpc-describe symbol=financecore.v1.ExpenseService
make grpc-describe symbol=financecore.v1.UpdateExpenseRequest
```

To call RPCs, send both metadata entries. Use the same descriptor set (or reflection, which is on with
`APP_ENV=dev`), e.g. after `set -a; source .env; set +a` to load `.env` into your shell:

```sh
grpcurl -plaintext -protoset bin/financecore.binpb \
  -H "authorization: Bearer $GRPC_API_KEY" \
  -H "x-telegram-user-id: 123456789" \
  localhost:50051 financecore.v1.CategoryService/ListCategories   # Docker mode: localhost:51051
```

### gRPC messages

Each resource has one message (`Category`, `Expense`), and every RPC wraps it in its own response, e.g.
`CreateExpenseResponse { expense }` and `ListExpensesResponse { expenses, next_page_token }`. Dates and amounts use
Google's common types:

|Field|Type|JSON form (grpcurl)|
|:--|:--|:--|
|`transaction_dt`|`google.type.Date`|`{"year": 2026, "month": 10, "day": 17}`|
|`amount`|`google.type.Decimal`|`{"value": "12500.50"}`: an exact decimal string, never a float|

A date must be a real calendar date (`2026-02-30` and partial dates with a zero year, month or day are
`INVALID_ARGUMENT`), and a missing or non-numeric amount is `INVALID_ARGUMENT`; the range rules (date from
2000-01-01 to tomorrow, amount > 0 with at most 2 decimals) are the same as GraphQL's.

```sh
grpcurl -plaintext -H "authorization: Bearer $GRPC_API_KEY" -H "x-telegram-user-id: 123456789" \
  -d '{"categoryId": "9b2f6c1e-4a7d-4c3e-8f1a-2d5e6b7c8a90", "transactionDt": {"year": 2026, "month": 10, "day": 17}, "amount": {"value": "12500.50"}}' \
  localhost:50051 financecore.v1.ExpenseService/CreateExpense
```

Go clients import `github.com/revirathya/revirathya-finance-core/gen/grpc/financecore/v1`; the Google types are in
`google.golang.org/genproto/googleapis/type/date` and `.../type/decimal`.

## Development

```sh
make run               # run the app locally (go run ./cmd)
make build             # build the binary to bin/financecore
make proto-generate    # buf: gRPC stubs        -> gen/grpc (googleapis deps pinned in buf.lock)
make sqlc-generate     # sqlc: query code       -> gen/sqlc  (queries in sqlc/queries)
make gql-generate      # gqlgen: GraphQL code   -> gen/graphql
make generate          # all three codegens
make hooks             # once per clone: use the repo's git hooks (.githooks)
make lint              # golangci-lint (config: .golangci.yaml)
make check             # vet + lint + test + proto-lint
```

All generated code lives in `gen/` and is committed; CI fails if it's out of date.

### Git hooks and commit messages

Run `make hooks` once after cloning. It points git at `.githooks/`:

- **pre-commit**: `gofmt` and `go vet` when Go files are staged; when a source-of-truth file is staged (`proto/`,
  `graphql/schema/`, `sqlc/queries/`, `migrations/`, codegen config), it runs `make generate` and fails if `gen/`
  changed, until you stage the regenerated code. Needs the codegen tools; `SKIP_CODEGEN=1 git commit …` skips that
  part (CI still checks it).
- **commit-msg**: every commit message starts with a [Conventional Commits](https://www.conventionalcommits.org/)
  type: `type(scope): summary`, e.g. `feat(expense): add request_id` or `fix: handle nil description`. Types: `feat`,
  `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `build`, `perf`, `revert`; `!` after the type marks a breaking
  change.

PRs are squash-merged and CI checks the PR title against the same format, since it becomes the commit on `main`.
Merging into `main` needs a PR, squash merge, all four CI checks passing (`test`, `migration-test`, `codegen`,
`pr-title`; nobody can bypass them), and an approval from a code owner (`.github/CODEOWNERS`:
`@revirathya/finance-core`). Admins may skip only the approval, while the team has a single member.

### Adding an entity

Every entity is a vertical slice through the same layers. In order:

1. **Migration**: `make migrate-create name=<name>`, with `created_at` / `updated_at` and the `set_updated_at`
   trigger; add the `.up.sql` to `sqlc.yaml`.
2. **SQL**: queries in `sqlc/queries/<entity>.sql` (lists use keyset pagination), then `make sqlc-generate`.
3. **Domain** (`internal/domain/<entity>.go`): the entity, a `Form` and a `Patch` (pointer fields) with `Validate()`,
   the repository interface, cursor and page types for lists; add the interface to `domain.Repository`.
4. **Repository** (`internal/repository/postgres/<entity>/`): implement the interface, map Postgres errors to
   domain errors, read the row back after writes that have derived fields; add it to `BootstrapRepository`.
5. **Usecase** (`internal/usecase/<entity>/`): validate, call the repository, build pages; add it to `usecase.Usecase`.
6. **APIs**, as needed: the proto messages and RPCs (`make proto-generate`) plus a handler registered in
   `internal/delivery/grpc/server.go`; the GraphQL schema (`make gql-generate`) plus resolvers. Document every field,
   argument and error code in the contract.
7. **Tests**: domain `Validate()`, usecase with a hand-written fake repository, repository integration, and the
   end-to-end tests for each API.
8. **Docs**: README (API behaviour, the integration table if needed), CHANGELOG.

CLAUDE.md's *Conventions* section has the rules each step follows (patch updates, pagination, error mapping, scalars).

### Tests

```sh
make test              # unit tests only: fast, no Docker (go test ./test/...)
make test-integration  # everything: unit + integration tests, with -race, against a real Postgres (needs Docker)
```

Tests live in `test/`, mirroring the `internal/` package they cover (e.g. `test/internal/usecase/category` tests
`internal/usecase/category`); shared helpers are in `testutil/`. gRPC handlers and GraphQL resolvers are
tested end-to-end against the real servers, usecases and Postgres, with only Google sign-in faked. CI runs everything with `-race`, the same as `make test-integration`. Integration tests carry the `integration` build tag; each package's `TestMain` calls `testutil.RunWithPostgres`, which starts a throwaway Postgres container,
applies `migrations/` and provides `testutil.TestPool`. They never touch your dev database. Tracing tests call
`testutil.InstallTracing` (before building servers or pools) and read spans with `testutil.Spans`.

To get editor support in integration test files, add the tag to gopls (e.g. `.vscode/settings.json`):
`"gopls": { "buildFlags": ["-tags=integration"] }`.

## Further docs

- [Architecture decision records](docs/adr/README.md): why the main design choices were made.
- [Production readiness](docs/production-readiness.md): what's done and what's planned.
- [Rate limiting spec](docs/rate-limiting.md), [Postman Google login](docs/postman-google-login.md).
- [Security policy](SECURITY.md): how to report a vulnerability.

## License

[Apache License 2.0](LICENSE). Copyright notice in [NOTICE](NOTICE); keep it when redistributing.

## Changelog

See [CHANGELOG.md](CHANGELOG.md).
