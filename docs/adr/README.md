# Architecture decision records

Each record explains one significant decision: the situation, what was decided, and what it costs. Records are
never rewritten after they're accepted; a decision that changes gets a new record that supersedes the old one.

| # | Decision | Status |
|:--|:--|:--|
| [0001](0001-two-apis.md) | Two APIs: gRPC for services, GraphQL for the Web Console | Accepted |
| [0002](0002-money-as-exact-decimals.md) | Money as exact decimals, never floats | Accepted |
| [0003](0003-keyset-pagination.md) | Keyset (cursor) pagination | Accepted |
| [0004](0004-updates-are-patches.md) | Updates are patches | Accepted |
| [0005](0005-idempotent-creates.md) | Idempotent creates with a per-user request ID | Accepted |
| [0006](0006-layering.md) | Layering: domain at the centre, consumer-side interfaces | Accepted |
| [0007](0007-no-downstream-compatibility-yet.md) | No compatibility guarantees until the first consumer | Accepted, temporary |

## Adding a record

Copy the shape of an existing one into `NNNN-short-title.md` with the next number, and add it to the table:

- **Status**: Proposed, Accepted, Superseded by NNNN, or Deprecated.
- **Context**: the problem and the forces at play, written so someone new understands why it mattered.
- **Decision**: what was chosen, in the active voice.
- **Consequences**: what gets easier, what gets harder, and what to watch.
