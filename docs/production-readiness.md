# Production Readiness Checklist

Review of `revirathya-finance-core` (Go gRPC + GraphQL service) against what a production deploy needs.
Tick items off as they land.

Last reviewed: 2026-10-19

## 1. Bugs found in the current code

- [x] **Shutdown never runs gracefully.** `ctx` in [`cmd/main.go`](../cmd/main.go) is
  `context.Background()` and nothing ever cancels it, so `GracefulStop()` / `Shutdown()` in
  [`cmd/api/startup.go`](../cmd/api/startup.go) never run on SIGTERM and in-flight requests are killed.
  - [x] `main.go` uses `signal.NotifyContext(..., os.Interrupt, syscall.SIGTERM)`.
  - [x] The shutdown timeout context is created after the signal, from `context.Background()`.
  - [x] `http.ErrServerClosed` from the GraphQL server is treated as a clean exit.
  - [x] `GracefulStop()` falls back to `Stop()` when the shutdown deadline passes.
  - [x] The GraphQL server uses `ListenAndServe()` on its own port, not the gRPC listener.
  - Verified: both servers listen (`:8080`, `:50051`), and SIGINT exits cleanly with no panic.
- [x] **gRPC error codes are wrong and leak internals.** In
  [`internal/delivery/grpc/handler/category_handler.go`](../internal/delivery/grpc/handler/category_handler.go) (and
  the other handlers):
  - [x] `expense_handler.go` and `user_context_handler.go` already return `InvalidArgument` for bad input and
    `NotFound` for `domain.ErrNotFound`.
  - [x] `category_handler.go`: an invalid UUID now returns `codes.InvalidArgument`.
  - [x] `category_handler.go`: `domain.ErrNotFound` now returns `codes.NotFound`.
  - [x] All handlers: usecase errors go through the shared `toStatus` mapper
    ([`internal/delivery/grpc/error/error.go`](../internal/delivery/grpc/error/error.go)), which logs the full error and returns
    a generic `internal error` to clients.
  - Optional follow-up: map foreign-key violations (e.g. unknown `category_id`) to a domain error, then to
    `InvalidArgument` in `toStatus`.
- [x] **GraphQL input errors were reported as internal errors.** Several resolvers return bad-input errors as plain
  `fmt.Errorf(...)`, and `ErrorPresenter` only recognises `invalidInputError`. So a bad UUID, date or amount comes
  back as `internal error` / `INTERNAL` instead of `BAD_USER_INPUT`.
  - [x] `expense.resolver.go`: `UpdateExpense`
  - [x] `expense.resolver.go`: `CreateExpense` returns `BAD_USER_INPUT`.
  - [x] `expense.resolver.go`: `CreateExpense` messages for `transactionDt` and `amount`.
  - [x] `expense.resolver.go`: `CreateExpense` messages for `userId` and `categoryId`.
  - [x] `expense.resolver.go`: `UpdateExpense` `transaction_date` typo fixed to `transaction_dt`.
  - [x] `category.resolver.go`
  - [x] `user.resolver.go`
- [x] **Missing mutation input crashed the resolver.** In
  [`graphql/schema/expense.graphql`](../graphql/schema/expense.graphql), `createExpense(input: CreateExpenseInput)`
  and `updateExpense(id: ID!, patch: UpdateExpenseInput)` declare the input as optional. Calling
  `createExpense { id }` with no input passes `nil` to the resolver, which panics with a nil-pointer dereference
  (gqlgen recovers it and returns `INTERNAL`).
  - Done: both inputs are required (`CreateExpenseInput!`, `UpdateExpenseInput!`) and resolvers regenerated.
    Verified: calling them without input no longer panics.
- [x] **`ErrorPresenter` hid GraphQL validation and parse errors.** Its `default` branch turns every unknown
  error into `internal error` / `INTERNAL`, including errors gqlgen produces itself before any resolver runs:
  unknown fields, missing required arguments, syntax errors. Clients can't see what's wrong with their query.
  - Fix: in [`internal/delivery/graphql/error/error.go`](../internal/delivery/graphql/error/error.go), return the default
    presenter's error unchanged when its `code` extension is `errcode.ValidationFailed` or `errcode.ParseFailed`
    (`github.com/99designs/gqlgen/graphql/errcode`). These only describe the query, never server internals.
  - Done: validation and parse errors pass through unchanged. Verified: unknown fields, syntax errors and missing
    required arguments return `GRAPHQL_VALIDATION_FAILED` / `GRAPHQL_PARSE_FAILED` with gqlgen's message.
  - [x] Follow-up: the `default` branch dropped the original error without logging it, so GraphQL internal errors
    are invisible on the server. Log it there (e.g. `slog.Error`), like `toStatus` does for gRPC.
