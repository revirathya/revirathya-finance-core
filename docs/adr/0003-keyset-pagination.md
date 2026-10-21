# 0003. Keyset (cursor) pagination

**Status:** Accepted

## Context

Expense lists grow without bound. Offset pagination (`LIMIT 20 OFFSET 400`) gets slower on every page, because
the database still reads and discards the skipped rows, and it repeats or skips rows when expenses are added or
deleted between requests.

## Decision

Page every unbounded list by **keyset**: "the rows after the last one I saw", in a fixed order.

- A **total order** with a unique tiebreaker: expenses by `transaction_dt DESC, id DESC`; categories by
  `category_name` (unique per user, so no tiebreaker).
- An **index** that matches the order.
- An **opaque cursor**: the last row's sort key, base64url-encoded, validated completely when decoded.
- Fetch `size + 1` rows to know whether another page follows; the page size defaults to 20, capped at 50.
- Shared helpers: `pagination.Request`, `NormalizedSize`, and `pagination.Cut`.

On the wire: GraphQL `first` / `after` returning `nodes` + `pageInfo`; gRPC `page_size` / `page_token` returning
`next_page_token` (AIP-158).

## Consequences

- Every page costs the same, however deep, and concurrent writes don't shift results.
- Clients can't jump to page N; they can only go forward. That fits the use cases (infinite lists, "read all").
- Each new list needs a cursor type, an index matching its order, and the same `size + 1` pattern.
- Clients must stop on an empty next token, not on a short page.
