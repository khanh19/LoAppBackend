package onboarding

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

func uuidFromString(v string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(v)
	return id, err
}

func numericFromFloat(v float64) pgtype.Numeric {
	var n pgtype.Numeric
	// pgx v5 Numeric.Scan rejects float64; use a decimal string instead.
	_ = n.Scan(strconv.FormatFloat(v, 'f', -1, 64))
	return n
}

func numericToFloat(v pgtype.Numeric) float64 {
	if !v.Valid {
		return 0
	}
	f, err := v.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