- [x] **CORS default never matches.** The default `ALLOWED_ORIGINS` in
  [`internal/config/config.go`](../internal/config/config.go) was `"localhost,127.0.0.1"`, but browsers send a
  full origin such as `http://localhost:3000`.
  - Done: defaults now include the scheme, and values are trimmed after splitting.
  - Note: `http://localhost` only matches port 80; set `ALLOWED_ORIGINS` with the frontend's port.
- [x] **`make grpc-list` / `make grpc-describe` point at a missing file.** `GRPC_PROTOS` in the
  [`Makefile`](../Makefile) listed `user_contexts.proto`; the file is `user_telegram_contexts.proto`.
- [x] **docker-compose references a service that doesn't exist.** In
  [`infra/docker/docker-compose.yaml`](../infra/docker/docker-compose.yaml), `finance-core` had `depends_on: postgres`, but the service was
  `finance-db`.
  - [x] Services renamed to `db` / `migration` / `core`, and `depends_on` uses the right names.
  - [x] `core` waits for `migration` to complete (`service_completed_successfully`).
  - [x] Containers get an in-network `DATABASE_URI` (`@db:5432`) built from `DB_USER` / `DB_PASSWORD` /
    `DB_NAME`; `DB_PASSWORD` is required.
  - [x] Host ports are configurable, and container ports are fixed (`8080`, `50051`).
  - [x] Make targets renamed to `docker-up` / `docker-down` / `docker-logs` / `docker-build`; `svc=` selects
    services and `args=` passes extra `docker compose down` flags. `make docker-run` removed.
  - [x] Image tags pinned via the required `VERSION` variable instead of `:latest`.
  - [x] `.env.example` uses `DB_HOST_PORT` (default `5532`), matching compose and `DATABASE_URI`.
  - [x] `make logs` no longer follows the old `postgres` service.
  - [x] Makefile follow-ups: `svc` default applies only to `docker-build`, images are tagged with `$(VERSION)`
    (no separate `tag` variable), and `.PHONY` lists the new target names.
- [x] **`http.Server` had no timeouts.** `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` and `IdleTimeout` were
  unset in `setupGraphQLServer`, which left the server open to slowloris-style attacks.
  - Done: all four server timeouts are set (5s / 10s / 15s / 60s).
  - Done: `/graphql` is wrapped in `http.TimeoutHandler` (10s, shorter than `WriteTimeout`), so slow requests
    are cancelled through the request context.

## 2. Mandatory before production

### Security
- [x] **Authentication and authorization.** Every request is now tied to an authenticated user, and all data
  is scoped to that user.
  - [x] Phase 1: gRPC service auth. The bot sends `authorization: Bearer <GRPC_API_KEY>`, checked in a unary
    interceptor with a constant-time comparison.
  - [x] Phase 2: shared plumbing. `ErrUnauthenticated` / `ErrForbidden` / `ErrConflict` / `ErrInvalidInput`, a
    principal in the request context, and mapping in `ToStatus` and `ErrorPresenter`.
  - [x] Phase 3: users and Google sign-in. `loginWithGoogle` verifies a Google ID token and returns a session
    JWT (`JWT_SECRET`, `JWT_TTL`); `me` and `updateUser`; unique `email` and `google_sub`.
  - [x] Phase 4: Telegram binding. One-time codes from the Web Console (`createTelegramBindCode`), redeemed by the
    bot (`BindTelegramAccount`); `x-telegram-user-id` interceptor; attempt limit; `unbindTelegram`.
  - [x] Phase 5: per-user categories, with a composite foreign key so expenses can only use their owner's
    categories.
  - [x] Phase 6: authorization sweep. No client-supplied user IDs remain in GraphQL or gRPC; every query filters
    by the caller's `user_id`, and other users' records return `NOT_FOUND`.
  - [ ] Coordinated Telegram bot update for phases 4–6: send the API key and `x-telegram-user-id` on every call,
    implement `/bind`, drop the removed `user_id` fields and `GetUserTelegramContextByTelegramID`.
  - Later, if the API is ever reached over an untrusted network or by more services: mTLS instead of the shared
    API key.
