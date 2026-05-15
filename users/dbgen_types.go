package users

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func uuidFromString(v string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(v)
	return id, err
}

func timestamptzValue(v pgtype.Timestamptz) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return v.Time.UTC()
}

func timestamptzPtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time.UTC()
	return &t
}

func dateFromString(v *string) pgtype.Date {
	if v == nil {
		return pgtype.Date{}
	}
	var d pgtype.Date
	if err := d.Scan(*v); err != nil {
		return pgtype.Date{}
	}
	return d
}

func dateToStringPtr(v pgtype.Date) *string {
	if !v.Valid {
		return nil
	}
	s := v.Time.UTC().Format("2006-01-02")
	return &s
}
