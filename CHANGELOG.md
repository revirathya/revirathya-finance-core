# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

First release.

### Added

#### Data and APIs
- Categories, expenses, users and Telegram conversation state in Postgres (schema `app`, migrations
  `000001`–`000007`), served over **gRPC** (for services such as the Telegram bot) and **GraphQL** (for the Web
  Console) from one binary.
- GraphQL: `me`, `updateUser`, `categories` / `createCategory` / `updateCategory` / `deleteCategory`, `expenses` /
  `createExpense` / `updateExpense` / `deleteExpense`, `createTelegramBindCode`, `unbindTelegram`, `loginWithGoogle`.
- gRPC: `CategoryService` (`ListCategories`, `GetCategoryByID`), `ExpenseService` (create, get, list, update, delete),
  `UserTelegramContextService` (`GetUserTelegramContext`, `UpdateUserTelegramContext`, `BindTelegramAccount`).
- Typed values: dates and amounts are `google.type.Date` / `google.type.Decimal` on gRPC and `Date` / `Decimal` / `UUID`
  scalars on GraphQL; amounts are exact decimals (`NUMERIC(14, 2)`).
- Keyset pagination on expenses (newest first) and categories (by name): GraphQL `first` / `after` with `nodes` and
  `pageInfo`, gRPC `page_size` / `page_token` / `next_page_token`. Default 20, max 50.
- Updates are patches: omitted fields keep their value, and `""` clears `description` / `lastName`.
- Idempotent expense creation: an optional `request_id` / `requestId` makes a retry return the first expense; a reused
  key with a different payload is `INVALID_ARGUMENT` / `BAD_USER_INPUT`.
- Input validation with field-level messages (`amount: must be greater than 0`), returned as `INVALID_ARGUMENT` /
  `BAD_USER_INPUT`.
- API documentation in the contracts: comments on every proto element and descriptions on every GraphQL type, field
  and argument, including error codes and limits.

#### Authentication
- GraphQL sign-in with Google: `loginWithGoogle` exchanges a Google ID token for a session token (JWT, `JWT_TTL`).
- gRPC: the bot sends `authorization: Bearer <GRPC_API_KEY>` and the sender's `x-telegram-user-id`; Telegram accounts
  are bound with one-time codes from the Web Console (valid 10 minutes, stored as hashes, 10 attempts per account per
  5 minutes).
- All data is scoped to the signed-in user; another user's record is `NOT_FOUND`.

#### Limits and reliability
- Per-user write limit shared by both APIs: 15 writes a minute by default (`WRITE_RATE_LIMIT`, `WRITE_RATE_WINDOW`);
  over it, `RESOURCE_EXHAUSTED` with `retry-after` (gRPC) or `RATE_LIMITED` with `retryAfter` (GraphQL).
- Request limits: 1 MB messages and bodies, a 10-second default deadline, GraphQL token (5000) and complexity (200)
  limits, `POST`-only GraphQL, keepalive enforcement, and a 10-second Postgres statement timeout.
- Health checks: `GET /healthz`, `GET /readyz` (database), gRPC `grpc.health.v1`, and `financecore healthcheck` for the
  Docker health check.
- Graceful shutdown on SIGINT / SIGTERM, and panic recovery in both APIs.

#### Observability
- Structured logs (`log/slog`, JSON by default; `LOG_FORMAT`, `LOG_LEVEL`) with a request ID, and a trace ID inside a
  span; one access log line per request, with `user_id` when the caller is signed in. Secrets, tokens, headers and
  bodies are never logged.
- Prometheus metrics on a separate port (`METRICS_PORT`): gRPC and GraphQL requests and latency, GraphQL errors by
  code, `rate_limited_total`, connection pool stats, `financecore_build_info`, Go runtime and process.
- OpenTelemetry tracing over OTLP, off unless `OTEL_EXPORTER_OTLP_ENDPOINT` is set: gRPC, HTTP, GraphQL operations and
  resolvers, and SQL queries, without SQL parameters or GraphQL query text, variables or arguments. Jaeger for local
  use (`make docker-up tracing=1`).
- Build version stamped into the binary (`VERSION`): `GET /version`, `financecore version`, the build-info metric, and
  the traces' `service.version`.

#### Operations and development
- Docker images on distroless: `core`, and a `migration` image with `migrate` built from this module. Docker Compose
  for local use, and Make targets for building, running, migrating, code generation and scanning.
- `APP_ENV` (`dev` / `prod`): in `dev`, gRPC reflection, GraphQL introspection and the GraphiQL playground
  (`GET /playground`) default to on.
- CI on every PR: tests with the race detector (unit and integration), lint, `govulncheck`, generated-code drift,
  a migration up/down test, Conventional Commit PR titles, and Trivy scans of both images (also weekly).
- Docs: ADRs (`docs/adr/`), runbooks (`docs/runbooks/`), and the open production-readiness items.

[Unreleased]: https://github.com/revirathya/revirathya-finance-core/commits/main
