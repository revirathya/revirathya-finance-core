# 0005. Idempotent creates with a per-user request ID

**Status:** Accepted

## Context

When a create's response is lost (a network blip, a timeout, a client restart), the client retries, and the user
ends up with two identical expenses. Categories don't have this problem, because a retried create hits the unique
name and gets `CONFLICT`; expenses have no natural key.

## Decision

`CreateExpense` accepts an optional **request ID** (`request_id` on gRPC, `requestId` on GraphQL), following AIP-155:

- The client generates a UUID **once per user action** and reuses it on every retry.
- It's stored on the expense, unique **per user** (a partial unique index, `WHERE request_id IS NOT NULL`).
- `INSERT … ON CONFLICT DO NOTHING`; on a conflict, the existing expense is returned. Postgres serializes
  concurrent retries, so exactly one row is created.
- The usecase compares the stored expense with the request; a reused key with a **different** payload is
  `INVALID_ARGUMENT` / `BAD_USER_INPUT`, since it's most likely a client bug that would otherwise lose an expense.

## Consequences

- Retries are safe without any client-side bookkeeping beyond keeping the key.
- Clients that don't send a key behave as before.
- Edge cases accepted: deleting the expense frees its key, so a late retry creates a new one; a retry after the
  expense was updated no longer matches and is rejected. Fixing the second would need storing the original request.
- Other creates (categories) don't need a key today; add one the same way if they ever do.
