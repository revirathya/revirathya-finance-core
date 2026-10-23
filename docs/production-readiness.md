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
  - Fix: in [`internal/delivery/http/graphql/error/error.go`](../internal/delivery/http/graphql/error/error.go), return the default
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
  - [ ] Before a second gRPC client: per-service credentials instead of the single shared `GRPC_API_KEY` (an API key or
    client certificate per calling service, so each can be identified, limited and revoked on its own; mTLS once
    the network isn't trusted). Other services will call this API (microservices), so the current "one key + a
    Telegram ID" model only fits the bot. Also decide how a service-to-service call names its user, since not every
    caller will have a Telegram ID.
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
- [x] **Timeouts and limits.**
  - [x] HTTP server timeouts (see bug list).
  - [x] Per-request context deadline for GraphQL (`http.TimeoutHandler`).
  - [x] gRPC: `MaxRecvMsgSize` 1 MB, `MaxConcurrentStreams` 100, 10s connection timeout, keepalive params and
    enforcement policy (client pings at most every 30s).
  - [x] gRPC default deadline: `interceptor/timeout` gives calls without a client deadline 10s; `ToStatus` maps
    `DeadlineExceeded` / `Canceled` instead of reporting them as `INTERNAL`.
  - [x] pgxpool settings in code: `MaxConns` 10, `MinConns` 1, `MaxConnLifetime` 1h, `MaxConnIdleTime` 30m,
    5s connect timeout, `statement_timeout` 10s, `application_name`.
  - [x] `make test-integration` and CI run `-tags=integration ./test/...`, so integration tests outside
    `test/internal/repository/` (e.g. `test/internal/platform/database`) run too.
  - Optional: `DB_MAX_CONNS` (and maybe `statement_timeout`) as config once the deploy target is known.
  - Optional: GraphQL `ErrorPresenter` still logs resolver errors from a request timeout as `graphql request
    failed`; map `context.DeadlineExceeded` there too to cut that noise.
- [x] **Panic recovery.** gRPC: an interceptor after `requestid` and `accesslog` and before the auth interceptors,
  so it covers auth too; logs the panic with a stack trace and returns `Internal`. GraphQL: `RecoverFunc`.
- [ ] **Config validation.**
  - [x] Fail fast on invalid values: required secrets, `JWT_SECRET` length, `JWT_TTL` duration.
  - [ ] Keep secrets in env or a secret manager.
  - [x] `APP_ENV` gates dev-only features: in `dev`, reflection, introspection and the playground default to on.
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
  - [x] Runner hygiene: `ubuntu-24.04` pinned, `timeout-minutes: 10` on every job, superseded PR runs cancelled
    (`concurrency`, PR-only, so `main` runs are never cancelled), `actions/checkout@v7` and `actions/setup-go@v7`.
  - [x] `go mod tidy -diff` and `go vet`
  - [x] `golangci-lint` v2 (`.golangci.yaml`: `standard` + `bodyclose`, `errorlint`, `gosec`, `misspell`; gofmt and
    goimports; `make lint`, part of `make check`; CI via `golangci/golangci-lint-action@v8`, pinned to `v2.15.0`).
    - Optional later: `revive` (naming, e.g. `uuidId` -> `uuidID`) and `noctx`.
  - [x] `go test -race -tags=integration ./test/...` (unit and integration tests in one step)
  - [x] `buf lint`
  - [ ] Later: `buf breaking --against main`, once other services depend on the gRPC API.
  - [ ] `govulncheck` (deferred, see section 5)
  - [x] Generated-code drift check (regenerate, then fail on any modified or untracked file)
  - [x] Migration test: `up`, `down --all`, `up` against a Postgres service container
  - [x] Integration tests: `go test -race -tags=integration ./test/...`.
- [ ] **Tests.** All tests live in `test/`, mirroring `internal/`.
  - [x] Repository integration tests against real Postgres (testcontainers-go): `test/internal/repository/postgres`,
    one container per package via `testutil.RunWithPostgres`; run with `make test-integration`.
  - [x] Usecase tests for every usecase, including `user`, `auth` and `telegrambindcode`.
  - [x] Unit tests for the gRPC plumbing (all interceptors, `ToStatus`), config, logging, session tokens and the
    rate limiter.
  - [x] Restore the tests removed in the restructuring (`a38fd29`), now in `test/internal/delivery/http/`: the
    production handler from `graphql.Setup`, `ChainMiddleware`, every HTTP middleware, health handlers, and
    `ErrorPresenter` / `RecoverFunc` (with errors wrapped the way gqlgen wraps them).
  - [x] Fix the three bugs these tests found:
    - [x] `ChainMiddleware` skips the last middleware and reverses the order, so there are no HTTP access logs
      and no GraphQL request timeout.
    - [x] The `"POST /graphql"` route makes the mux reject CORS preflight (`OPTIONS`) with 405, so browsers
      can't call the API. Register `"/graphql"`; gqlgen still rejects non-`POST` requests.
    - [x] `ErrorPresenter` changes the message to "internal error" before logging the wrapped resolver error,
      so the log loses the real cause. Log `gqlErr.Err` (the original error) before masking.
  - [x] gRPC handler and GraphQL resolver tests: end-to-end integration tests through the production servers
    (`dlvGrpc.Setup`, `dlvHttp.Setup` + `RegisterHandler`) on the real usecases, repositories and Postgres, with
    only Google faked (`testutil.NewUsecases`, `testutil.FakeGoogleVerifier`).
  - [x] Fix the bug these tests found: `createExpense` / `updateExpense` (GraphQL) returned `categoryId` as the zero
    UUID and `categoryName` as `""`. The repository now reads the expense back with `GetExpenseByID` after the write
    (the `RETURNING *` row has no `category_name`), and the resolvers map both fields.

### Documentation
- [ ] **README / runbook.**
  - [x] Local setup.
  - [x] Every env var with its default.
  - [x] Ports and endpoints.
  - [ ] Deploy steps.
  - [ ] Migration rollback procedure.

## 3. Nice-to-have

### Before the first consumer (breaking changes are free now)
There are no downstream consumers yet (see CLAUDE.md), so API shape changes cost nothing today and break every
client later. Do these first.
- [x] Proto `google.type.Date` for dates and `google.type.Decimal` for amounts instead of strings, matching the
  GraphQL `Date` / `Decimal` scalars (`Decimal` rather than `Money`: one currency, and `NUMERIC(14, 2)` is exactly a
  decimal). Converters in `internal/delivery/grpc/converter`; googleapis is a buf dependency (`buf.lock`).
  Generated Go code moved to `gen/grpc`.
- [x] Shared messages: one message per resource, wrapped by each RPC's own `<Rpc>Response` (buf `STANDARD` lint
  requires a unique, `<Rpc>Response`-named response per RPC).
  - [x] `Category` (`ListCategoriesResponse`, `GetCategoryByIDResponse`).
  - [x] `Expense` (`CreateExpenseResponse`, `GetExpenseByIDResponse`, `UpdateExpenseResponse`,
    `ListExpensesResponse`).
  - [x] `UserTelegramContext` (`GetUserTelegramContextResponse`, `UpdateUserTelegramContextResponse`).
