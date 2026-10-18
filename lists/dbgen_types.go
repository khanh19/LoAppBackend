package lists

import (
	"strconv"
	"strings"

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

func numericPtr(v pgtype.Numeric) *float64 {
	if !v.Valid {
		return nil
	}
	f, err := v.Float64Value()
	if err != nil || !f.Valid {
		return nil
	}
	out := f.Float64
	return &out
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// libraryCoverURL drops ephemeral Google CDN URLs and empty covers.
func libraryCoverURL(cover *string) *string {
	if cover == nil {
		return nil
	}
	v := strings.TrimSpace(*cover)
	if v == "" || strings.Contains(strings.ToLower(v), "googleusercontent.com") {
		return nil
	}
	return &v
}
