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
(built from `cmd/`) over two listeners.

Stack: Go 1.26, pgx/v5, gRPC, gqlgen, sqlc, golang-migrate.

## Common commands

```sh
make docker-up             # all compose services (db -> migration -> core); svc=db for Postgres only
make docker-down           # stop services; extra compose flags via args="-v --remove-orphans"
make docker-logs           # follow logs; svc=<service> for one service
make docker-build          # build core image tagged $(VERSION); svc=migration for the migration image ($(MIGRATION_VERSION))
make run                   # run the app locally (go run ./cmd, reads .env)
make build                 # build binary to bin/financecore
make test                  # unit tests only (no Docker, no -race): go test ./test/...
make test-integration      # everything: unit + integration (-race -tags=integration ./test/...), a superset of make test (needs Docker)
make vet                   # go vet ./...
make lint                  # golangci-lint run ./... (config: .golangci.yaml; lints integration tests too)
make check                 # vet + lint + test + proto-lint
make generate              # proto-generate + sqlc-generate + gql-generate
make hooks                 # git config core.hooksPath .githooks (pre-commit: gofmt/vet/codegen drift; commit-msg: conventional commits)

# Single test
go test ./test/internal/usecase/category/... -run TestCreateCategory -v

# Migrations (golang-migrate, needs DATABASE_URI in .env)
make migrate-up
make migrate-down
make migrate-create name=<name>   # creates migrations/<seq>_<name>.{up,down}.sql
make migrate-version

# Codegen — run after editing source-of-truth files
make proto-generate   # buf generate: proto/**/*.proto -> gen/grpc/** (deps from buf.lock: buf.build/googleapis/googleapis)
make sqlc-generate    # sqlc generate: sqlc/queries/*.sql -> gen/sqlc
make gql-generate     # gqlgen generate: graphql/schema/*.graphql -> gen/graphql + resolver stubs in internal/delivery/http/graphql/resolver

# gRPC dev tools: build the protos with buf into bin/financecore.binpb and read it with grpcurl (no server, no reflection)
make grpc-list
make grpc-describe symbol=financecore.v1.CategoryService
```

Docker files live in `infra/docker/`. The Makefile runs compose as
`docker compose --project-directory . -f infra/docker/docker-compose.yaml`, so it reads the root `.env` and keeps
the project name (and its `pgdata` volume). `make docker-build` builds `infra/docker/Dockerfile` (distroless runtime
image; it builds `./cmd` from the committed `gen/` code, so regenerate before building). `make docker-build
svc=migration` builds `infra/docker/migration.Dockerfile`, a standalone `migrate/migrate` image bundled with
`migrations/`. The core image is tagged `$(VERSION)` and the migration image `$(MIGRATION_VERSION)`, both from
`.env`; docker-compose requires both. Make parses leading-dash words as its own flags, so extra compose flags are
passed through make variables (`args="..."`), not as `make docker-down -v`.

Two run modes: local mode (`make docker-up svc=db` + `make run`, app reads `DATABASE_URI` pointing at
`localhost:$DB_HOST_PORT`) and Docker mode (`make docker-up`, compose builds an in-network DSN to `db:5432` and
pins container ports `50051`/`8080`/`9090`; host ports come from `*_HOST_PORT`; the metrics port is bound to `127.0.0.1` only).

## Architecture

Clean-architecture-style layering, one subtree per layer under `internal/`, with a parallel vertical slice per
domain entity (category, expense, user, Telegram context, Telegram bind code):

