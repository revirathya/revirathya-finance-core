# 0002. Money as exact decimals, never floats

**Status:** Accepted

## Context

Binary floating point can't represent most decimal amounts exactly: `0.1 + 0.2` is `0.30000000000000004`. For a
finance service, rounding errors that accumulate across sums, or an amount that changes on a round trip through a
client, are unacceptable.

## Decision

Keep amounts as exact decimals at every step:

- **Database:** `NUMERIC(14, 2)`: up to 12 digits before the point and 2 after.
- **Go:** `github.com/shopspring/decimal`.
- **gRPC:** `google.type.Decimal` (a string value).
- **GraphQL:** a `Decimal` scalar, always a JSON string; a JSON number is rejected.

Validation enforces the column's limits before the database does: greater than 0, at most 2 decimal places, below
10¹².

## Consequences

- No float ever touches an amount, so totals and round trips are exact.
- Clients must send amounts as strings and parse them with a decimal type, not a float.
- Changing the precision means a migration and updating `ValidateDecimal`'s precision and scale together.
- One currency is assumed; multi-currency would need `google.type.Money` and a currency column.
