package places

import (
	"context"
	"errors"
	"strings"
	"time"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) maybeRefreshPlace(ctx context.Context, placeID string) {
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.refreshPlaceIfStale(bgCtx, placeID)
	}()
}

func (s *Service) refreshPlaceIfStale(ctx context.Context, placeID string) error {
	placeUUID, err := uuidFromString(placeID)
	if err != nil {
		return err
	}

	q := dbgen.New(s.db)
	row, err := q.GetPlaceByID(ctx, placeUUID)
	if err != nil {
		return err
	}
	if row.GooglePlaceID == nil || strings.TrimSpace(*row.GooglePlaceID) == "" {
		return nil
	}

	lastSynced := timestamptzToTime(row.LastSyncedAt)
	if !isProStale(lastSynced) && !needsPhotoRefresh(lastSynced, row.PhotoNames) {
		return nil
	}

	includePhotos := needsPhotoRefresh(lastSynced, row.PhotoNames)
	resolved, err := s.google.getPlaceDetails(ctx, *row.GooglePlaceID, "", includePhotos)
	if err != nil {
		return err
	}

	cityUUID, err := uuidFromString(row.CityID)
	if err != nil {
		return err
	}

	_, err = q.UpsertPlaceFromGoogle(ctx, upsertParamsFromResolved(cityUUID, resolved))
	return err
}

func (s *Service) resolveAndUpsertPlace(
	ctx context.Context,
	db dbgen.DBTX,
	googlePlaceID string,
	sessionToken string,
	citySlug string,
	lat float64,
	lng float64,
) (string, error) {
	q := dbgen.New(db)
	existing, err := q.GetPlaceByGooglePlaceID(ctx, stringPtr(googlePlaceID))
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", errs.WrapCode(err, errs.Internal, "failed to lookup place")
	}

	includePhotos := true
	resolved, err := s.google.getPlaceDetails(ctx, googlePlaceID, sessionToken, includePhotos)
	if err != nil {
		return "", errs.WrapCode(err, errs.Internal, "failed to resolve google place")
	}

	city, err := resolveCity(ctx, q, citySlug, lat, lng)
	if err != nil {
		return "", err
	}
	cityUUID, err := uuidFromString(city.ID)
	if err != nil {
		return "", errs.WrapCode(err, errs.Internal, "invalid city id")
	}

	placeID, err := q.UpsertPlaceFromGoogle(ctx, upsertParamsFromResolved(cityUUID, resolved))
	if err != nil {
		return "", errs.WrapCode(err, errs.Internal, "failed to upsert place")
	}
	return placeID, nil
}

func upsertParamsFromResolved(cityID pgtype.UUID, resolved *googleResolvedPlace) dbgen.UpsertPlaceFromGoogleParams {
	params := dbgen.UpsertPlaceFromGoogleParams{
		CityID:          cityID,
		GooglePlaceID:   stringPtr(resolved.GooglePlaceID),
		Name:            resolved.Name,
		Neighborhood:  stringPtr(resolved.Neighborhood),
		Address:       stringPtr(resolved.Address),
		Latitude:      numericFromFloat(resolved.Latitude),
		Longitude:     numericFromFloat(resolved.Longitude),
		PriceLevel:    resolved.PriceLevel,
		RatingCached:  numericFromOptionalFloat(resolved.Rating),
		UserRatingCount: resolved.UserRatingCount,
		Phone:         stringPtr(resolved.Phone),
		Website:       stringPtr(resolved.Website),
		BusinessStatus: stringPtr(resolved.BusinessStatus),
		CoverImageUrl: stringPtr(resolved.CoverImageURL),
		Tags:          resolved.Tags,
	}
	if len(resolved.HoursJSON) > 0 {
		params.HoursJson = resolved.HoursJSON
	}
	if len(resolved.PhotoNames) > 0 {
		params.PhotoNames = resolved.PhotoNames
	}
	if len(resolved.Tags) > 0 {
		params.Tags = resolved.Tags
	} else {
		params.Tags = []string{}
	}
	return params
}

func resolveCity(ctx context.Context, q *dbgen.Queries, citySlug string, lat, lng float64) (dbgen.GetCityByHintRow, error) {
	citySlug = strings.TrimSpace(citySlug)
	if citySlug != "" {
		city, err := q.GetCityByHint(ctx, dbgen.GetCityByHintParams{
			CityHint:    citySlug,
			CityHintAlt: cityHintAlt(citySlug),
		})
		if err == nil {
			return city, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return dbgen.GetCityByHintRow{}, errs.WrapCode(err, errs.Internal, "failed to resolve city")
		}
	}

	// Default to HCMC when city slug is missing or unknown.
	city, err := q.GetCityByHint(ctx, dbgen.GetCityByHintParams{
		CityHint:    "hcmc",
		CityHintAlt: "ho chi minh city",
	})
	if err != nil {
		return dbgen.GetCityByHintRow{}, &errs.Error{Code: errs.InvalidArgument, Message: "city_slug does not match an active city"}
	}
	_ = lat
	_ = lng
	return city, nil
}
