package events

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func timestamptzToString(v pgtype.Timestamptz) string {
	if !v.Valid {
		return ""
	}
	return v.Time.UTC().Format(time.RFC3339)
}

func optionalCitySlug(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
