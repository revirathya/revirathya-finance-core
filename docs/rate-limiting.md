# Rate limiting: write operations

Status: **implemented** (item 6 of the plan in
[production-readiness.md](production-readiness.md#6-suggested-order)). Each section marks what is decided and what
is still open.

## Goal

Stop a single user (or a misbehaving client acting for them) from flooding the database with writes: a retry loop,
a stuck bot, or a script. It is not abuse protection for anonymous traffic; that needs limits per IP or per API key
in front of auth, and is out of scope here.

## Decisions

| Topic | Decision | Notes |
|:--|:--|:--|
| What is limited | **Writes only** | Reads stay unlimited for now. |
| Limit | **15 writes per user per minute** | One budget per user, shared by gRPC and GraphQL. Configurable (see [Configuration](#configuration)). |
| Key | **The user ID** from the principal (`auth.UserIDFrom`) | So the check runs **after** authentication; unauthenticated calls are rejected before they count. |
| Deployment | **Single instance** | The limiter is in memory, per process (see [Multiple instances](#multiple-instances)). |
| Algorithm | **Fixed window**, `platform/ratelimit` as it is today | The window starts at the user's first write and lasts one minute. |

## What counts as a write

One limited call = one write. A call rejected by the limiter doesn't count.

| API | Counted | Not counted |
|:--|:--|:--|
| GraphQL | `createCategory`, `updateCategory`, `deleteCategory`, `createExpense`, `updateExpense`, `deleteExpense`, `updateUser`, `createTelegramBindCode`, `unbindTelegram` | `loginWithGoogle` (unauthenticated, see Goal), every query, `ok` |
| gRPC | `CreateExpense`, `UpdateExpense`, `DeleteExpense` | `BindTelegramAccount` (has its own limit: 10 attempts per Telegram account per 5 minutes), `UpdateUserTelegramContext` (see below), every read, health checks |

**GraphQL counts each top-level mutation field.** A request with three mutations in one document uses three writes;
if the budget runs out partway, the remaining fields fail with `RATE_LIMITED` and the earlier ones keep their
results (GraphQL runs mutation fields one after another). For a mutation whose result is non-null (`deleteExpense`,
`updateUser`, ...), GraphQL's null rule turns the whole `data` into `null` when it fails, but the earlier mutations
were still written.

**`UpdateUserTelegramContext` is excluded.** The bot calls it on every step of a conversation to store its own
state, so a normal "add an expense" flow can take several calls; counting them would throttle ordinary use before
any data is written. *Open:* give it a separate, higher limit (e.g. 120 per minute) if the bot ever loops on it.

## Response when the limit is hit

| API | Response |
|:--|:--|
| gRPC | `RESOURCE_EXHAUSTED`, message `too many write requests, try again later`, and `retry-after` response metadata |
| GraphQL | An error with `extensions.code: "RATE_LIMITED"` and `extensions.retryAfter` on the mutation field, same message; HTTP status stays `200`, like every other GraphQL error |

**Decided: clients are told when to retry.** `retry-after` (gRPC header metadata) and `retryAfter` (GraphQL) are whole
seconds until the user's window resets, rounded up, at least 1. The limiter's `Take` returns that duration, the
delivery layers return `*errDom.RateLimitError{RetryAfter}` (unwraps to `errDom.ErrRateLimited`), and the two error
mappers turn it into the responses above.

## Where it runs

- **gRPC:** a unary interceptor, `interceptor/writelimit`, **after `telegramuser`** (so the user is known), checking
  `info.FullMethod` against `dlvGrpc.RateLimitedMethods`.
- **GraphQL:** a gqlgen field middleware (`AroundFields`, package `graphql/middleware/writelimit`), counting only fields on the
  `Mutation` type that are in `dlvGql.RateLimitedMutations`. Running in gqlgen rather than as HTTP middleware is what
  makes "each mutation field counts" and a per-field error possible; a field middleware (not a root-field one) gives
  the error its field path.
- **One shared `ratelimit.Limiter`**, created in `cmd/main.go` and passed to both, so the budget is per user across
  both APIs.

The counted operations are listed in code next to each check (a set of full method names for gRPC, of mutation field
names for GraphQL), so adding a write operation means adding it to that list; a test should fail when a new
mutation or write RPC isn't classified as counted or excluded.

## Configuration

**Decided:** `WRITE_RATE_LIMIT` (positive integer, default `15`) and `WRITE_RATE_WINDOW` (positive Go duration, default
`1m`). Invalid values are startup errors.

## Observability

- A counter `rate_limited_total{api}` (`api` is `grpc` or `graphql`; bounded labels only, never the user ID).
- The access log line already records the resulting code (`ResourceExhausted` / the GraphQL error).
- Rejected calls are client errors: not logged as errors.

## Multiple instances

With N replicas, each keeps its own counters, so a user can make up to N × 15 writes per minute, and the limit
"resets" when a request lands on a different replica. Acceptable for one instance; before running several, move the
counters to a shared store (Redis `INCR` + `EXPIRE` per user and window is enough for a fixed window).
**Postponed** until the deploy target is known (one instance or several, and whether a managed Redis is available).

## Known limits of a fixed window

A user can make 15 writes at the end of one window and 15 at the start of the next, so up to 30 in a short burst.
That's fine for flood protection. If smoothness matters later, switch to a sliding window or token bucket inside
`platform/ratelimit` without changing the callers.

## Tests

- **gRPC** (`test/internal/delivery/grpc/handler/writelimit_integration_test.go`): writes past the limit are
  `RESOURCE_EXHAUSTED` with `retry-after`, and allowed again after the window (fake clock); updates and deletes
  count; the budget is per user; reads, `UpdateUserTelegramContext` and health checks don't count; every RPC is
  classified as counted or not.
- **GraphQL** (`test/internal/delivery/http/graphql/resolver/writelimit_integration_test.go`): `RATE_LIMITED` with
  `retryAfter`; each mutation field in a document counts and the ones past the limit fail on their own; the budget is
  per user and shared with gRPC; queries, `loginWithGoogle` and anonymous mutations don't count; every mutation is
  classified as counted or not.
- **Units:** `Limiter.Take` (wait until reset, denied events don't extend the window), `RateLimitError` (rounding),
  and the `WRITE_RATE_*` config values.
