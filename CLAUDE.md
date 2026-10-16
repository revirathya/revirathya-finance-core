# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Instructions

1. Never edit anything unless explicitly asked to. Answer questions, investigate, and propose changes
   freely, but do not modify, create, or delete files until the user explicitly requests the change.
2. Exception to rule 1: keep the Markdown docs in sync without asking. Whenever a change (yours or the user's)
   makes any `*.md` file out of date — `README.md`, `CLAUDE.md`, `CHANGELOG.md`, `docs/*.md` — update those docs
   to match, and say which ones were updated. This covers docs only, never code or config.
3. Tests are written by Claude, not the user. When guiding an implementation step, don't tell the user to write
   tests. Once the user has finished the implementation and Claude has reviewed it, Claude writes the unit tests
   for it without being asked (exception to rule 1 for test files), runs them, and reports the results —
   including tests that fail because of a bug found in the review.

## Overview

Revirathya Finance Core is a Go service managing financial data (users, categories, expenses, Telegram
conversation state). It exposes both gRPC and GraphQL APIs backed by Postgres, served from a single binary
(`cmd/api`) over two listeners.

Stack: Go 1.26, pgx/v5, gRPC, gqlgen, sqlc, golang-migrate.

## Common commands

```sh
make docker-up             # all compose services (db -> migration -> core); svc=db for Postgres only
make docker-down           # stop services; extra compose flags via args="-v --remove-orphans"
make docker-logs           # follow logs; svc=<service> for one service
make docker-build          # build core image tagged $(VERSION); svc=migration for the migration image ($(MIGRATION_VERSION))
make run                    # run the API locally (reads .env)
make build                  # build binary to bin/api
make test                   # go test ./...
make vet                    # go vet ./...
make check                  # vet + test + proto-lint

# Single test
go test ./internal/usecase/category/... -run TestCreateCategory -v

# Migrations (golang-migrate, needs DATABASE_URI in .env)
make migrate-up
make migrate-down
make migrate-create name=<name>   # creates migrations/<seq>_<name>.{up,down}.sql
make migrate-version

# Codegen — run after editing source-of-truth files
make proto-generate   # buf generate: proto/**/*.proto -> gen/go/**  (edit .proto, then regenerate)
make sqlc-generate    # sqlc generate: internal/repository/sqlc/queries/*.sql -> internal/repository/sqlc/gen
make gql-generate     # gqlgen generate: graphql/schema/*.graphql -> gen/graphql + resolver stubs

# gRPC dev tools (read local proto files; server reflection is only registered with GRPC_ENABLE_REFLECTION=true)
make grpc-list
make grpc-describe symbol=financecore.v1.CategoryService
```

Docker: `make docker-build` builds the app image from `Dockerfile` (which itself runs `buf generate` and
`gqlgen generate` inside the builder stage — generated code does not need to be committed for Docker builds,
but IS committed under `gen/` for local `go build`/IDE use). `make docker-build svc=migration` builds
`migration.Dockerfile`, a standalone `migrate/migrate` image bundled with `migrations/`. The core image is tagged
`$(VERSION)` and the migration image `$(MIGRATION_VERSION)`, both from `.env`; docker-compose requires both. Make parses leading-dash words as its own flags,
so extra compose flags are passed through make variables (`args="..."`), not as `make docker-down -v`.

Two run modes: local mode (`make docker-up svc=db` + `make run`, app reads `DATABASE_URI` pointing at
`localhost:$DB_HOST_PORT`) and Docker mode (`make docker-up`, compose builds an in-network DSN to `db:5432` and
pins container ports `50051`/`8080`; host ports come from `*_HOST_PORT`).

## Architecture

Clean-architecture-style layering, one subtree per layer under `internal/`, with a parallel vertical slice per
domain entity (category, expense, user, user context):

```
proto/, graphql/schema/   source-of-truth API contracts (hand-written)
gen/go, gen/graphql        generated code from the above (buf / gqlgen) — do not hand-edit
internal/domain            entities + repository interfaces + domain errors (no framework deps)
internal/usecase/<entity>  business logic; depends only on domain interfaces, not on pgx/gRPC/gqlgen
internal/repository/
  sqlc/queries/*.sql       hand-written SQL, source-of-truth for sqlc
  sqlc/gen/                generated query code (sqlc) — do not hand-edit
  postgres/<entity>.go     implements domain.<Entity>Repository using sqlcgen.Queries + pgxpool
internal/delivery/
  grpc/<entity>_handler.go     gRPC handlers, implement generated *ServiceServer interfaces
  grpc/auth/                   gRPC interceptors: APIKeyAuth (bot), TelegramUser (x-telegram-user-id -> principal)
  grpc/error/                  ToStatus: domain error -> gRPC status (import with an alias, e.g. grpcErr)
  grpc/requestid/              request-ID interceptor (x-request-id metadata); must stay first in the chain
  grpc/accesslog/              one log line per call (method, code, duration); right after requestid
  grpc/recovery/               panic-recovery interceptor; after accesslog (so panics are logged), before auth
  graphql/<entity>.resolver.go gqlgen resolvers (regenerated stubs — resolver.go itself is hand-maintained DI)
  graphql/middleware.go        WithAuth: session token -> principal
  graphql/recovery.go          RecoverFunc: logs resolver panics via slog; ErrorPresenter skips logging them again
internal/auth                 principal + Telegram ID in context, session JWT (TokenManager), bind-code helpers
internal/platform/database   pgxpool setup + pgx<->domain type conversion helpers (UUID/Date/Numeric/Timestamp)
internal/platform/google     Google ID-token verifier (implements domain.GoogleVerifier)
internal/platform/ratelimit  in-memory fixed-window limiter (per process)
internal/platform/logging    slog setup (LOG_FORMAT json/text, LOG_LEVEL), request ID in context, EnsureRequestID
internal/config               env-based config loading (.env via godotenv, falls back to OS env)
cmd/api                       wires everything together and runs gRPC + GraphQL servers concurrently
```

Data flow for a request: delivery (grpc handler or graphql resolver) -> usecase -> domain repository interface
-> postgres repository impl -> sqlc-generated query -> pgx. Domain types (`internal/domain/*.go`) are the
shared currency between usecase and repository; conversion to/from pgx wire types happens only inside
`internal/repository/postgres/*.go` via `internal/platform/database/convert.go` helpers
(`ToPgxUUID`/`FromPgxUUID`, `ToPgxDate`/`FromPgxDate`/`ToNullablePgxDate`, `ToPgxNumeric`/`FromPgxNumeric`).

`cmd/api/main.go` builds repositories -> usecases -> handlers/resolvers in that order, then
`cmd/api/startup.go` starts the gRPC server and the GraphQL HTTP server (mounted at `/graphql`, wrapped in a
hand-rolled CORS middleware from `cmd/api/cors.go`; `/healthz` and `/readyz` from `cmd/api/health.go` sit outside
CORS and auth; the whole mux is wrapped in `withRequestID(withAccessLog(...))` from `cmd/api/middleware.go`) inside an `errgroup`, so either server failing — or context
cancellation — brings both down together via graceful shutdown.

Authentication: GraphQL requests carry a session JWT (`Authorization: Bearer`), turned into a principal by
`WithAuth`; tokens come from `loginWithGoogle`. gRPC calls carry the bot's API key plus `x-telegram-user-id`;
`TelegramUser` looks up the bound user and sets the principal (unbound users may only call
`BindTelegramAccount`). Handlers and resolvers take the user from `auth.UserIDFrom(ctx)` and pass it explicitly
to usecases; never accept a user ID from the request. Every query on user-owned data filters by `user_id`, and
another user's record returns `domain.ErrNotFound` (not `ErrForbidden`).

Error handling: repositories map `pgx.ErrNoRows` (or zero affected rows for `:execrows`) to
`domain.ErrNotFound`, unique violations (`23505`) to `domain.ErrConflict`, and foreign-key violations (`23503`)
to `domain.ErrConflict` (deleting something in use) or `domain.ErrInvalidInput` (referencing something that
isn't the user's). gRPC maps domain errors with `grpcErr.ToStatus`; GraphQL with `ErrorPresenter`
(`internal/delivery/graphql/error.go`), which also passes gqlgen validation/parse errors through. Unknown errors
are logged and reported as a generic internal error (no raw error leakage to clients).

Note: the gRPC `UserTelegramContextService` proto file is `proto/financecore/v1/user_telegram_contexts.proto`
but the Go package/handler naming uses `UserTelegramContext` (singular `financecorev1.UserTelegramContext*`) —
check `gen/go/financecore/v1` for the actual generated symbol names when wiring new RPCs.

## Database

Single schema `app` (not `public`). Migrations live in `migrations/` as sequential golang-migrate pairs.
`000001_init_schema` creates `users`, `user_telegram_contexts`, `categories`, `expenses` plus an
`update_ts_now()` trigger function that auto-maintains `updated_at` on every table. `000002_add_user_auth` adds
`users.google_sub` and unique emails, `000003_add_telegram_bind` adds `telegram_bind_codes`, and
`000004_categories_per_user` makes categories per user, with a composite foreign key
`expenses (category_id, user_id) -> categories (id, user_id)` so an expense can only use its owner's categories.
There is no seed data.

`sqlc.yaml` lists each migration's `.up.sql` explicitly as its schema source — add every new migration there,
or `make sqlc-generate` won't see the new columns. SQL in `internal/repository/sqlc/queries/*.sql` is
linted with `.sqlfluff` (postgres dialect, 2-space indent).

## Conventions to follow when extending

- Every new table in a migration must have `created_at TIMESTAMPTZ DEFAULT NOW()` and
  `updated_at TIMESTAMPTZ DEFAULT NOW()` columns, plus a `set_updated_at` trigger
  (`BEFORE UPDATE ... FOR EACH ROW EXECUTE FUNCTION app.update_ts_now()`), following the pattern in
  `000001_init_schema.up.sql`.
- API compatibility with downstream consumers:
  - gRPC (`proto/`): breaking changes, including renumbering fields, are allowed. Downstream services follow
    this repo's proto changes. Only keep wire compatibility (keep field numbers, `reserved` removed ones) when
    the user explicitly asks for it for a given change.
  - Everything else (GraphQL schema, database schema/migrations, env/config, Docker images): treat breaking
    changes as risky for downstream consumers. Point out the possible conflict before making or recommending
    such a change.
- Adding a new entity/feature touches, in order: migration -> sqlc query -> `make sqlc-generate` -> domain
  entity+repo interface -> postgres repo impl -> usecase -> (grpc handler and/or graphql resolver) -> wire up
  in `cmd/api/main.go` / `startup.go`.
- Logging: use `log/slog`'s `*Context` methods (`slog.InfoContext`, `slog.ErrorContext`, ...) wherever a request
  context exists, so the request ID is attached. Never log secrets, tokens, `Authorization` headers, full DSNs, the
  config struct, or raw request bodies. `cmd/api/main.go` returns errors from `run()` instead of panicking.
- Usecases take and return `internal/domain` types only; they must not import pgx, gqlgen, or gRPC generated
  packages.
- Usecase tests use hand-written fakes implementing the domain repository interface (see
  `internal/usecase/category/usecase_test.go`) rather than mocking frameworks.
- GraphQL resolver files under `internal/delivery/graphql/` are regenerated by `gqlgen generate` except
  `resolver.go`, which is the manually maintained dependency-injection root — add new usecase dependencies
  there.
