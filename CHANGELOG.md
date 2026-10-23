# Changelog

All notables changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Pre-release - APIs, GraphQL schema, and gRPC contracts may change without notice.

## [Unreleased]

### Added
- Trivy image scanning in CI (`Image scan` workflow: every PR, weekly, on demand) and `make image-scan`; builds fail on
  HIGH/CRITICAL vulnerabilities with a fix. Exceptions go in `.trivyignore`.
- Per-user write limit, shared by gRPC and GraphQL: 15 writes per minute by default (`WRITE_RATE_LIMIT`,
  `WRITE_RATE_WINDOW`). Over it, gRPC returns `RESOURCE_EXHAUSTED` with `retry-after` metadata and GraphQL
  `RATE_LIMITED` with `extensions.retryAfter` (seconds). Counted: gRPC `CreateExpense` / `UpdateExpense` /
  `DeleteExpense` and every GraphQL mutation except `loginWithGoogle`. New metric `rate_limited_total{api}`.
- OpenTelemetry tracing, off unless `OTEL_EXPORTER_OTLP_ENDPOINT` is set: spans for gRPC calls, HTTP requests,
  GraphQL operations and resolvers, and SQL queries (no SQL parameters, and no GraphQL query text, variables or
  argument values), W3C `traceparent` propagation,
  sampling via `OTEL_TRACES_SAMPLER`, and `trace_id` / `span_id` on log lines. Jaeger for local use as the compose
  profile `tracing` (UI on `127.0.0.1:16686`).
- Idempotent expense creation: an optional `request_id` (gRPC) / `requestId` (GraphQL) UUID on create. Retrying with
  the same key returns the expense the first call created; reusing it for a different expense is `INVALID_ARGUMENT` /
  `BAD_USER_INPUT`. Keys are per user.
- `APP_ENV` (`dev` / `prod`, default `prod`). In `dev`, gRPC reflection and GraphQL introspection default to on and
  `GET /playground` serves GraphiQL; explicit `GRPC_ENABLE_REFLECTION` / `GQL_ENABLE_INTROSPECTION` still win.
- API documentation in the contracts: doc comments on every proto service, RPC, message and field (with error
  codes), and descriptions on every GraphQL type, field and argument. README section "Integrating a new service".
- `make grpc-list` / `make grpc-describe` work without a running server (buf-built descriptor set).
- Prometheus metrics at `GET /metrics` on a separate port (`METRICS_PORT`, default `9090`; Docker host port
  `METRICS_HOST_PORT`, default `9900`): gRPC calls and latency, `/graphql` requests and latency, GraphQL errors by
  code, connection pool stats, Go runtime and process metrics.
- **Authentication and per-user data.** Every request is tied to an authenticated user, and all categories,
  expenses and Telegram contexts are scoped to that user.
- GraphQL sign-in with Google: `loginWithGoogle(idToken:)` verifies a Google ID token and returns a session
  token (JWT); requests send `Authorization: Bearer <token>`. New `me` query and `updateUser` mutation. Users
  are created on first sign-in.
- gRPC authentication for the Telegram bot: `authorization: Bearer <GRPC_API_KEY>` plus
  `x-telegram-user-id` metadata on every call.
- Telegram binding: `createTelegramBindCode` and `unbindTelegram` (GraphQL), `BindTelegramAccount` (gRPC).
  One-time codes, valid 10 minutes, stored only as hashes; at most 10 bind attempts per Telegram account per
  5 minutes.
- Migrations: `000002_add_user_auth` (`google_sub`, unique `email`), `000003_add_telegram_bind`
  (`telegram_bind_codes`), `000004_categories_per_user`.
- Optional gRPC server reflection (`GRPC_ENABLE_REFLECTION`, off by default) and GraphQL introspection
  (`GQL_ENABLE_INTROSPECTION`, off by default).
- Graceful shutdown on SIGINT/SIGTERM, HTTP server timeouts and a per-request timeout for GraphQL.
- Health checks: `GET /healthz` (liveness), `GET /readyz` (database readiness), the standard gRPC
  `grpc.health.v1` service, and a Docker `healthcheck` for `core` via `financecore healthcheck`.
- gRPC panic recovery: a panic in a handler returns `Internal` instead of crashing the process. GraphQL resolver
  panics are logged as structured log lines with their stack trace and request ID.
- CI workflow: tests with the race detector, generated-code drift check, migration up/down test.
- Doc comments on all proto services and messages; `docs/production-readiness.md`,
  `docs/postman-google-login.md`.
