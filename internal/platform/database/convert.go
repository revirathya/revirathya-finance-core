package database

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// UUID
func FromPgxUUID(val pgtype.UUID) uuid.UUID {
	return uuid.UUID(val.Bytes)
}

func ToPgxUUID(val uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Valid: true,
		Bytes: val,
	}
}

func ToOptionalPgxUUID(val *uuid.UUID) pgtype.UUID {
	res := pgtype.UUID{Valid: val != nil}
	if res.Valid {
		res.Bytes = *val
	}
	return res
}

// Date
func FromPgxDate(val pgtype.Date) time.Time {
	return val.Time
}

func ToPgxDate(val time.Time) pgtype.Date {
	return pgtype.Date{
		Valid: true,
		Time:  val,
	}
}

func ToOptionalPgxDate(val *time.Time) pgtype.Date {
	res := pgtype.Date{Valid: val != nil}
	if res.Valid {
		res.Time = *val
	}
	return res
}

// Timestamp
func ToPgxTimestamp(val time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Valid: true,
		Time:  val,
	}
}

// Numeric
func FromPgxNumeric(val pgtype.Numeric) decimal.Decimal {
	return decimal.NewFromBigInt(val.Int, val.Exp)
}

func ToPgxNumeric(val decimal.Decimal) pgtype.Numeric {
	return pgtype.Numeric{
		Valid: true,
		Int:   val.Coefficient(),
		Exp:   val.Exponent(),
	}
}

func ToOptionalPgxNumeric(val *decimal.Decimal) pgtype.Numeric {
	res := pgtype.Numeric{Valid: val != nil}
	if res.Valid {
		res.Int = val.Coefficient()
		res.Exp = val.Exponent()
	}
	return res
}