```
proto/, graphql/schema/        source-of-truth API contracts (hand-written)
sqlc/queries/*.sql             hand-written SQL, source of truth for sqlc
gen/grpc, gen/graphql, gen/sqlc  generated code (buf / gqlgen / sqlc), committed — do not hand-edit
migrations/                    golang-migrate pairs
cmd/main.go                    package main: config, logger, DB pool, wiring, run(); `healthcheck` subcommand
cmd/api/startup.go             RunServers: runs the gRPC, HTTP and metrics servers in an errgroup, graceful shutdown
cmd/sub/healthcheck.go         RunHealthcheck(port): `financecore healthcheck` (calls /readyz on HTTP_PORT; used by Docker)
internal/domain                entities, forms/patches with Validate(), repository interfaces (no framework deps)
                               bootstrap.go: domain.Repository, the bundle of every repository interface
internal/domain/error          domain errors + ValidationError; always import as errDom
internal/domain/validation     per-field validation helpers (ValidateString/Date/Decimal) used by Validate()
internal/domain/constant       DefaultPageSize / MaxPageSize
internal/domain/pagination     Request{Size, Token} + NormalizedSize (< 0 invalid, 0 -> default, cap at max),
                               Cut[T](rows, size) -> page, last, more
internal/usecase/<entity>      business logic; depends only on domain interfaces, not on pgx/gRPC/gqlgen
internal/usecase/bootstrap.go  usecase.Usecase{Auth, Category, Expense, ...} + BootstrapUsecase(verifier, domain.Repository);
                               depends only on domain interfaces, never on internal/repository (no pgx in usecase's imports)
internal/repository/
  postgres/<entity>/repository.go  implements domain.<Entity>Repository with gen/sqlc Queries + pgxpool
  bootstrap.go                     BootstrapRepository(pool) -> domain.Repository (the Postgres implementations)
internal/delivery/grpc/
  server.go                    Setup: handlers, interceptor chain, health + optional reflection -> ServerLayer
  handler/<entity>_handler.go  implement the generated *ServiceServer interfaces
  handler/usecases.go          the usecase interfaces the handlers consume (only the methods they call)
  error/                       ToStatus: domain error -> gRPC status (import with an alias, e.g. errGrpc)
  converter/                   google.type.Date <-> time.Time, google.type.Decimal <-> decimal.Decimal, ParseUUID /
                               ParseOptionalUUID (input errors are ValidationErrors)
  interceptor/requestid        x-request-id metadata; must stay first in the chain
  interceptor/accesslog        one log line per call (method, code, duration); right after requestid
  interceptor/recovery         panic recovery; after accesslog (so panics are logged), before auth
  interceptor/timeout          default 10s deadline for calls without a client deadline; right after recovery
  interceptor/apikeyauth       bot API key (authorization: Bearer)
  interceptor/telegramuser     x-telegram-user-id -> principal (unbound users: BindTelegramAccount only)
internal/delivery/http/
  server.go                    Setup(env, port, ping): the HTTP server + mux (requestid -> accesslog), /healthz, /readyz,
                               and GET /playground (GraphiQL) only when APP_ENV=dev -> ServerLayer
  health/                      /healthz and /readyz handlers
  middleware/helper.go         ChainMiddleware (first listed = outermost, like grpc.ChainUnaryInterceptor)
  middleware/recorder.go       StatusRecorder: shared status capture (first status wins) for accesslog and metrics
  middleware/<name>/           HTTP-wide middleware: requestid, accesslog, metricscollector (records into platform/metrics)
  graphql/setup.go             RegisterHandler: gqlgen handler + limits on the ServerLayer's mux at /graphql
  graphql/resolver/            gqlgen resolvers (*.resolver.go regenerated; resolver.go is hand-maintained DI;
                               usecases.go: the usecase interfaces the resolvers consume)
  graphql/scalar/              gqlgen function-based scalars (gqlgen.yml models): Date <-> time.Time,
                               Decimal <-> decimal.Decimal, UUID <-> uuid.UUID
  graphql/error/error.go       ErrorPresenter: domain error -> GraphQL error code (import as errGql)
  graphql/error/recovery.go    RecoverFunc: logs resolver panics via slog; ErrorPresenter skips logging them again
  graphql/middleware/<name>/   /graphql-only middleware (after metricscollector, which is outermost): limitbody, withcors, withauth (session JWT -> principal), withtimeout
internal/delivery/metrics/     Setup: separate metrics server (METRICS_PORT) serving only GET /metrics
internal/auth                  principal + Telegram ID in context, session JWT (TokenManager), bind-code helpers
internal/platform/database     pgxpool setup (limits + statement_timeout, set in code) + pgx<->domain type conversion helpers (UUID/Date/Numeric/Timestamp)
internal/platform/google       Google ID-token verifier (implements domain.GoogleVerifier)
internal/platform/ratelimit    in-memory fixed-window limiter (per process)
internal/platform/metrics      Prometheus registry + collectors (gRPC, HTTP, GraphQL errors, pool); ObserveHTTP, GraphQLError
internal/platform/logging      slog setup (LOG_FORMAT json/text, LOG_LEVEL), request ID in context, EnsureRequestID
internal/config                env-based config loading (.env via godotenv, falls back to OS env); APP_ENV (dev|prod, default prod)
                               sets the defaults of GRPC_ENABLE_REFLECTION / GQL_ENABLE_INTROSPECTION, explicit values win
infra/docker/                  Dockerfile, migration.Dockerfile, docker-compose.yaml
test/internal/...              all tests, mirroring the internal/ package they cover
testutil/                      shared test helpers: RunWithPostgres/TestPool/New* (integration tag), CaptureLogs
```