- [ ] **TLS** on both listeners, or terminated by a trusted proxy or mesh, with that decision documented.
- [x] **GraphQL hardening.** The server is built with `handler.New` instead of `handler.NewDefaultServer`. Its
  tests against the production handler were removed in the restructuring; see **Tests** below.
  - [x] Only the `POST` transport; `GET`, websockets, multipart and APQ removed. Query cache kept.
  - [x] `extension.FixedComplexityLimit(200)` and `SetParserTokenLimit(5000)`. No separate depth limit: the
    schema has no recursive types, so complexity covers deep queries too.
  - [x] Introspection and "Did you mean …?" suggestions off unless `GQL_ENABLE_INTROSPECTION=true`.
  - [x] Request body limited to 1 MB (`413`), including bodies sent without `Content-Length`.
  - [x] `COMPLEXITY_LIMIT_EXCEEDED` passed through `ErrorPresenter` instead of being masked as `INTERNAL`.
- [x] **Input validation**: forms and patches have `Validate()` (helpers in `internal/domain/validation`), called
  by the usecases; errors are `errDom.ValidationError`, which unwraps to `ErrInvalidInput`, and clients get its
  field message (`amount: must be greater than 0`). Postgres `22001` / `22003` map to `ErrInvalidInput` as a
  backstop in every repository that writes text or amounts.
  - Google names are validated, not cut: Google documents at most 60 characters per name field (Workspace), within
    the 100-character limit. Optional: log a warning in `LoginWithGoogle` if Google's profile ever fails validation.

### Reliability
- [x] **Health checks.**
  - [x] `grpc.health.v1` on the gRPC server, exempt from auth. Reports liveness (`SERVING` while the process
    runs, `NOT_SERVING` during shutdown); database readiness is `/readyz`.
  - [x] `/healthz` (liveness) on HTTP.
  - [x] `/readyz` (readiness, pings the DB with a 2s timeout) on HTTP.
  - [x] Docker `healthcheck` for `core`, via the binary's own `healthcheck` subcommand (the image is distroless).
  - Optional later: report `/readyz` as not ready as soon as shutdown starts; tie gRPC health to the database
    if a gRPC load balancer or probe starts depending on it.
- [ ] **Timeouts and limits.**
  - [x] HTTP server timeouts (see bug list).
  - [x] Per-request context deadline for GraphQL (`http.TimeoutHandler`).
  - [ ] gRPC `MaxRecvMsgSize` and keepalive settings.
  - [ ] pgxpool settings: `MaxConns`, `MaxConnLifetime`, etc.
- [x] **Panic recovery.** gRPC: an interceptor after `requestid` and `accesslog` and before the auth interceptors,
  so it covers auth too; logs the panic with a stack trace and returns `Internal`. GraphQL: `RecoverFunc`.
- [ ] **Config validation.**
  - [x] Fail fast on invalid values: required secrets, `JWT_SECRET` length, `JWT_TTL` duration.
  - [ ] Keep secrets in env or a secret manager.
  - [ ] Add `APP_ENV` to gate dev-only features (reflection, introspection, playground).
- [ ] **Migrations.**
  - [x] Remove `000002_temp_seed` (per `CHANGELOG.md`).
  - [ ] Run migrations as a separate job before deploy, not on app start.

### Observability
- [ ] **Structured logging.**
  - [x] `log/slog` with JSON output (`LOG_FORMAT`, default `json`; `LOG_LEVEL`, default `info`).
  - [x] Request ID on every request (`X-Request-ID` / `x-request-id`), added to log lines by the logger.
  - [x] Access logs: one line per HTTP request and gRPC call (method, status/code, duration), level by outcome;
    successful health checks skipped.
  - [x] Error logs carry the request ID: `ToStatus(ctx, ...)`, `ErrorPresenter`, recovery and `readyz` log with
    `slog.*Context`.
  - [x] Replace `panic(err)` at startup with logged errors and a non-zero exit (`run() error` in `main.go`).
  - [ ] Never log secrets or full DSNs.
  - [x] Never log raw request bodies. Request-level errors from gqlgen's transport (no resolver path, no wrapped
    error), such as a body that isn't valid JSON, are returned as `BAD_REQUEST` / "invalid request" and not logged.
  - [x] GraphQL resolver panics go through the logger: `RecoverFunc` (`SetRecoverFunc`) logs the panic and stack
    trace with the request ID, instead of gqlgen's default plain-text output to stderr; `ErrorPresenter` doesn't log
    them a second time.
  - [ ] Optional: `user_id` in access log lines.

