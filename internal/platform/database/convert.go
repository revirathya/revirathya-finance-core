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

func ToNullablePgxDate(val *time.Time) pgtype.Date {
	res := pgtype.Date{Valid: val != nil}
	if res.Valid {
		res.Time = *val
	}
	return res
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
