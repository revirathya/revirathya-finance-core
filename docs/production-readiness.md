# Production Readiness Checklist

Review of `revirathya-finance-core` (Go gRPC + GraphQL service) against what a production deploy needs.
Tick items off as they land.

Last reviewed: 2026-10-11

## 1. Bugs found in the current code

- [x] **Shutdown never runs gracefully.** `ctx` in [`cmd/api/main.go`](../cmd/api/main.go) is
  `context.Background()` and nothing ever cancels it, so `GracefulStop()` / `Shutdown()` in
  [`cmd/api/startup.go`](../cmd/api/startup.go) never run on SIGTERM and in-flight requests are killed.
  - [x] `main.go` uses `signal.NotifyContext(..., os.Interrupt, syscall.SIGTERM)`.
  - [x] The shutdown timeout context is created after the signal, from `context.Background()`.
  - [x] `http.ErrServerClosed` from the GraphQL server is treated as a clean exit.
  - [x] `GracefulStop()` falls back to `Stop()` when the shutdown deadline passes.
  - [x] The GraphQL server uses `ListenAndServe()` on its own port, not the gRPC listener.
  - Verified: both servers listen (`:8080`, `:50051`), and SIGINT exits cleanly with no panic.
- [x] **gRPC error codes are wrong and leak internals.** In
  [`internal/delivery/grpc/category_handler.go`](../internal/delivery/grpc/category_handler.go) (and the other
  handlers):
  - [x] `expense_handler.go` and `user_context_handler.go` already return `InvalidArgument` for bad input and
    `NotFound` for `domain.ErrNotFound`.
  - [x] `category_handler.go`: an invalid UUID now returns `codes.InvalidArgument`.
  - [x] `category_handler.go`: `domain.ErrNotFound` now returns `codes.NotFound`.
  - [x] All handlers: usecase errors go through the shared `toStatus` mapper
    ([`internal/delivery/grpc/error.go`](../internal/delivery/grpc/error.go)), which logs the full error and returns
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
  - Fix: in [`internal/delivery/graphql/error.go`](../internal/delivery/graphql/error.go), return the default
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
  [`docker-compose.yml`](../docker-compose.yml), `finance-core` had `depends_on: postgres`, but the service was
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
- [ ] **Authentication and authorization.**
  - gRPC (service-to-service): mTLS, or a shared token checked in an interceptor.
  - GraphQL (end users): JWT or session.
  - Today GraphQL accepts `userId` as an argument, so any caller can read or write any user's expenses.
    **Top priority for financial data.**
- [ ] **TLS** on both listeners, or terminated by a trusted proxy or mesh, with that decision documented.
- [ ] **GraphQL hardening.** `handler.NewDefaultServer` enables introspection, GET, websockets and APQ with no
  query limits.
  - [ ] Build the server explicitly and choose only the transports you need.
  - [ ] Add `extension.FixedComplexityLimit` and a depth limit.
  - [ ] Disable introspection in production, unless clients need it.
- [ ] **Input validation**: UUIDs, positive amounts, valid dates, string lengths. Do it in the usecase or at the
  edge; on the proto side, `protovalidate` is the usual tool.

### Reliability
- [ ] **Health checks.**
  - [ ] `grpc.health.v1` on the gRPC server.
  - [ ] `/healthz` (liveness) on HTTP.
  - [ ] `/readyz` (readiness, pings the DB) on HTTP.
- [ ] **Timeouts and limits.**
  - [ ] HTTP server timeouts (see bug list).
  - [ ] Per-request context deadline.
  - [ ] gRPC `MaxRecvMsgSize` and keepalive settings.
  - [ ] pgxpool settings: `MaxConns`, `MaxConnLifetime`, etc.
- [ ] **Panic recovery** interceptor on gRPC (gqlgen already recovers by default).
- [ ] **Config validation.**
  - [ ] Fail fast on invalid values.
  - [ ] Keep secrets in env or a secret manager.
  - [ ] Add `APP_ENV` to gate dev-only features (reflection, introspection, playground).
- [ ] **Migrations.**
  - [ ] Remove `000002_temp_seed` (per `CHANGELOG.md`).
  - [ ] Run migrations as a separate job before deploy, not on app start.

### Observability
- [ ] **Structured logging.**
  - [ ] `log/slog` with JSON output.
  - [ ] Request ID or trace ID on every log line.
  - [ ] gRPC unary interceptor for logging.
  - [ ] Replace `panic(err)` at startup with logged errors and a non-zero exit.
  - [ ] Never log secrets or full DSNs.

### Quality gates
- [ ] **CI pipeline** (no `.github/` yet) running on every PR:
  - [ ] `go vet`
  - [ ] `golangci-lint`
  - [ ] `go test -race ./...`
  - [ ] `buf lint`
  - [ ] `buf breaking --against main`
  - [ ] `govulncheck`
  - [ ] Generated-code drift check: `buf generate && sqlc generate && gqlgen generate && git diff --exit-code`
- [ ] **Tests.**
  - [ ] Repository integration tests against real Postgres (testcontainers-go).
  - [ ] gRPC handler and GraphQL resolver tests.
  - [ ] `internal/usecase/user` tests (currently none).

### Documentation
- [ ] **README / runbook.**
  - [ ] Local setup.
  - [ ] Every env var with its default.
  - [ ] Ports and endpoints.
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
- [ ] Version-tagged images instead of `:latest`.
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
  - Done: gated by `ENABLE_GRPC_REFLECTION` (default `false`); set it to `true` in the local `.env`.
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
- [ ] **Introspection** gated by environment: on in dev, off or behind auth in production.
- [ ] **`"""descriptions"""`** on SDL types and fields. GraphiQL and Apollo Sandbox show them as documentation.
- [ ] **Playground** at `/playground`, enabled only in dev.
- Schema registry (Hive, Apollo GraphOS): skip unless there are several GraphQL consumers or federation.

## 5. Later (after the production-readiness checklist)

- [ ] **Set method to clear `description`.** Omitting `description` in `UpdateExpense` keeps the current value
  (`COALESCE`), so there's no way to clear it. Add an explicit way, e.g. a dedicated RPC/mutation or a field mask.

## 6. Suggested order

1. Bug fixes (section 1).
2. Auth, health checks, gRPC error mapping, timeouts.
3. Logging, recovery interceptors, GraphQL limits.
4. CI with lint, test, `buf breaking` and the codegen drift check.
5. Environment-gated reflection and introspection; proto and SDL doc comments.
6. Metrics and tracing, pagination, proper scalars.