### Quality gates
- [ ] **CI pipeline** ([`.github/workflows/ci.yaml`](../.github/workflows/ci.yaml)) running on every PR:
  - [x] `go mod tidy -diff` and `go vet`
  - [ ] `golangci-lint`
  - [x] `go test -race ./...`
  - [x] `buf lint`
  - [ ] `buf breaking --against main`. Optional: per CLAUDE.md, breaking gRPC changes are allowed because
    downstream services follow this repo, so this would only be useful as an informational check.
  - [ ] `govulncheck` (deferred, see section 5)
  - [x] Generated-code drift check (regenerate, then fail on any modified or untracked file)
  - [x] Migration test: `up`, `down --all`, `up` against a Postgres service container
  - [x] Integration tests: `go test -race -tags=integration ./test/internal/repository/...`.
- [ ] **Tests.** All tests live in `test/`, mirroring `internal/`.
  - [x] Repository integration tests against real Postgres (testcontainers-go): `test/internal/repository/postgres`,
    one container per package via `testutil.RunWithPostgres`; run with `make test-integration`.
  - [x] Usecase tests for every usecase, including `user`, `auth` and `telegrambindcode`.
  - [x] Unit tests for the gRPC plumbing (all interceptors, `ToStatus`), config, logging, session tokens and the
    rate limiter.
  - [x] Restore the tests removed in the restructuring (`a38fd29`), now in `test/internal/delivery/graphql/`: the
    production handler from `graphql.Setup`, `ChainMiddleware`, every HTTP middleware, health handlers, and
    `ErrorPresenter` / `RecoverFunc` (with errors wrapped the way gqlgen wraps them).
  - [x] Fix the three bugs these tests found:
    - [x] `ChainMiddleware` skips the last middleware and reverses the order, so there are no HTTP access logs
      and no GraphQL request timeout.
    - [x] The `"POST /graphql"` route makes the mux reject CORS preflight (`OPTIONS`) with 405, so browsers
      can't call the API. Register `"/graphql"`; gqlgen still rejects non-`POST` requests.
    - [x] `ErrorPresenter` changes the message to "internal error" before logging the wrapped resolver error,
      so the log loses the real cause. Log `gqlErr.Err` (the original error) before masking.
  - [ ] gRPC handler and GraphQL resolver tests.

### Documentation
- [ ] **README / runbook.**
  - [x] Local setup.
  - [x] Every env var with its default.
  - [x] Ports and endpoints.
  - [ ] Deploy steps.
  - [ ] Migration rollback procedure.

## 3. Nice-to-have

### Observability
- [ ] Prometheus metrics at `/metrics`, using the `go-grpc-middleware` prometheus provider plus gqlgen
  tracing.
- [ ] OpenTelemetry tracing from gRPC and GraphQL through to pgx (`otelpgx`).

