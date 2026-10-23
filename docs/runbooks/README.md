# Runbooks

Step-by-step procedures for recurring tasks: what to do, in what order, and how to tell it worked. Decisions and their
reasons live in the [ADRs](../adr/README.md); a runbook only says how.

| # | Runbook | Use it when |
|:--|:--|:--|
| [0001](0001-postman-google-login.md) | Get a GraphQL session token from Postman | Calling the GraphQL API by hand, without the Web Console |

## Adding a runbook

Copy the shape of an existing one into `NNNN-short-title.md` with the next number, and add it to the table:

- **Title**: the task, as an action ("Get a…", "Rotate…", "Roll back…").
- **When / Result / Needs**: when to use it, what you have at the end, and what access or tools it takes.
- **Steps**: numbered, each with the exact command or screen, and how to check it worked.
- **Troubleshooting**: symptoms, causes and fixes, when there are known failure modes.

Unlike ADRs, runbooks are kept up to date: change one whenever the procedure changes.
