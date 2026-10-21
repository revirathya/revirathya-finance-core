# 0004. Updates are patches

**Status:** Accepted

## Context

Updates used to require every field except the description, which could be omitted to keep it. That mixed rule
was hard to explain, and it made the common case awkward: the bot often changes a single field, such as only the
amount, but had to resend everything else.

## Decision

Every update is a **patch**:

- A field that is **omitted** (or `null` in GraphQL) keeps its current value.
- A field that is **sent** is validated like on create and replaces the value.
- `""` **clears** clearable text (`description`, `lastName`).
- An update with no fields returns the object unchanged.

How "omitted" is expressed:

- **gRPC:** proto3 `optional` and message fields (presence), not `google.protobuf.FieldMask`.
- **GraphQL:** nullable input fields.
- **Domain:** `*Patch` structs with pointer fields; `Validate()` checks only the set ones.
- **SQL:** `COALESCE(sqlc.narg('x'), x)` per column, and `NULLIF(…, '')` where `""` must store `NULL`.

## Consequences

- One rule for every entity, and partial updates are natural for clients.
- A field can't be "set to its zero value" except through the `""` rule; no field needs that today.
- A field mask would be needed if a non-text field ever has to be cleared or reset explicitly.
