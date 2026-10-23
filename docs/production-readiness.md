# Production readiness

What is still open before and after the first release, grouped by what each item waits on, and the documentation
still to write. Everything else is in place and documented where it lives: the [README](../README.md)
(configuration, APIs, limits, observability, CI), the [ADRs](adr/README.md) (why), and the
[runbooks](runbooks/README.md) (how).

## Waiting on the deploy target

Where and how the service runs (for example a VM with Docker Compose, Cloud Run, or Kubernetes) decides these.

- **TLS.** Encrypt both listeners (gRPC and HTTP), or terminate TLS at a trusted proxy or mesh, and record which in an
  ADR. Today both listen in plain text.
- **Secrets.** Move `JWT_SECRET`, `GRPC_API_KEY`, `DB_PASSWORD` and the Google client ID out of `.env` files into the
  platform's secret store, with a rotation procedure as a runbook.
- **Migrations as a deploy step.** Run the migration image as a separate job before the new version starts, instead
  of the compose `migration` service that runs on every start.
- **Deploy and rollback runbooks.** Write down the deploy steps, and how to roll back a release and a migration
  (`migrate down`, and recovering a migration left `dirty`).
- **Tracing backend.** Pick where OTLP traces go in production and a sampling ratio (`OTEL_TRACES_SAMPLER`,
  `OTEL_TRACES_SAMPLER_ARG`); Jaeger stays the local default.
- **Dashboards and alerts.** Build them on the existing metrics: error rate, latency, `rate_limited_total`, and
  `db_pool_acquire_wait_total` growth.
- **Shared write-limit counters.** With more than one instance, move the limiter's counters to a shared store such as
  Redis, or each instance allows its own 15 writes a minute ([ADR 0008](adr/0008-per-user-write-limit.md)).
- **Database pool settings as config.** Expose `DB_MAX_CONNS` (and maybe `statement_timeout`) once the database's
  size and connection limits are known; today they are constants in code.
- **Health checks for the platform.** If a load balancer or probe needs it, report `/readyz` as not ready as soon as
  shutdown starts, and tie gRPC health to the database.
- **Read replica routing,** if a replica is added: send lists and gets to it, and keep writes, and reads that must see
  a write just made, on the primary.

## Waiting on the first consumer

When another service or a front end depends on the API, its compatibility starts to matter.

- **Supersede [ADR 0007](adr/0007-no-downstream-compatibility-yet.md).** Replace "breaking changes are allowed" with
  the compatibility rules below, and update the matching rule in CLAUDE.md.
- **`buf breaking` in CI.** Check every PR against `main`, so a breaking proto change fails before it ships.
- **Proto evolution rules.** Mark removed fields `reserved` (number and name), never reuse field numbers, and introduce
  `financecore.v2` for redesigns while keeping `v1` running.
- **GraphQL deprecation.** Mark fields `@deprecated` and keep them until clients have moved, instead of removing them.
- **Per-service gRPC credentials.** Before a second gRPC client, give each calling service its own API key or client
  certificate, so each can be identified, limited and revoked on its own, and decide how a service names its user
  when it has no Telegram ID.
- **Published contracts,** once several teams or languages depend on the API: the Buf Schema Registry (`buf push`)
  with rendered docs and generated SDKs.

## Waiting on the team growing

The repository is set up for one maintainer; these change with a second.

- **Code review without a bypass.** Remove the admin bypass from the `main-review` ruleset, so every PR needs a code
  owner's approval from someone other than its author.
- **Dependency updates.** Turn on Dependabot or Renovate for Go modules, GitHub Actions and Docker base images, when
  there are people to review their PRs.
- **Automated releases.** Generate the CHANGELOG and GitHub releases from the Conventional Commits history (for
  example release-please), instead of writing them by hand.

## Documentation to write

Worth recording before or soon after the first release. They can start in `docs/adr/` and `docs/runbooks/` and move
to the team's central documentation site once it exists, so this repository keeps only what is specific to it.

ADRs, for decisions already made that a newcomer would ask about:

- **Authentication.** Web sign-in exchanges a Google ID token for the service's own session token; the bot uses one
  shared API key plus Telegram binding; another user's record is `NOT_FOUND`, not `FORBIDDEN`.
- **Telemetry stays private.** Tracing is opt-in; spans carry no SQL parameters, GraphQL query text, variables or
  arguments (hence the in-repo GraphQL tracing extension instead of otelgqlgen); metrics are on their own port with
  bounded labels; logs never carry secrets, tokens, headers or bodies.
- **Supply chain.** Images are scanned with Trivy; trivy-action is pinned to a commit and the Trivy image to a digest;
  `migrate` is built from this module rather than taken from `migrate/migrate`; Go is a patch-level minimum in
  `go.mod`, moved together with the Dockerfiles.

Runbooks that don't need the deploy target:

- **Rotate a secret.** `JWT_SECRET` (signs everyone out), `GRPC_API_KEY` (one shared key, so the bot must switch at
  the same time), and the Google OAuth client.
- **Fix a failing image or vulnerability scan.** Bump Go (`go.mod` and both Dockerfiles) or the dependency, rebuild,
  `make image-scan`; or accept the finding in `.trivyignore` with an expiry and a reason.
- **Recover a failed migration locally.** `migrate version`, `force` and `down` when a migration is left `dirty`; the
  production version comes with the deploy runbooks.
