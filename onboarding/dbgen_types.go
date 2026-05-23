package onboarding

import (
	"github.com/jackc/pgx/v5/pgtype"
)

func uuidFromString(v string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(v)
	return id, err
}

func numericFromFloat(v float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(v)
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
