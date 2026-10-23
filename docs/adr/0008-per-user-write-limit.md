# 0008. A per-user write limit, shared by both APIs

**Status:** Accepted

## Context

A single user, or a misbehaving client acting for them, can flood the database with writes: a retry loop, a stuck
bot, or a script. Both APIs write the same data for the same user, so a limit on one API alone is easy to get around
by accident. This is not protection against anonymous abuse; that needs limits per IP or per API key in front of
authentication, and is out of scope.

## Decision

Every user gets **one write budget per window, shared by gRPC and GraphQL**: 15 writes per minute by default
(`WRITE_RATE_LIMIT`, `WRITE_RATE_WINDOW`; invalid values are startup errors).

- **Key:** the user ID from the principal, so the check runs after authentication; unauthenticated calls are rejected
  before they count.
- **Algorithm:** a fixed window in memory (`platform/ratelimit`), starting at the user's first write. One limiter is
  built in `cmd/main.go` and passed to both APIs. A call the limiter rejects doesn't count.
- **What counts:** writes only.

  | API | Counted | Not counted |
  |:--|:--|:--|
  | gRPC | `CreateExpense`, `UpdateExpense`, `DeleteExpense` | reads; health checks; `BindTelegramAccount` (its own limit: 10 attempts per Telegram account per 5 minutes); `UpdateUserTelegramContext` (the bot stores its conversation state on every step, so counting it would throttle normal use before any data is written) |
  | GraphQL | every `Mutation` field except the two on the right, **each field counted**, so a document with three mutations uses three writes | `loginWithGoogle` (unauthenticated), `ok`, every query |

  The lists live next to the checks (`dlvGrpc.RateLimitedMethods`, `dlvGql.RateLimitedMutations`), and tests fail
  until every RPC and mutation is classified as counted or not.
- **Where:** gRPC in the `writelimit` interceptor, last in the chain (after `telegramuser`, which sets the principal);
  GraphQL in a gqlgen field middleware (`AroundFields`), which makes per-field counting and a per-field error
  possible.
- **Response:** gRPC `RESOURCE_EXHAUSTED` with `retry-after` response metadata; GraphQL an error on the mutation field
  with `extensions.code: "RATE_LIMITED"` and `extensions.retryAfter` (HTTP status stays `200`). Both say `too many
  write requests, try again later`; the wait is whole seconds until the window resets, rounded up, at least 1. The
  delivery layers return `*errDom.RateLimitError{RetryAfter}`, and the two error mappers turn it into these responses.
- **Observability:** `rate_limited_total{api}` (`grpc` / `graphql`). Rejections are client errors, not logged as
  errors.

## Consequences

- A runaway client is stopped after 15 writes a minute, whichever API it uses, and is told when to retry.
- A fixed window allows a burst of up to twice the limit across a window boundary (15 at the end of one window, 15
  at the start of the next). That's fine for flood protection; a sliding window or token bucket can replace it inside
  `platform/ratelimit` without changing the callers.
- With N instances, each keeps its own counters, so a user gets up to N × 15 writes a minute. Running more than one
  instance needs shared counters (Redis `INCR` + `EXPIRE` per user and window is enough for a fixed window); that
  waits for the deploy target.
- When a rate-limited GraphQL mutation has a non-null result (`deleteExpense`, `updateUser`, ...), GraphQL's null
  rule makes the whole `data` `null`, although the earlier mutations in the document were written.
- Every new write RPC or mutation has to be classified; the classification tests make that impossible to forget.
- Open: if the bot ever loops on `UpdateUserTelegramContext`, give it a separate, higher limit (e.g. 120 a minute).
