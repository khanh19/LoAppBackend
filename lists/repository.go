package lists

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func listPlaceLists(ctx context.Context, db *pgxpool.Pool) ([]PlaceListSummary, error) {
	rows, err := dbgen.New(db).ListActivePlaceLists(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list place lists")
	}

	lists := make([]PlaceListSummary, 0, len(rows))
	for _, row := range rows {
		lists = append(lists, PlaceListSummary{
			ID:            row.ID,
			Slug:          row.Slug,
			Title:         row.Title,
			Category:      row.Category,
			Area:          row.Area,
			Occasions:     row.Occasions,
			CitySlug:      row.CitySlug,
			CityName:      row.CityName,
			EntryCount:    row.EntryCount,
			CoverImageURL: row.CoverImageUrl,
		})
	}
	return lists, nil
}

func getPlaceListBySlug(ctx context.Context, db *pgxpool.Pool, slug string) (*PlaceListDetail, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "slug is required"}
	}

	q := dbgen.New(db)
	list, err := q.GetPlaceListBySlug(ctx, slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &errs.Error{Code: errs.NotFound, Message: "list not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to get place list")
	}

	listUUID, err := uuidFromString(list.ID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "invalid list id")
	}

	entries, err := q.ListPlaceListEntriesByListID(ctx, listUUID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list place list entries")
	}

	detail := &PlaceListDetail{
		ID:         list.ID,
		Slug:       list.Slug,
		Title:      list.Title,
		Subtitle:   list.Subtitle,
		Category:   list.Category,
		Area:       list.Area,
		Occasions:  list.Occasions,
		ListType:   list.ListType,
		Visibility: list.Visibility,
		SavesCount: list.SavesCount,
		StopsCount: list.StopsCount,
		CitySlug:   list.CitySlug,
		CityName:   list.CityName,
		Entries:    make([]PlaceListEntry, 0, len(entries)),
	}
	if list.CreatorUserID != "" || list.CreatorDisplayName != nil || list.CreatorUsername != "" || list.CreatorAvatarUrl != nil {
		detail.Creator = &PlaceListCreator{
			UserID:      list.CreatorUserID,
			DisplayName: list.CreatorDisplayName,
			Username:    list.CreatorUsername,
			AvatarURL:   list.CreatorAvatarUrl,
		}
	}
	for _, row := range entries {
		detail.Entries = append(detail.Entries, PlaceListEntry{
			StopOrder:     row.StopOrder,
			SeedName:      row.SeedName,
			Note:          row.Note,
			TimeLabel:     row.TimeLabel,
			ActivityType:  row.ActivityType,
			ImageURL:      row.ImageUrl,
			PlaceID:       row.PlaceID,
			Name:          row.PlaceName,
			GooglePlaceID: row.GooglePlaceID,
			Address:       row.Address,
			Neighborhood:  row.Neighborhood,
			Latitude:      numericToFloat(row.Latitude),
			Longitude:     numericToFloat(row.Longitude),
			Rating:        numericToFloat(row.RatingCached),
			PriceLevel:    row.PriceLevel,
			CoverImageURL: row.CoverImageUrl,
			Tags:          row.Tags,
		})
	}
	return detail, nil
}

func listRecentSyncRuns(ctx context.Context, db *pgxpool.Pool, limit int) ([]SyncRunSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := dbgen.New(db).ListRecentSyncRuns(ctx, int32(limit))
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list sync runs")
	}

	runs := make([]SyncRunSummary, 0, len(rows))
	for _, row := range rows {
		var details []SyncEntryResult
		if len(row.Details) > 0 {
			_ = json.Unmarshal(row.Details, &details)
		}
		runs = append(runs, SyncRunSummary{
			ID:             row.ID,
			StartedAt:      timestamptzToString(row.StartedAt),
			FinishedAt:     timestamptzToString(row.FinishedAt),
			Status:         row.Status,
			EntriesAdded:   row.EntriesAdded,
			EntriesSkipped: row.EntriesSkipped,
			EntriesFailed:  row.EntriesFailed,
			Details:        details,
		})
	}
	return runs, nil
}

func timestamptzToString(v pgtype.Timestamptz) string {
	if !v.Valid {
		return ""
	}
	return v.Time.UTC().Format(time.RFC3339)
}