- Make targets `docker-build`, `docker-up`, `docker-down`, `docker-logs`; separate `MIGRATION_VERSION` for the
  migration image.

### Changed
- The migration image builds `migrate` from this module on distroless instead of using `migrate/migrate`, so it
  carries the app's patched Go and dependencies. Go 1.26.10 is now the minimum (`go 1.26.10` in `go.mod`) and the
  Dockerfiles build with it.
- GraphQL: the user comes from the session, never from an argument: `expenses(userId:)` is `expenses(transactionDt:)`,
  `CreateExpenseInput.userId` and `updateExpense`'s `userId` are gone, and `user(id:)` is replaced by `me`.
- GraphQL: mutation inputs are required (`CreateExpenseInput!`, `UpdateExpenseInput!`, `UpdateUserInput!`),
  `UpdateCategoryInput.categoryName` is required, and `deleteExpense` / `deleteCategory` return the deleted `UUID!`.
- GraphQL scalars: `transactionDt` is `Date` (`YYYY-MM-DD`), `amount` is `Decimal` (a string), and identifiers are
  `UUID`. All three are JSON strings; variables are declared with the scalar type (`$dt: Date`, `$id: UUID!`). An
  invalid value, or an amount sent as a JSON number, returns `BAD_USER_INPUT`.
- GraphQL `expenses` is paged and returns `ExpenseConnection!` (`nodes`, `pageInfo { hasNextPage endCursor }`), with
  `first` (default 20, max 50) and `after`.
