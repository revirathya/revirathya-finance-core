# 0001. Two APIs: gRPC for services, GraphQL for the Web Console

**Status:** Accepted

## Context

The service has two kinds of clients with different needs:

- **Services**, starting with the Telegram bot and later other internal services. They want a typed contract,
  generated clients in several languages, efficient binary calls, and deadlines.
- **The Web Console** (and any future web front end). It wants to ask for exactly the fields a screen needs, in one
  round trip, from a browser, with a schema it can explore.

One API would serve one of them poorly: a browser can't easily call gRPC, and services gain little from GraphQL's
flexible queries while losing typed stubs and deadlines.

## Decision

Expose both, from one binary and one set of usecases:

- **gRPC** (`proto/`) for service-to-service calls. Authentication: an API key per caller plus, for the bot, the
  Telegram user.
- **GraphQL** (`graphql/schema/`) for web front ends. Authentication: a session JWT from Google sign-in. All future
  web consoles use GraphQL.

Both are thin delivery layers over the same usecases, so behaviour (validation, ownership, errors) is identical.

## Consequences

- Two contracts to keep in sync with each other and with the code; both are documented in place and regenerated.
- Shared rules (validation messages, pagination, patch updates) must be expressed in each API's idiom
  (`INVALID_ARGUMENT` vs `BAD_USER_INPUT`, `page_token` vs `after`).
- The end-to-end tests run every operation through both servers.