- [x] Category pagination (`categories` / `ListCategories`), by `category_name` (unique per user, so no tiebreaker
  and no migration: the unique constraint's index matches).
- [x] CRUD conventions on the APIs (client-visible, so settle them before the first consumer):
  - [x] Update semantics: **patch** for every entity. Omitted (or `null`) = keep; `""` clears `description` and
    `lastName`. gRPC uses proto3 `optional` fields (presence) rather than `update_mask`; GraphQL uses nullable input
    fields. SQL `COALESCE(narg, column)` per column; domain patches have pointer fields; usecases take the patch.
    - [x] gRPC `UpdateExpense` handler fixed: it required `transaction_dt` and `amount` and turned an omitted
      `category_id` into a random UUID; it now uses `FromOptionalProtoDate`, `FromOptionalProtoDecimal` and a `nil`
      category ID.
  - [x] GraphQL mutation results: return the object directly. Payload types
    (`createExpense(form): CreateExpensePayload { expense }`) aren't needed yet.
  - Already settled: Create / Update / Get return the object as stored (including derived fields such as
    `category_name`); gRPC wraps it in a one-field `<Rpc>Response` (buf `STANDARD` lint); Delete returns an empty
    `Delete*Response` on gRPC ([AIP-135](https://google.aip.dev/135)) and the deleted `UUID!` on GraphQL (for client
    cache eviction); deleting something missing is `NOT_FOUND`.

### Integration docs (for the first new service)
- [x] Proto doc comments on every service, RPC, message and field, with each RPC's error codes, limits, ordering,
  and patch/pagination behaviour.
- [x] GraphQL descriptions on every type, field and argument, including auth and error codes on `Query` / `Mutation`.
- [x] `APP_ENV` (`dev` / `prod`, default `prod`) and a dev-only GraphQL playground (`GET /playground`, GraphiQL).
- [x] README section "Integrating a new service": auth (gRPC API key + `x-telegram-user-id`, or a GraphQL session
  JWT), required metadata and headers, error codes, limits, pagination, scalars and retries, in one place. Most of
  it exists already, spread across the README.

### Observability
- [x] Prometheus metrics at `/metrics` on a separate `METRICS_PORT` listener: gRPC via the `go-grpc-middleware`
  prometheus provider, `/graphql` requests and latency, GraphQL errors by code, pgxpool stats, Go runtime and process.
  Optional later: dashboards and alerts (error rate, latency, `db_pool_acquire_wait_total` growth).
- [x] OpenTelemetry tracing from gRPC and GraphQL through to pgx, exported over OTLP when
  `OTEL_EXPORTER_OTLP_ENDPOINT` is set; Jaeger as the compose profile `tracing` locally.
  - [x] `internal/platform/tracing` (`Setup` / `Install`, W3C propagation, sampler and resource from `OTEL_*`),
    `otelgrpc`, `otelhttp` (span names by route), `otelpgx` (no parameters), `trace_id` / `span_id` in logs; health
    checks not traced.
  - [x] GraphQL spans from an in-repo extension (`internal/delivery/http/graphql/tracing`) instead of otelgqlgen,
    which always records the query text and argument literals (an inline `idToken` would reach the backend).
  - [x] Operation spans named from the document (`query TraceMe { … }`) when the request has no `operationName`.
  - [x] `make docker-up tracing=1` (profile + endpoint together), so `.env`'s endpoint is for local mode only.
  - [ ] Production tracing backend and sampling ratio, with the deploy target.

### API design
- [x] Pagination on `expenses` / `ListExpenses`: keyset on `(transaction_dt, id)`, opaque cursor, default 20 and
  max 50; GraphQL `ExpenseConnection` (`nodes`, `pageInfo`), gRPC page tokens per
  [AIP-158](https://google.aip.dev/158). Categories stay unpaged (short per-user list); page them the same way if
  that changes.
- [x] GraphQL custom scalars `Date` and `Decimal` instead of `String` for `transactionDt` and `amount`, and `UUID`
  instead of `ID` for identifiers (`internal/delivery/http/graphql/scalar`, function-based, mapped in `gqlgen.yml`).
- [x] Idempotency key on `CreateExpense` (`request_id` per [AIP-155](https://google.aip.dev/155), `requestId` in
  GraphQL): migration `000007_expense_request_id`, unique per user; a retry returns the first expense, and a reused
  key with a different payload is `INVALID_ARGUMENT` / `BAD_USER_INPUT`. Known edge cases: a key is freed when its
  expense is deleted, and a retry after the expense was updated no longer matches (so it's rejected).
- [ ] Rate limiting on writes: 15 per user per minute, shared by gRPC and GraphQL, single instance. Full spec in
  [rate-limiting.md](rate-limiting.md). (Today only Telegram binding is limited.)
- [x] Normalize CRUD in the repositories and usecases (internal only, so no client impact; the end-to-end tests cover
  the behaviour). The behaviour rules below already hold; what's left is consistency:
  - [x] The owner is always an argument, never part of a form (`CreateExpense(ctx, userID, form)`; `ExpenseForm`
    has no `UserID`).
  - [x] Usecase creates take a form, like updates take a patch (`CreateCategory(ctx, userID, form)`,
    `CreateExpense(ctx, userID, form)`); GraphQL create arguments are named `form`, updates `patch`.
  - [x] `errors.AsType[*pgconn.PgError]` everywhere.
  - [x] Removed the unused `UserUsecase.CreateUser` / `GetUserByGoogleSub` (`AuthUsecase` uses the repository).
  - [x] Receiver `uc` in every usecase; repository fields named after their entity (`uc.expense`, `uc.category`).
  - [x] Shared page request: every list usecase takes `pagination.Request` and cuts with `pagination.Cut`
    (`ListExpenses(ctx, userID, domain.ExpenseFilter, pagination.Request)`, `ListCategories(ctx, userID,
    pagination.Request)`).
  - Create / Update return the stored object, read back after the write when it has derived fields.
  - Get never returns `(nil, nil)`: missing, or another user's, is `errDom.ErrNotFound`.
  - List returns an empty slice (not `nil`, not an error) for no results; unbounded lists are paginated.
  - Delete returns only `error`, with `ErrNotFound` for a missing row; `ErrConflict` when it's in use.
  - [x] One update rule (patch) applied the same way to every form/patch.
- [x] gRPC input errors in one format: the handlers parse UUIDs with `converter.ParseUUID` / `ParseOptionalUUID`, so a
  malformed ID is `INVALID_ARGUMENT` with exactly `id: must be a UUID` (or `category_id: …`) instead of the
  library's error text.
- [x] `LICENSE`: Apache License 2.0, with the copyright notice in `NOTICE`.
- [x] `CODEOWNERS`: `* @revirathya/finance-core` (team with access to the repo).
- [x] `SECURITY.md` (reports through GitHub private vulnerability reporting).
  - [ ] Enable private vulnerability reporting: repository Settings → Security → Private vulnerability reporting.
- [x] PR template (`.github/pull_request_template.md`).
- Dependabot or Renovate: not wanted for now.
- [x] Conventional commits: every commit (`.githooks/commit-msg`) and PR title (CI `pr-title`,
  `amannn/action-semantic-pull-request`) starts with a type. An automated CHANGELOG and releases can follow later;
  the CHANGELOG stays hand-written for now.
- [x] `make generate` (all three codegens), `make hooks`, and a pre-commit hook (`gofmt`, `go vet`, codegen drift).
- [x] ADRs in [`docs/adr/`](adr/README.md): two APIs, exact decimals, keyset pagination, patch updates, idempotent
  creates, layering, and the temporary compatibility rule.
- [x] Branch protection on `main`, as two rulesets so CI can never be skipped:
  - `main` (**no bypass**): PR required (squash only, 0 approvals), required checks `test`, `migration-test`,
    `codegen`, `pr-title`; no force push or deletion.
  - `main-review`: 1 approval with code-owner review; bypass for organization and repository admins, pull requests
    only, because the team has one member and GitHub doesn't count the author's own approval.
  - [ ] When `finance-core` has a second member: remove the bypass from `main-review`.
  - [x] Repository settings: squash commit title "Pull request title"; merge and rebase commits disabled.

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
| Go clients | `go get` this module and import `gen/grpc/financecore/v1` |
| Non-Go clients | `buf generate "https://github.com/revirathya/revirathya-finance-core.git#subdir=proto"` |
| Several teams or languages (later) | Buf Schema Registry: versioned modules, rendered docs, generated SDKs |

- [x] **Server reflection** (`google.golang.org/grpc/reflection`), so `grpcurl`, Postman and Evans can
  discover services without proto files.
  - Done: gated by `GRPC_ENABLE_REFLECTION` (default `false`); set it to `true` in the local `.env`.
  - Keep it off in production, at least until auth is in place.
- [x] **`make grpc-list` / `make grpc-describe`** read a buf-built descriptor set (`bin/financecore.binpb`, includes the
  `google/type` imports): no server and no reflection needed. `GRPC_PROTOS` removed.
- [x] **Proto doc comments** on every service, RPC, message and field.
  - Comments must sit directly above the element (no blank line), or tools ignore them.
  - Remove detached section labels such as `// Common` and `// RPC: ...`.
  - Document what types don't say: formats (UUID), constraints (unique, max length), ordering, error codes.
  - Error codes can be documented now: the gRPC error mapping (section 1) is fixed.
  - [x] `categories.proto` (error codes to be added after the gRPC error mapping fix)
  - [x] `expenses.proto` (error codes to be added after the gRPC error mapping fix)
  - [x] `user_telegram_contexts.proto` (error codes to be added after the gRPC error mapping fix; refine the
    meaning of `conversation` / `state` / interaction IDs once the bot's usage is settled)
- [ ] Later, once other services depend on the API: **`buf breaking` in CI**, the main guard for their clients.
- [ ] Later, at the same point: **evolution rules** written down: mark removed fields as `reserved` (number and name), never reuse field
  numbers, and introduce `financecore.v2` for breaking redesigns while keeping `v1` running.
- [ ] Optional: render docs with `protoc-gen-doc`, or rely on Buf Schema Registry's rendered docs.
- [ ] Later: **Buf Schema Registry** (`buf push`) once more than one client team or language depends on the
  API.

### GraphQL
- [x] **Introspection** gated by `GQL_ENABLE_INTROSPECTION` (default `false`).
- [x] **Descriptions** on SDL types and fields; the dev playground shows them as documentation.
- [x] **Playground** at `/playground`, enabled only with `APP_ENV=dev`.
- Schema registry (Hive, Apollo GraphOS): skip unless there are several GraphQL consumers or federation.

## 5. Later (after the production-readiness checklist)

- [x] **Make `description` non-nullable, with `""` to clear it.** Done in `000005_expense_description_not_null`. Today `expenses.description` is nullable, and
  omitting it in `UpdateExpense` keeps the current value (`COALESCE`), so there's no way to clear it. Change it to
  `TEXT NOT NULL DEFAULT ''`: clients clear it by sending `""`, and omitting it still keeps the current value.
  - Migration: backfill existing `NULL`s to `''`, then `SET DEFAULT ''` and `SET NOT NULL`; the down migration
    drops both. Add it to `sqlc.yaml`.
  - `CreateExpense` inserts `NULL` when no description is sent, which a `NOT NULL` column rejects: use
    `COALESCE(sqlc.narg('description'), '')` in the query (or default to `""` before inserting).
  - After `make sqlc-generate`, `description` in the generated rows becomes `string` instead of `*string`; adjust
    the repository mapping. Input can keep `*string` (`nil` = keep, `""` = clear).
- [ ] **Enable `govulncheck` in CI before the first deploy.** It's commented out in
  [`.github/workflows/ci.yaml`](../.github/workflows/ci.yaml) until a deployment target exists. Uncomment it, pin
  the latest release, and fix any reported vulnerabilities before the service goes live.

## 6. Suggested order

Done so far: bug fixes, auth, health checks, error mapping, timeouts and limits, logging, recovery, GraphQL
hardening, input validation, CI with lint and drift checks, metrics, pagination, scalars, delivery tests,
breaking API cleanups (typed proto dates/amounts, shared messages, category pagination, patch updates),
integration docs (contract comments, `APP_ENV`, playground, "Integrating a new service"), and the usecase layer
no longer depending on `internal/repository` (`domain.Repository`), gRPC input errors in one format
(`converter.ParseUUID`), CRUD normalized across repositories and usecases (owner as argument, forms and patches,
`pagination.Request` / `Cut`), and consumer-side usecase interfaces in both APIs.

Next, in this order:

1. ~~`converter.ParseUUID`~~ (done).
2. ~~Normalize CRUD~~ (done).
3. ~~Idempotency key~~ (done).
4. ~~Repo hygiene~~ (done; Dependabot skipped for now).
5. ~~OpenTelemetry tracing~~ (done; production backend and sampling decided with the deploy target).
6. **Rate limiting on writes**, per [rate-limiting.md](rate-limiting.md).
7. **Trivy** image scanning (section 3, Docker).

Optional, as more entities are added: generic paging types (`domain.Page[T]`, one cursor helper) instead of a
`*Cursor` / `*ListQuery` / `*Page` per entity.

Waiting on outside decisions:

- **Before a second gRPC client:** per-service credentials (section 2, Security).
- **If a read replica is added:** route read-only queries (lists, gets) to it, and keep writes, and reads that must
  see a write just made (Create / Update read back), on the primary. The single pool in `platform/database` becomes
  a primary pool plus a replica pool.
- **When the deploy target is chosen:** TLS, secrets, migrations as a separate job, `govulncheck`, the runbook
  (deploy steps, migration rollback), and shared rate-limit counters if there will be several instances.
- **When the first consumer depends on the API:** `buf breaking`, written evolution rules, and revisiting the
  CLAUDE.md compatibility rule.
