# Rate limiting: write operations

Status: **specified, not implemented** (item 6 of the plan in
[production-readiness.md](production-readiness.md#6-suggested-order)). Each section marks what is decided and what
is still open, so the spec can be adjusted before or after it's built.

## Goal

Stop a single user (or a misbehaving client acting for them) from flooding the database with writes: a retry loop,
a stuck bot, or a script. It is not abuse protection for anonymous traffic; that needs limits per IP or per API key
in front of auth, and is out of scope here.

## Decisions

| Topic | Decision | Notes |
|:--|:--|:--|
| What is limited | **Writes only** | Reads stay unlimited for now. |
| Limit | **15 writes per user per minute** | One budget per user, shared by gRPC and GraphQL. |
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
results (GraphQL runs mutation fields one after another).

**`UpdateUserTelegramContext` is excluded.** The bot calls it on every step of a conversation to store its own
state, so a normal "add an expense" flow can take several calls; counting them would throttle ordinary use before
any data is written. *Open:* give it a separate, higher limit (e.g. 120 per minute) if the bot ever loops on it.

## Response when the limit is hit

| API | Response |
|:--|:--|
| gRPC | `RESOURCE_EXHAUSTED`, message `too many write requests, try again later` |
| GraphQL | An error with `extensions.code: "RATE_LIMITED"` on the mutation field, same message; HTTP status stays `200`, like every other GraphQL error |

*Open:* tell clients when to retry: gRPC `retry-after` response metadata and GraphQL `extensions.retryAfter`, both in
whole seconds until the user's window resets. The limiter would need to return the window's start, not only a
`bool`.

## Where it runs

- **gRPC:** a unary interceptor, `interceptor/ratelimit`, **after `telegramuser`** (so the user is known), checking
  `info.FullMethod` against the list of counted RPCs.
- **GraphQL:** a gqlgen root-field middleware (`AroundRootFields`), counting only fields on the `Mutation` type that
  are in the list. Running in gqlgen rather than as HTTP middleware is what makes "each mutation field counts" and
  a per-field error possible.
- **One shared `ratelimit.Limiter`**, created in `cmd/main.go` and passed to both, so the budget is per user across
  both APIs.

The counted operations are listed in code next to each check (a set of full method names for gRPC, of mutation field
names for GraphQL), so adding a write operation means adding it to that list; a test should fail when a new
mutation or write RPC isn't classified as counted or excluded.

## Configuration

The limit and window are **constants in code** for now (`writeLimit = 15`, `writeWindow = time.Minute`). *Open:*
move them to config (`WRITE_RATE_LIMIT`, `WRITE_RATE_WINDOW`) when there is a reason to change them per deployment.

## Observability

- A counter `rate_limited_total{api}` (`api` is `grpc` or `graphql`; bounded labels only, never the user ID).
- The access log line already records the resulting code (`ResourceExhausted` / the GraphQL error).
- Rejected calls are client errors: not logged as errors.

## Multiple instances

With N replicas, each keeps its own counters, so a user can make up to N × 15 writes per minute, and the limit
"resets" when a request lands on a different replica. Acceptable for one instance; before running several, move the
counters to a shared store (Redis `INCR` + `EXPIRE` per user and window is enough for a fixed window). That choice
belongs with the deploy target (step 4).

## Known limits of a fixed window

A user can make 15 writes at the end of one window and 15 at the start of the next, so up to 30 in a short burst.
That's fine for flood protection. If smoothness matters later, switch to a sliding window or token bucket inside
`platform/ratelimit` without changing the callers.

## Tests to write with it

- 15 writes succeed, the 16th fails with `RESOURCE_EXHAUSTED` / `RATE_LIMITED`; after the window (fake clock) the
  user can write again.
- The budget is per user (another user is unaffected) and shared across gRPC and GraphQL.
- Reads, `loginWithGoogle`, `BindTelegramAccount`, `UpdateUserTelegramContext` and health checks never count.
- A document with several mutations counts each one, and the ones past the limit fail individually.
- Every mutation field and every gRPC write method is classified as counted or excluded.
