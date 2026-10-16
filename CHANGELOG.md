# Changelog

All notables changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Pre-release - APIs, GraphQL schema, and gRPC contracts may change without notice.

## [Unreleased]

### Added
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
- **Breaking (GraphQL):** client-supplied user IDs removed: `expenses(userId:)` is now `expenses(transactionDt:)`,
  `CreateExpenseInput.userId` and `updateExpense`'s `userId` are gone, and `user(id:)` is replaced by `me`.
- **Breaking (GraphQL):** mutation inputs are required (`CreateExpenseInput!`, `UpdateExpenseInput!`,
  `UpdateUserInput!`), `UpdateCategoryInput.categoryName` is required, and `deleteExpense` / `deleteCategory`
  return `ID!`.
- **Breaking (gRPC):** `user_id` removed from all request messages; the user comes from the Telegram binding.
  `UpdateExpenseRequest` fields renumbered.
- **Breaking:** categories belong to a user; names are unique per user, and an expense can only use its
  owner's categories.
- gRPC errors use proper status codes (`INVALID_ARGUMENT`, `NOT_FOUND`, `UNAUTHENTICATED`,
  `PERMISSION_DENIED`, `ALREADY_EXISTS`); internal error details are logged instead of returned.
- GraphQL validation and parse errors are returned as-is; bad input returns `BAD_USER_INPUT`.
- **Breaking (GraphQL):** introspection is off by default, so tools can no longer download the schema from the
  server unless `GQL_ENABLE_INTROSPECTION=true`; use the committed `graphql/schema/*.graphql`. Only `POST` with a
  JSON body is accepted (`GET`, websockets, multipart and persisted queries removed).
- GraphQL limits: 1 MB request body (`413`), 5000 query tokens, query complexity 200
  (`COMPLEXITY_LIMIT_EXCEEDED`).
- Logs are structured (`log/slog`), JSON by default; configurable with `LOG_FORMAT` and `LOG_LEVEL`. Startup
  failures are logged and exit with code 1 instead of panicking.
- Request IDs: `X-Request-ID` (HTTP) and `x-request-id` (gRPC) are accepted when valid, generated otherwise, and
  returned in the response.
- Access logs: one line per HTTP request and gRPC call with method, status, duration and request ID; `INFO` for
  success, `WARN` for client errors, `ERROR` for server errors. Successful health checks are not logged. Error
  logs carry the same request ID as the request's access log line.
- GraphQL: a malformed request (e.g. a body that isn't valid JSON) returns `BAD_REQUEST` / "invalid request"
  instead of `INTERNAL`, and its body is no longer logged.
- `ListExpenses` returns newest `transaction_dt` first; `UpdateExpense` can no longer change an expense's owner.
- `GRPC_PORT` / `GQL_PORT` are plain numbers (`50051`, not `:50051`). New required config: `GRPC_API_KEY`,
  `GOOGLE_CLIENT_ID`, `JWT_SECRET`, `MIGRATION_VERSION`.
- Go 1.26; all generated code is committed.

### Removed
- Dev-only seed migration `000002_temp_seed`.
- gRPC `GetUserTelegramContextByTelegramID`.
- GraphQL `user(id:)` (use `me`).
- `make docker-run` (use `make docker-up`).

### Fixed
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