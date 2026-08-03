package stamps

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
	_ = n.Scan(strconv.FormatFloat(v, 'f', -1, 64))
	return n
}

func numericFromOptionalFloat(v *float64) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{}
	}
	return numericFromFloat(*v)
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

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
