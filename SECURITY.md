# Security policy

## Reporting a vulnerability

Please **don't** open a public issue for a security problem. Report it privately through GitHub's
[private vulnerability reporting](https://github.com/revirathya/revirathya-finance-core/security/advisories/new)
(the repository's **Security** tab → **Report a vulnerability**).

Include what you found, how to reproduce it, and what an attacker could do with it. You'll get an acknowledgement,
and a fix or a plan, before anything is disclosed publicly.

## Scope

In scope: this service's code, its gRPC and GraphQL APIs, authentication (API keys, session tokens, Telegram
binding), and its Docker images.

Out of scope: denial of service by volume, findings that need a compromised server or database, and issues in
third-party dependencies that are already publicly known (those are tracked through Dependabot).

## Supported versions

Only the latest version on `main` receives security fixes.
