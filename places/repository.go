package places

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	proRefreshAfter   = 7 * 24 * time.Hour
	photoRefreshAfter = 30 * 24 * time.Hour
)

func getPlaceByID(ctx context.Context, db *pgxpool.Pool, id string) (*PlaceDetail, error) {
	placeUUID, err := uuidFromString(id)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid place id"}
	}

	row, err := dbgen.New(db).GetPlaceByID(ctx, placeUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "place not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to load place")
	}
	return mapPlaceRow(row), nil
}

func mapPlaceRow(row dbgen.GetPlaceByIDRow) *PlaceDetail {
	detail := mapSharedPlaceRow(
		row.ID,
		row.GooglePlaceID,
		row.Name,
		row.Address,
		row.Neighborhood,
		row.Latitude,
		row.Longitude,
		row.RatingCached,
		row.UserRatingCount,
		row.PriceLevel,
		row.HoursJson,
		row.Phone,
		row.Website,
		row.BusinessStatus,
		row.CoverImageUrl,
		row.PhotoNames,
		row.Tags,
		row.CitySlug,
		row.CityName,
		row.LastSyncedAt,
	)
	detail.VenueCategory = row.VenueCategory
	detail.VenueArchetype = row.VenueArchetype
	return detail
}

func mapGooglePlaceRow(row dbgen.GetPlaceByGooglePlaceIDRow) *PlaceDetail {
	detail := mapSharedPlaceRow(
		row.ID,
		row.GooglePlaceID,
		row.Name,
		row.Address,
		row.Neighborhood,
		row.Latitude,
		row.Longitude,
		row.RatingCached,
		row.UserRatingCount,
		row.PriceLevel,
		row.HoursJson,
		row.Phone,
		row.Website,
		row.BusinessStatus,
		row.CoverImageUrl,
		row.PhotoNames,
		row.Tags,
		row.CitySlug,
		row.CityName,
		row.LastSyncedAt,
	)
	detail.VenueCategory = row.VenueCategory
	detail.VenueArchetype = row.VenueArchetype
	return detail
}

func mapSharedPlaceRow(
	id string,
	googlePlaceID *string,
	name string,
	address *string,
	neighborhood *string,
	latitude pgtype.Numeric,
	longitude pgtype.Numeric,
	ratingCached pgtype.Numeric,
	userRatingCount *int32,
	priceLevel *int16,
	hoursJSON []byte,
	phone *string,
	website *string,
	businessStatus *string,
	coverImageURL *string,
	photoNames []string,
	tags []string,
	citySlug string,
	cityName string,
	lastSyncedAt pgtype.Timestamptz,
) *PlaceDetail {
	lastSynced := timestamptzToTime(lastSyncedAt)
	isStale := isProStale(lastSynced)
	persistentCoverImageURL := coverImageURL
	if coverImageURL != nil && isGoogleUserContentURL(*coverImageURL) {
		persistentCoverImageURL = nil
	}

	detail := &PlaceDetail{
		ID:              id,
		GooglePlaceID:   googlePlaceID,
		Name:            name,
		Address:         address,
		Neighborhood:    neighborhood,
		Latitude:        numericToFloat(latitude),
		Longitude:       numericToFloat(longitude),
		Rating:          numericToFloat(ratingCached),
		UserRatingCount: userRatingCount,
		PriceLevel:      priceLevel,
		Hours:           formatHoursFromJSON(hoursJSON),
		Phone:           phone,
		Website:         website,
		BusinessStatus:  businessStatus,
		CoverImageURL:   persistentCoverImageURL,
		Tags:            tags,
		CitySlug:        citySlug,
		CityName:        cityName,
		LastSyncedAt:    lastSynced.Format(time.RFC3339),
		IsStale:         isStale,
	}
	if len(hoursJSON) > 0 {
		detail.HoursJSON = json.RawMessage(hoursJSON)
	}
	if persistentCoverImageURL != nil && *persistentCoverImageURL != "" {
		detail.PhotoURLs = []string{*persistentCoverImageURL}
	}
	_ = photoNames
	return detail
}

func formatHoursFromJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var hours struct {
		WeekdayDescriptions []string `json:"weekdayDescriptions"`
	}
	if err := json.Unmarshal(raw, &hours); err != nil {
		return ""
	}
	if len(hours.WeekdayDescriptions) == 0 {
		return ""
	}
	now := time.Now()
	idx := int(now.Weekday())
	if idx == 0 {
		idx = 6
	} else {
		idx--
	}
	if idx < len(hours.WeekdayDescriptions) {
		parts := strings.SplitN(hours.WeekdayDescriptions[idx], ": ", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		return hours.WeekdayDescriptions[idx]
	}
	return hours.WeekdayDescriptions[0]
}

func isProStale(lastSynced time.Time) bool {
	if lastSynced.IsZero() {
		return true
	}
	return time.Since(lastSynced) > proRefreshAfter
}

func needsPhotoRefresh(lastSynced time.Time, photoNames []string) bool {
	if len(photoNames) == 0 {
		return true
	}
	if lastSynced.IsZero() {
		return true
	}
	return time.Since(lastSynced) > photoRefreshAfter
}

func timestamptzToTime(v pgtype.Timestamptz) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return v.Time
}