- gRPC messages: one `Category` and one `Expense` message, wrapped by each RPC's response (`CreateExpenseResponse {
  expense }`, `ListExpensesResponse { expenses, next_page_token }`, ...). Dates are `google.type.Date` and amounts
  `google.type.Decimal` (`{"value": "12500.50"}`) instead of strings; an impossible date (e.g. 2026-02-30) or a
  missing amount returns `INVALID_ARGUMENT` with the field name. Generated Go code is in `gen/grpc`.
- Categories are paged by name: GraphQL `categories(first, after)` returns `CategoryConnection!` (`nodes`, `pageInfo`);
  gRPC `ListCategories` takes `page_size` / `page_token` and returns `next_page_token`. Default 20, max 50.
- gRPC `ListExpenses` is paged: `page_size` (default 20, max 50), `page_token`, and `next_page_token` in the response.
- GraphQL create mutations take `form:` instead of `input:` (`createCategory(form: …)`, `createExpense(form: …)`),
  matching `patch:` on updates.
- Updates are patches: `updateCategory`, `updateExpense`, `updateUser` and gRPC `UpdateExpense` /
  `UpdateUserTelegramContext` change only the fields that are sent; omitted (or `null`) fields keep their value, and
  `""` clears `description` / `lastName`. GraphQL update inputs are all nullable; `updateUser` takes `patch:`.
- Expense descriptions are never null: an expense without one has `""` (GraphQL `String!`, gRPC plain `string`;
  database `NOT NULL DEFAULT ''`, migration `000005_expense_description_not_null`). On update, omitting the
  description keeps it and `""` clears it.
- gRPC: `user_id` removed from all request messages; the user comes from the Telegram binding.
- Categories belong to a user; names are unique per user, and an expense can only use its owner's categories.
- gRPC errors use proper status codes (`INVALID_ARGUMENT`, `NOT_FOUND`, `UNAUTHENTICATED`, `PERMISSION_DENIED`,
  `ALREADY_EXISTS`); internal error details are logged instead of returned.
- GraphQL validation and parse errors are returned as-is; bad input returns `BAD_USER_INPUT`.
- GraphQL introspection is off by default (`GQL_ENABLE_INTROSPECTION=true` to enable); the schema is in
  `graphql/schema/*.graphql`. Only `POST` with a JSON body is accepted (no `GET`, websockets, multipart or persisted
  queries).
- GraphQL limits: 1 MB request body (`413`), 5000 query tokens, query complexity 200 (`COMPLEXITY_LIMIT_EXCEEDED`).
- Logs are structured (`log/slog`), JSON by default; configurable with `LOG_FORMAT` and `LOG_LEVEL`. Startup
  failures are logged and exit with code 1 instead of panicking.
- Request IDs: `X-Request-ID` (HTTP) and `x-request-id` (gRPC) are accepted when valid, generated otherwise, and
  returned in the response.
- Access logs: one line per HTTP request and gRPC call with method, status, duration and request ID; `INFO` for
  success, `WARN` for client errors, `ERROR` for server errors. Successful health checks are not logged. Error
  logs carry the same request ID as the request's access log line.
- GraphQL: a malformed request (e.g. a body that isn't valid JSON) returns `BAD_REQUEST` / "invalid request"
  instead of `INTERNAL`, and its body is not logged.
- Input validation: text fields are trimmed and length-checked (category name, first and last name: 1-100
  characters; description: at most 500; Telegram context fields: at most 100), amounts must be greater than 0
  with at most 2 decimal places and below 10^12, and expense dates must be between 2000-01-01 and tomorrow.
  Invalid input returns `INVALID_ARGUMENT` / `BAD_USER_INPUT` with a message naming the field (e.g.
  `amount: must be greater than 0`), and also when only the database catches it.
- gRPC limits: requests over 1 MB return `RESOURCE_EXHAUSTED`; calls without a client deadline time out after
  10 seconds (`DEADLINE_EXCEEDED`); clients may send keepalive pings at most every 30 seconds; idle connections
  close after 5 minutes. Timeouts and cancellations return `DEADLINE_EXCEEDED` / `CANCELED` instead of `INTERNAL`.
- Database: at most 10 connections per process, and Postgres cancels any statement running over 10 seconds. The
  pool is configured in code; `pool_*` parameters in `DATABASE_URI` are ignored.
- `ListExpenses` returns newest `transaction_dt` first; `UpdateExpense` can't change an expense's owner.
- Config: ports are plain numbers (`50051`, not `:50051`): `GRPC_PORT`, `HTTP_PORT` (GraphQL and health checks) and
  `METRICS_PORT`, with Docker host ports `GRPC_HOST_PORT`, `HTTP_HOST_PORT` and `METRICS_HOST_PORT`. Required:
  `GRPC_API_KEY`, `GOOGLE_CLIENT_ID`, `JWT_SECRET`, `MIGRATION_VERSION`.
- Docker compose publishes the metrics port on `127.0.0.1` only.
- Go 1.26; all generated code is committed.

### Removed
- Dev-only seed migration `000002_temp_seed`.
- gRPC `GetUserTelegramContextByTelegramID`.
- GraphQL `user(id:)` (use `me`).
- `make docker-run` (use `make docker-up`).

### Fixed
- GraphQL `createExpense` / `updateExpense` returned an empty `categoryId` (zero UUID) and `categoryName`.
- GraphQL `updateUser`: a blank `firstName` reports `first_name: is required` like every other validation error.
- GraphQL `deleteExpense` deleted nothing (it called the category usecase).
- `GetUserByID` returned `NOT_FOUND` for users without a Telegram binding.
- User repository returned an empty user instead of database errors.
- docker-compose referenced a non-existent service; `make grpc-list` pointed at a missing proto file; the
  default CORS origins could never match.

## [0.0.1] - 2026-09-09

Initial release.

#### Database
- Postgres schema (migration `00001_init_schema`):
    - `users` - user information
    - `user_telegram_contexts` - per-user context for Telegram interactions
    - `categories`- expenses categories
    - `expenses` - expense (transaction) records
- Dev-only seed migration (`000002_temp_seed`) - inserts sample categories, one user, and one Telegram context. Contains hardcoded data; remove before first real deploy and inject user data manually.

#### Categories
- Use cases: create, get by ID, list, update, delete
- GraphQL:
    - Queries: `category`, `categories`
    - Mutations: `createCategory`, `updateCategory`, `deleteCategory`
- gRPC (`CategoryService`):
    - Read Only: `ListCategories`, `GetCategoryByID`

#### Expenses
- Use cases: create, get by ID, list, update, delete
- GraphQL:
    - Query: `expense`, `expenses`
    - Mutations: `createExpense`, `updateExpense`, `deleteExpense`
- gRPC (`ExpenseService`):
    - Read Only: `ListExpenses`,
    - Write: `UpdateExpense`, `DeleteExpense`

#### Users
- Use cases: get by ID
- GraphQL:
    - Query: `user`

#### User Telegram Context
- Use cases: get by user ID, get by telegram ID, update
- gRPC (`UserTelegramContextService`):
    - Read only: `GetUserTelegramContext`, `GetUserTelegramContextByTelegramID`
    - Write: `UpdateUserTelegramContext`

#### Infrastructure
- Mutil-stage Docker build (distroless runtime)
- Dedicated migration image (`migrations.Dockerfile`, based on `migrate/migrate`)
- `docker-compose.yml` for local Postgres + migrations + service
- Makefile targets for build, test, codegen (buf / sqlc / gqlgen), and migrations

[Unreleased]: https://github.com/revirathya/revirathya-finance-core/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/revirathya/revirathya-finance-core/releases/tag/v0.0.1