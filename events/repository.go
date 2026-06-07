package events

import (
	"context"
	"strings"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5/pgxpool"
)

func listUpcomingEvents(ctx context.Context, db *pgxpool.Pool, citySlug string, limit int) ([]Event, error) {
	rows, err := dbgen.New(db).ListUpcomingEvents(ctx, dbgen.ListUpcomingEventsParams{
		CitySlug: optionalCitySlug(strings.TrimSpace(citySlug)),
		LimitVal: int32(limit),
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list upcoming events")
	}
	return mapUpcomingRows(rows), nil
}

func listExternalEvents(ctx context.Context, db *pgxpool.Pool, citySlug string, limit, offset int) ([]Event, int64, bool, error) {
	q := dbgen.New(db)
	slug := optionalCitySlug(strings.TrimSpace(citySlug))

	rows, err := q.ListExternalEvents(ctx, dbgen.ListExternalEventsParams{
		CitySlug:  slug,
		LimitVal:  int32(limit),
		OffsetVal: int32(offset),
	})
	if err != nil {
		return nil, 0, false, errs.WrapCode(err, errs.Internal, "failed to list events")
	}

	total, err := q.CountExternalEvents(ctx, slug)
	if err != nil {
		return nil, 0, false, errs.WrapCode(err, errs.Internal, "failed to count events")
	}

	events := mapExternalRows(rows)
	hasMore := int64(offset+len(events)) < total
	return events, total, hasMore, nil
}

func mapUpcomingRows(rows []dbgen.ListUpcomingEventsRow) []Event {
	events := make([]Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, mapUpcomingRow(row))
	}
	return events
}

func mapExternalRows(rows []dbgen.ListExternalEventsRow) []Event {
	events := make([]Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, mapExternalRow(row))
	}
	return events
}

func mapUpcomingRow(row dbgen.ListUpcomingEventsRow) Event {
	return Event{
		ID:             row.ID,
		EventName:      row.EventName,
		EventURL:       row.EventUrl,
		VenueName:      row.VenueName,
		VenueURL:       row.VenueUrl,
		ScheduleText:   row.ScheduleText,
		MusicText:      row.MusicText,
		Description:    row.Description,
		PosterImageURL: row.PosterImageUrl,
		ImportedAt:     timestamptzToString(row.ImportedAt),
		CitySlug:       row.CitySlug,
		CityName:       row.CityName,
		SourceSlug:     row.SourceSlug,
		SourceName:     row.SourceName,
	}
}

func mapExternalRow(row dbgen.ListExternalEventsRow) Event {
	return Event{
		ID:             row.ID,
		EventName:      row.EventName,
		EventURL:       row.EventUrl,
		VenueName:      row.VenueName,
		VenueURL:       row.VenueUrl,
		ScheduleText:   row.ScheduleText,
		MusicText:      row.MusicText,
		Description:    row.Description,
		PosterImageURL: row.PosterImageUrl,
		ImportedAt:     timestamptzToString(row.ImportedAt),
		CitySlug:       row.CitySlug,
		CityName:       row.CityName,
		SourceSlug:     row.SourceSlug,
		SourceName:     row.SourceName,
	}
}
