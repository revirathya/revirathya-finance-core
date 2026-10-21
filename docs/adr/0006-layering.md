# 0006. Layering: domain at the centre, consumer-side interfaces

**Status:** Accepted

## Context

The service has two APIs, one database, and more entities coming. Without clear boundaries, protocol details
(gRPC status codes, GraphQL types) and storage details (pgx types, SQL errors) leak into the business logic, and
every change touches every layer.

## Decision

Clean-architecture layering under `internal/`, with dependencies pointing inward:

- **`domain`**: entities, forms and patches with `Validate()`, repository interfaces, domain errors, pagination.
  Imports nothing else from the project, and no frameworks.
- **`usecase`**: one method per operation; depends only on `domain` (`BootstrapUsecase` takes `domain.Repository`,
  never `internal/repository`).
- **`repository`**: implements the domain interfaces with Postgres; the only layer that knows SQL.
- **`delivery`**: gRPC and GraphQL; parses requests, takes the user from the auth context, maps results and errors.
- **`platform`**: protocol-neutral infrastructure (database pool, logging, metrics, Google, rate limiting).

Interfaces are declared by their **consumer**: `domain` holds the repository interfaces the usecases need, and each
delivery package declares the usecase methods it calls (`handler/usecases.go`, `resolver/usecases.go`). Ownership is
always an explicit argument taken from the principal, never part of the input.

## Consequences

- Business rules are tested without a database or a server; a new API or a storage change stays in one layer.
- Each entity is a vertical slice through the same layers (see the README's "Adding an entity").
- More files and some mapping code per entity, and a consumer interface to update when a handler needs a new method.
- `go list -deps ./internal/usecase/...` must never show pgx or `internal/repository`.