Data flow for a request: delivery (grpc handler or graphql resolver) -> usecase (builds the domain form/patch and
calls `Validate()`) -> domain repository interface -> postgres repository impl -> sqlc-generated query -> pgx.
Domain types (`internal/domain/*.go`) are the shared currency between usecase and repository; conversion to/from
pgx wire types happens only inside `internal/repository/postgres/<entity>/repository.go` via
`internal/platform/database/convert.go` helpers (`ToPgxUUID`/`FromPgxUUID`, `ToPgxDate`/`FromPgxDate`/
`ToNullablePgxDate`, `ToPgxNumeric`/`FromPgxNumeric`).

`cmd/main.go` builds `repository.BootstrapRepository(pool)` -> `usecase.BootstrapUsecase(...)` ->
`dlvGrpc.Setup(...)` / `dlvHttp.Setup(...)` + `dlvGql.RegisterHandler(...)`, then `api.RunServers` runs the gRPC
server and the HTTP server
inside an `errgroup`, so either server failing — or SIGINT/SIGTERM — brings both down together via graceful
shutdown. On the HTTP side the whole mux is wrapped in `requestid` -> `accesslog`; `/graphql` additionally goes
through `limitbody` -> `withcors` -> `withauth` -> `withtimeout`; `/healthz` and `/readyz` sit outside CORS and auth.
The gRPC chain is `requestid` -> `accesslog` -> metrics -> `recovery` -> `timeout` -> `apikeyauth` -> `telegramuser`
(`InitializeMetrics` runs after the services are registered); server
options in `grpc/server.go` cap messages at 1 MB and set keepalive. `ToStatus` maps `context.DeadlineExceeded` /
`context.Canceled` to `DeadlineExceeded` / `Canceled` without logging them.

Authentication: GraphQL requests carry a session JWT (`Authorization: Bearer`), turned into a principal by the
`withauth` middleware; tokens come from `loginWithGoogle`. gRPC calls carry the bot's API key plus
`x-telegram-user-id`; `telegramuser` looks up the bound user and sets the principal (unbound users may only call
`BindTelegramAccount`). Handlers and resolvers take the user from `auth.UserIDFrom(ctx)` and pass it explicitly
to usecases; never accept a user ID from the request. Every query on user-owned data filters by `user_id`, and
another user's record returns `errDom.ErrNotFound` (not `ErrForbidden`).

Input validation: each form/patch in `internal/domain` (`CategoryForm`, `ExpensePatch`, `UserForm`, ...) has a
pointer-receiver `Validate()` that trims strings in place and returns an `*errDom.ValidationError{Field, Reason}`
for the first invalid field; usecases call it before the repository. Limits mirror the columns (`VARCHAR(100)`,
`NUMERIC(14, 2)`), counted in characters (`utf8.RuneCountInString`). The error mappers return the
`ValidationError` message to clients (`amount: must be greater than 0`).

