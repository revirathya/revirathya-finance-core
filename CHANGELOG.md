# Changelog

All notables changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Pre-1.0 - APIs, GraphQL schema, and gRPC contracts may change without notice.

## [Unreleased]

### [0.0.1] - 2026-08-09
Initial release.

#### Database
- Postgres schema (migration `00001_init_schema`):
    - `users` - user information
    - `user_telegram_contexts` - per-user context for Telegram interactions
    - `categories`- expenses categories
    - `expenses` - expense (transaction) records
- Dev-only seed migration (`000002_temp_seed`) - inserts sample categories, one user, and one Telegram context. Contains hardcoded data; remove before first real deploy and inject user data manually.

### Categories
- Use cases: create, get by ID, list, update, delete
- GraphQL:
    - Queries: `category`, `categories`
    - Mutations: `createCategory`, `updateCategory`, `deleteCategory`
- gRPC (`CategoryService`):
    - Read Only: `ListCategories`, `GetCategoryByID`

### Expenses
- Use cases: create, get by ID, list, update, delete
- GraphQL:
    - Query: `expense`, `expenses`
    - Mutations: `createExpense`, `updateExpense`, `deleteExpense`
- gRPC (`ExpenseService`):
    - Read Only: `ListExpenses`,
    - Write: `UpdateExpense`, `DeleteExpense`

### Users
- Use cases: get by ID
- GraphQL:
    - Query: `user`

### User Telegram Context
- Use cases: get by user ID, get by telegram ID, update
- gRPC (`UserTelegramContextService`):
    - Read only: `GetUserTelegramContext`, `GetUserTelegramContextByTelegramID`
    - Write: `UpdateUserTelegramContext`

## Infrastructure
- Mutil-stage Docker build (distroless runtime)
- Dedicated migration image (`migrations.Dockerfile`, based on `migrate/migrate`)
- `docker-compose.yml` for local Postgres + migrations + service
- Makefile targets for build, test, codegen (buf / sqlc / gqlgen), and migrations

[Unrelease]: https://github.com/revirathya/revirathya-finance-core/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/revirathya/revirathya-finance-core/releases/tag/v0.0.1