### API design
- [ ] Pagination on `ListCategories` and `expenses` (cursor-based, or page token per
  [AIP-158](https://google.aip.dev/158)).
- [ ] GraphQL custom scalars `Date` and `Decimal` instead of `String` for `transactionDt` and `amount`.
- [ ] Proto `google.type.Date`, plus `google.type.Money` or a units/nanos pattern for amounts.
- [ ] Reuse `Category` messages instead of duplicating fields in `GetCategoryByIDResponse`.
- [ ] Idempotency key on create mutations, so Telegram retries don't create duplicate expenses.
- [ ] Rate limiting, per client or per user.

### Repo hygiene
- [ ] `LICENSE`
- [ ] `CODEOWNERS`
- [ ] `SECURITY.md`
- [ ] PR template.
- [ ] Dependabot or Renovate.
- [ ] Conventional commits with an automated CHANGELOG and releases.
- [ ] `make lint`, `make generate` (all three codegens) and a pre-commit hook.
- [ ] `docs/` with ADRs, such as "why both gRPC and GraphQL" or "money stored as numeric".

### Docker
- [x] Version-tagged images instead of `:latest` (`VERSION`, `MIGRATION_VERSION`).
- [ ] Image scanning (Trivy).
- [ ] Build-time version info via `-ldflags "-X main.version=..."`, exposed at `/version`.

## 4. Schema discovery ("metadata server")

### gRPC

The `.proto` files are the contract. Each kind of consumer gets them a different way; never copy `.proto`
files into client repos by hand.

| Consumer | How they get the spec |
|---|---|
| People and tools (Postman, grpcurl, Evans) | Server reflection (dev/staging), or import `.proto` files with `proto/` as the import path |
| Go clients | `go get` this module and import `gen/go/financecore/v1` |
| Non-Go clients | `buf generate "https://github.com/revirathya/revirathya-finance-core.git#subdir=proto"` |
| Several teams or languages (later) | Buf Schema Registry: versioned modules, rendered docs, generated SDKs |

- [x] **Server reflection** (`google.golang.org/grpc/reflection`), so `grpcurl`, Postman and Evans can
  discover services without proto files.
  - Done: gated by `GRPC_ENABLE_REFLECTION` (default `false`); set it to `true` in the local `.env`.
  - Keep it off in production, at least until auth is in place.
- [ ] **Simplify `make grpc-list` / `make grpc-describe`** to rely on reflection, dropping `GRPC_PROTOS`.
- [ ] **Proto doc comments** on every service, RPC, message and field.
  - Comments must sit directly above the element (no blank line), or tools ignore them.
  - Remove detached section labels such as `// Common` and `// RPC: ...`.
  - Document what types don't say: formats (UUID), constraints (unique, max length), ordering, error codes.
  - Only document error codes once the gRPC error mapping (section 1) is fixed.
  - [x] `categories.proto` (error codes to be added after the gRPC error mapping fix)
  - [x] `expenses.proto` (error codes to be added after the gRPC error mapping fix)
  - [x] `user_telegram_contexts.proto` (error codes to be added after the gRPC error mapping fix; refine the
    meaning of `conversation` / `state` / interaction IDs once the bot's usage is settled)
- [ ] **`buf breaking` in CI.** The most important guard for clients.
- [ ] **Evolution rules** written down: mark removed fields as `reserved` (number and name), never reuse field
  numbers, and introduce `financecore.v2` for breaking redesigns while keeping `v1` running.
- [ ] Optional: render docs with `protoc-gen-doc`, or rely on Buf Schema Registry's rendered docs.
- [ ] Later: **Buf Schema Registry** (`buf push`) once more than one client team or language depends on the
  API.

### GraphQL
- [x] **Introspection** gated by `GQL_ENABLE_INTROSPECTION` (default `false`).
- [ ] **`"""descriptions"""`** on SDL types and fields. GraphiQL and Apollo Sandbox show them as documentation.
- [ ] **Playground** at `/playground`, enabled only in dev.
- Schema registry (Hive, Apollo GraphOS): skip unless there are several GraphQL consumers or federation.

## 5. Later (after the production-readiness checklist)

- [ ] **Make `description` non-nullable, with `""` to clear it.** Today `expenses.description` is nullable, and
  omitting it in `UpdateExpense` keeps the current value (`COALESCE`), so there's no way to clear it. Change it to
  `TEXT NOT NULL DEFAULT ''`: clients clear it by sending `""`, and omitting it still keeps the current value.
  - Migration: backfill existing `NULL`s to `''`, then `SET DEFAULT ''` and `SET NOT NULL`; the down migration
    drops both. Add it to `sqlc.yaml`.
  - `CreateExpense` inserts `NULL` when no description is sent, which a `NOT NULL` column rejects: use
    `COALESCE(sqlc.narg('description'), '')` in the query (or default to `""` before inserting).
  - After `make sqlc-generate`, `description` in the generated rows becomes `string` instead of `*string`; adjust
    the repository mapping. Input can keep `*string` (`nil` = keep, `""` = clear).
  - Downstream (CLAUDE.md): GraphQL `Expense.description` can become `String!`, and gRPC responses always carry
    a value. Clients that treat `null` as "no description" must accept `""`.
- [ ] **Enable `govulncheck` in CI before the first deploy.** It's commented out in
  [`.github/workflows/ci.yaml`](../.github/workflows/ci.yaml) until a deployment target exists. Uncomment it, pin
  the latest release, and fix any reported vulnerabilities before the service goes live.

## 6. Suggested order

1. Bug fixes (section 1).
2. Auth, health checks, gRPC error mapping, timeouts.
3. Logging, recovery interceptors, GraphQL limits.
4. CI with lint, test, `buf breaking` and the codegen drift check.
5. Environment-gated reflection and introspection; proto and SDL doc comments.
6. Metrics and tracing, pagination, proper scalars.