Error handling: domain errors live in `internal/domain/error`, imported as
`errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"`; `errDom.ValidationError` unwraps to
`errDom.ErrInvalidInput`. Repositories map `pgx.ErrNoRows` (or zero affected rows for `:execrows`) to
`errDom.ErrNotFound`, unique violations (`23505`) to `errDom.ErrConflict`, foreign-key violations (`23503`) to
`errDom.ErrConflict` (deleting something in use) or `errDom.ErrInvalidInput` (referencing something that isn't the
user's), and `22001` (value too long) / `22003` (numeric out of range) to `errDom.ErrInvalidInput` as a backstop
for input that skipped `Validate()`. gRPC maps domain errors with `errGrpc.ToStatus`; GraphQL with
`errGql.ErrorPresenter`, which also passes gqlgen validation/parse errors through. Unknown errors are logged and
reported as a generic internal error (no raw error leakage to clients).

Note: the gRPC `UserTelegramContextService` proto file is `proto/financecore/v1/user_telegram_contexts.proto`
but the Go package/handler naming uses `UserTelegramContext` (singular `financecorev1.UserTelegramContext*`) —
check `gen/grpc/financecore/v1` for the actual generated symbol names when wiring new RPCs.

## Database

Single schema `app` (not `public`). Migrations live in `migrations/` as sequential golang-migrate pairs.
`000001_init_schema` creates `users`, `user_telegram_contexts`, `categories`, `expenses` plus an
`update_ts_now()` trigger function that auto-maintains `updated_at` on every table. `000002_add_user_auth` adds
`users.google_sub` and unique emails, `000003_add_telegram_bind` adds `telegram_bind_codes`, and
`000004_categories_per_user` makes categories per user, with a composite foreign key
`expenses (category_id, user_id) -> categories (id, user_id)` so an expense can only use its owner's categories.
`000007_expense_request_id` adds `expenses.request_id` (optional idempotency key, unique per user via a partial
index `WHERE request_id IS NOT NULL`; `CreateExpense` uses `ON CONFLICT ... DO NOTHING` and returns the existing
expense, and the usecase rejects a reused key whose payload differs). `000006_expense_pagination_index` replaces the `(user_id, transaction_dt DESC)` index with
`(user_id, transaction_dt DESC, id DESC)` for keyset pagination. `000005_expense_description_not_null` makes `expenses.description` `NOT NULL DEFAULT ''`: `nil` input means
"not sent" (create stores `''`, update keeps the value via `COALESCE`), `""` clears it.
There is no seed data.

`sqlc.yaml` lists each migration's `.up.sql` explicitly as its schema source — add every new migration there,
or `make sqlc-generate` won't see the new columns. SQL in `sqlc/queries/*.sql` is linted with `.sqlfluff`
(postgres dialect, 2-space indent).

## Tests

All tests live under `test/`, mirroring the package they cover (`test/internal/usecase/category` tests
`internal/usecase/category`), as external `_test` packages, so they only use exported API. Integration tests carry
the `//go:build integration` tag; each integration package has
`func TestMain(m *testing.M) { os.Exit(testutil.RunWithPostgres(m)) }`, which starts a Postgres container, applies
`migrations/` and sets `testutil.TestPool` / `testutil.TestDSN`. Data migrations get a test in `test/migrations/`
that migrates down, inserts old-shape rows, and checks them after `up` and `down` (`testutil.MigrationsDir()`). Handlers and resolvers are tested end-to-end
(`test/internal/delivery/grpc/handler`, `test/internal/delivery/http/graphql/resolver`): the production servers on
`testutil.NewUsecases(verifier)` (real repositories on `TestPool`) with `testutil.FakeGoogleVerifier`. CI runs all tests,
including integration, with `-race` (the same command as `make test-integration`).

## Conventions to follow when extending

- Every new table in a migration must have `created_at TIMESTAMPTZ DEFAULT NOW()` and
  `updated_at TIMESTAMPTZ DEFAULT NOW()` columns, plus a `set_updated_at` trigger
  (`BEFORE UPDATE ... FOR EACH ROW EXECUTE FUNCTION app.update_ts_now()`), following the pattern in
  `000001_init_schema.up.sql`.
- API compatibility: there are no downstream consumers yet. Breaking changes are allowed everywhere (gRPC, GraphQL,
  database, config, Docker), including renumbering proto fields. Don't mark changes as breaking or add follow-up
  items for other services; do document how a new service integrates (README: auth, metadata, limits, pagination,
  errors). Revisit this rule when the first consumer depends on the API.
- Adding a new entity/feature touches, in order: migration (+ `sqlc.yaml`) -> sqlc query -> `make sqlc-generate` ->
  domain entity, form/patch with `Validate()`, repo interface -> postgres repo impl in
  `internal/repository/postgres/<entity>/` -> usecase -> (grpc handler and/or graphql resolver) -> wire up in
  `internal/domain/bootstrap.go` (`domain.Repository`), `internal/repository/bootstrap.go`, `internal/usecase/bootstrap.go`, `internal/delivery/grpc/server.go` and
  `internal/delivery/http/graphql/setup.go`.
- API docs live in the contracts: every proto service, RPC, message and field has a comment directly above it, and
  every GraphQL type, field, argument and input field a `"description"`. State what the type doesn't: formats,
  limits, ordering, patch/pagination behaviour, and each RPC's error codes. Keep them in sync when behaviour changes,
  then regenerate (`make proto-generate`, `make gql-generate`); `make grpc-describe` and the dev playground show them.
- Significant decisions are recorded as ADRs in `docs/adr/` (`NNNN-short-title.md`, listed in `docs/adr/README.md`).
  A decision that changes gets a new record that supersedes the old one; accepted records aren't rewritten. ADR 0007
  (no compatibility guarantees yet) must be superseded when the first consumer depends on the API.
- Commit messages and PR titles follow Conventional Commits (`feat(scope): …`, `fix: …`, `chore: …`; the commit-msg hook
  and CI's `pr-title` job enforce it). PRs are squash-merged into `main`.
- Code must pass `make lint` (CI runs golangci-lint). Silence a finding only with a targeted
  `//nolint:<linter> // <reason>` on that line, never by disabling the linter.
- Logging: use `log/slog`'s `*Context` methods (`slog.InfoContext`, `slog.ErrorContext`, ...) wherever a request
  context exists, so the request ID is attached. Never log secrets, tokens, `Authorization` headers, full DSNs, the
  config struct, or raw request bodies. `cmd/main.go` returns errors from `run()` instead of panicking.
- Usecase structs name their dependencies after the entity, not `xxxRepo` (`uc.expense.CreateExpense(...)`), with
  receiver `uc`. Usecases take and return `internal/domain` types only; they must not import pgx, gqlgen, gRPC generated packages,
  or `internal/repository` (including in `usecase/bootstrap.go`, which takes `domain.Repository`). Check with
  `go list -deps ./internal/usecase/... | grep -E 'jackc/pgx|internal/repository'` (should print nothing).
- Usecase tests use hand-written fakes implementing the domain repository interface (see
  `test/internal/usecase/category/usecase_test.go`) rather than mocking frameworks.
- GraphQL resolver files under `internal/delivery/http/graphql/resolver/` are regenerated by `gqlgen generate` except
  `resolver.go` (the dependency-injection root) and `usecases.go`; resolver bodies survive regeneration.
- Delivery depends on usecases through consumer-side interfaces: `delivery/grpc/handler/usecases.go` and
  `delivery/http/graphql/resolver/usecases.go` each declare only the methods that layer calls (gRPC's category
  interface has just the two reads). `NewResolver(tokenManager, usecase.Usecase)` and `dlvGrpc.Setup` unpack
  `usecase.Usecase` into those interface-typed fields; resolvers call `r.expense.CreateExpense(...)`, never the bundle.
  A new usecase method a handler or resolver needs is added to that layer's interface.
- Ownership is always an explicit argument taken from the principal, never a form field:
  `CreateExpense(ctx, userID, form)`, `UpdateExpense(ctx, id, userID, patch)`, `DeleteExpense(ctx, id, userID)`.
  Usecase creates take a domain form (`domain.ExpenseForm`) the way updates take a patch; the handler or resolver
  builds it. GraphQL create mutations take `form:`, updates `patch:`.
- Updates are patches: omitted = keep, `""` clears clearable text (`description`, `last_name`). Domain patches
  (`*Patch`) have pointer fields and `Validate()` checks only the set ones; usecases take the patch struct; SQL uses
  `COALESCE(sqlc.narg('x')::TYPE, x)` per column (`CASE ... NULLIF(..., '')` where `""` must store `NULL`). gRPC
  update requests use proto3 `optional` / message fields for presence (no `update_mask`); GraphQL update inputs are
  all nullable. Mutations return the object directly (no payload types).
- List endpoints use keyset (cursor) pagination, as `ListExpenses` and `ListCategories` do: a total order with a
  unique tiebreaker (`ORDER BY transaction_dt DESC, id DESC`; categories need none, `category_name` is unique per
  user), a comparison against the cursor, an index matching the order, `LIMIT page size + 1` to detect a next page,
  and an opaque base64url cursor (`domain.ExpenseCursor`, `domain.CategoryCursor`). Decoding a cursor validates it
  completely (a text cursor must be valid UTF-8, or Postgres rejects the parameter as an internal error). List usecases
  take `(ctx, userID, [filter,] pagination.Request)` (`domain.ExpenseFilter` holds the filters; lists without
  filters, like `ListCategories`, take none), normalize the
  size with `NormalizedSize` (`constant.DefaultPageSize` / `constant.MaxPageSize`), fetch `size + 1` and cut the page
  with `pagination.Cut(rows, size)`, which also returns the last row for the next cursor. GraphQL returns `nodes` + `pageInfo`; gRPC follows AIP-158
  (`page_size`, `page_token`, `next_page_token`).
- Metrics: one registry (`platform/metrics`), passed in, never the Prometheus global. Labels must be bounded
  (route names, codes, method names), never client-supplied values or IDs. `/metrics` lives only on the metrics
  server, never on the public HTTP port.
- Proto messages: one message per resource (`Category`, `Expense`), and every RPC gets its own request and
  `<Rpc>Response` (buf `STANDARD` lint: `RPC_REQUEST_RESPONSE_UNIQUE`, `RPC_RESPONSE_STANDARD_NAME`), which wraps the
  resource (`CreateExpenseResponse { Expense expense = 1; }`). Dates are `google.type.Date` and amounts
  `google.type.Decimal`, never strings; handlers convert with `delivery/grpc/converter`, whose input errors are
  `ValidationError`s so `ToStatus` returns `INVALID_ARGUMENT` with a `field: reason` message. A `.proto` using
  `google.type.*` must import it (`google/type/date.proto`); `go_package` is
  `.../gen/grpc/financecore/v1;financecorev1` and must match `buf.gen.yaml`'s `out`.
- GraphQL dates, amounts and IDs use the `Date` / `Decimal` / `UUID` scalars, never `String` or `ID`. A scalar's
  `Unmarshal*` returns `errGql.NewInvalidInputError(...)` so bad input is `BAD_USER_INPUT`, not `INTERNAL`. Resolvers
  get `time.Time` / `decimal.Decimal` / `uuid.UUID` and don't parse them.
- Packages named after a builtin or a common identifier (`error`) are always imported with an alias:
  `errDom` (domain), `errGrpc` (gRPC mapper), `errGql` (GraphQL mapper).
