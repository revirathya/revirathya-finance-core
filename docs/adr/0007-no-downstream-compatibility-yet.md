# 0007. No compatibility guarantees until the first consumer

**Status:** Accepted, temporary

## Context

No service or front end depends on this API yet. Keeping every change backward compatible now (reserved proto
fields, deprecated GraphQL fields, versioned migrations for old clients) would slow the design down while the API is
still taking shape, and protect nobody.

## Decision

Until the first consumer depends on the API, breaking changes are allowed everywhere: gRPC (including renumbering
fields), GraphQL, the database schema, config and Docker. Changes aren't marked as breaking and don't carry follow-up
tasks for other services. What a new integrator needs is documented instead: the README's "Integrating a new
service", and comments and descriptions in the contracts.

## Consequences

- API cleanups are cheap now, so they were done before the first consumer (typed dates and amounts, shared messages,
  patch updates, pagination).
- **This record must be superseded when the first consumer depends on the API.** At that point: turn on
  `buf breaking` in CI, write the proto evolution rules (reserve removed fields, never reuse numbers, `v2` for
  redesigns), mark GraphQL fields `@deprecated` before removing them, and update the compatibility rule in CLAUDE.md.
