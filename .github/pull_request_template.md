## What and why

<!-- What changes, and why. Link the issue or task if there is one. -->

## Checklist

- [ ] The PR title follows Conventional Commits: `type(scope): summary`, e.g. `feat(expense): add request_id`
      (types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `build`, `perf`, `revert`; `!` marks a
      breaking change)
- [ ] `make lint` and `make test-integration` pass
- [ ] Generated code is regenerated (`make generate`) if `proto/`, `graphql/schema/`, `sqlc/queries/` or
      `migrations/` changed
- [ ] Migrations have an up and a down, the up file is listed in `sqlc.yaml`, and a data migration has a test in
      `test/migrations/`
- [ ] Contract docs updated: proto comments and GraphQL descriptions for any changed field, argument or error code
- [ ] Docs updated: README (API behaviour), CHANGELOG, `docs/` if a decision changed (new ADR in `docs/adr/`)
- [ ] No secrets, tokens or DSNs in code, tests, logs or this description
