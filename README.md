# Revirathya - Finance Core

## Overview
Core services to managing financial data. Exposes gRPC and GraphQL as APIs backed by Postgres.

**Stack:**
- Go 1.25
- pgx
- gRPC
- gqlgen (GraphQL)
- sqlc
- golang-migrate

## Prerequisites
- Go 1.25
- Go Tools: `buf`, `sqlc`, `golang-migrate`
- Docker

## Quickstart
```sh
cp .env.example .env # After copy, manually adjust .env values

make up # Start services
make down # Stop services
```

## Configuration

|Var|Description|
|:--|:--|
|`DATABASE_URI`|Postgres DSN, e.g. postgres://revirathya:revirathya@localhost:5432/finance?sslmode=disable|
|`GRPC_PORT`|gRPC listen port (e.g., 50051)|
|`GQL_PORT`|GraphQL listen port (e.g., 8080)|
|`ALLOWED_ORIGINS`|Comma-separated CORS origins for GraphQL|

## Migrations
- Local: `make migrate-up` / `make migrate-down` / `make migrate-create=<X>`
- Docker: built from `migrations.Dockerfile` (`migrate/migrate` + bundled `migrations`)

## Docker
```sh
make docker-build
docker compose up
```

## Development
```sh
make proto-generate # buf: gRPC stubs
make sqlc-generate  # sqlc: query code
make gql-generate   # gqlgen: GraphQL resolvers
make check          # vet + test + proto-lint
```

## Changelog

See [CHANGELOG.md](CHANGELOG.md).
