package lists

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) createPlan(ctx context.Context, userID string, req *CreatePlanRequest) (*PlaceListDetail, error) {
	normalized, err := normalizeCreatePlanRequest(req)
	if err != nil {
		return nil, err
	}

	creatorID, err := uuidFromString(userID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.InvalidArgument, "invalid user id")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to begin plan create transaction")
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := dbgen.New(tx)
	city, err := q.GetCityByHint(ctx, dbgen.GetCityByHintParams{
		CityHint:    normalized.CitySlug,
		CityHintAlt: cityHintAlt(normalized.CitySlug),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &errs.Error{Code: errs.InvalidArgument, Message: "city_slug does not match an active city"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to resolve plan city")
	}
	cityID, err := uuidFromString(city.ID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "invalid city id")
	}

	listID, err := q.CreatePlan(ctx, dbgen.CreatePlanParams{
		Slug:          normalized.Slug,
		Title:         normalized.Title,
		Subtitle:      normalized.Subtitle,
		Category:      normalized.Category,
		Area:          normalized.Area,
		Occasions:     stringSliceOrEmpty(normalized.Occasions),
		CityID:        cityID,
		CreatorUserID: creatorID,
		Visibility:    normalized.Visibility,
		SavesCount:    normalized.SavesCount,
		SortOrder:     0,
	})
	if err != nil {
		if isUniqueViolation(err, "place_lists_slug_key") {
			return nil, &errs.Error{Code: errs.AlreadyExists, Message: "plan slug already exists"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to create plan")
	}
	listUUID, err := uuidFromString(listID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "invalid plan id")
	}

	for _, stop := range normalized.Stops {
		placeID, err := s.createOrUsePlanStopPlace(ctx, q, cityID, stop)
		if err != nil {
			return nil, err
		}
		placeUUID, err := uuidFromString(placeID)
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "invalid stop place id")
		}
		if err := q.CreatePlanEntry(ctx, dbgen.CreatePlanEntryParams{
			ListID:       listUUID,
			PlaceID:      placeUUID,
			SeedName:     stop.PlaceName,
			StopOrder:    stop.StopOrder,
			Note:         stop.Note,
			TimeLabel:    stop.TimeLabel,
			ActivityType: stop.ActivityType,
			ImageUrl:     stop.ImageURL,
		}); err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to create plan stop")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to commit plan")
	}

	return getPlaceListBySlug(ctx, s.db, normalized.Slug)
}

func (s *Service) createOrUsePlanStopPlace(ctx context.Context, q *dbgen.Queries, cityID pgtype.UUID, stop CreatePlanStopRequest) (string, error) {
	if stop.PlaceID != nil {
		placeID := strings.TrimSpace(*stop.PlaceID)
		if _, err := uuidFromString(placeID); err != nil {
			return "", &errs.Error{Code: errs.InvalidArgument, Message: "place_id must be a valid UUID"}
		}
		return placeID, nil
	}

	placeID, err := q.CreatePlanPlace(ctx, dbgen.CreatePlanPlaceParams{
		CityID:        cityID,
		GooglePlaceID: stop.GooglePlaceID,
		Name:          stop.PlaceName,
		Neighborhood:  stop.Neighborhood,
		Address:       stop.Address,
		Latitude:      numericFromOptionalFloat(stop.Latitude),
		Longitude:     numericFromOptionalFloat(stop.Longitude),
		CoverImageUrl: stop.ImageURL,
		// places.tags is NOT NULL; nil []string encodes as SQL NULL.
		Tags: stringSliceOrEmpty(stop.Tags),
	})
	if err != nil {
		return "", errs.WrapCode(err, errs.Internal, "failed to create plan place")
	}
	return placeID, nil
}

func normalizeCreatePlanRequest(req *CreatePlanRequest) (*CreatePlanRequest, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	normalized := *req
	normalized.Title = strings.TrimSpace(normalized.Title)
	if normalized.Title == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "title is required"}
	}
	normalized.Subtitle = strings.TrimSpace(normalized.Subtitle)
	normalized.CitySlug = strings.TrimSpace(normalized.CitySlug)
	if normalized.CitySlug == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "city_slug is required"}
	}
	normalized.Category = strings.TrimSpace(normalized.Category)
	normalized.Area = strings.TrimSpace(normalized.Area)
	normalized.Visibility = strings.TrimSpace(normalized.Visibility)
	if normalized.Visibility == "" {
		normalized.Visibility = "public"
	}
	if normalized.Visibility != "public" && normalized.Visibility != "private" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "visibility must be public or private"}
	}
	if normalized.SavesCount < 0 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "saves_count cannot be negative"}
	}

	normalized.Slug = normalizePlanSlug(normalized.Slug)
	if normalized.Slug == "" {
		normalized.Slug = slugFromTitle(normalized.Title)
	}
	if normalized.Slug == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "slug is required"}
	}
	normalized.Occasions = normalizeStringSlice(normalized.Occasions)

	if len(normalized.Stops) == 0 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "at least one stop is required"}
	}
	normalized.Stops = append([]CreatePlanStopRequest(nil), normalized.Stops...)
	seenStopOrders := make(map[int32]struct{}, len(normalized.Stops))
	seenPlaceNames := make(map[string]struct{}, len(normalized.Stops))
	for i := range normalized.Stops {
		stop, err := normalizeCreatePlanStop(normalized.Stops[i], i+1)
		if err != nil {
			return nil, err
		}
		if _, ok := seenStopOrders[stop.StopOrder]; ok {
			return nil, &errs.Error{Code: errs.InvalidArgument, Message: "stop_order values must be unique"}
		}
		seenStopOrders[stop.StopOrder] = struct{}{}
		placeNameKey := strings.ToLower(stop.PlaceName)
		if _, ok := seenPlaceNames[placeNameKey]; ok {
			return nil, &errs.Error{Code: errs.InvalidArgument, Message: "stop place_name values must be unique"}
		}
		seenPlaceNames[placeNameKey] = struct{}{}
		normalized.Stops[i] = stop
	}

	return &normalized, nil
}

func normalizeCreatePlanStop(stop CreatePlanStopRequest, fallbackStopOrder int) (CreatePlanStopRequest, error) {
	stop.PlaceName = strings.TrimSpace(stop.PlaceName)
	if stop.PlaceName == "" {
		return stop, &errs.Error{Code: errs.InvalidArgument, Message: "stop place_name is required"}
	}
	if stop.StopOrder <= 0 {
		stop.StopOrder = int32(fallbackStopOrder)
	}
	stop.TimeLabel = strings.TrimSpace(stop.TimeLabel)
	stop.ActivityType = strings.TrimSpace(stop.ActivityType)
	stop.Note = strings.TrimSpace(stop.Note)
	stop.PlaceID = trimStringPtr(stop.PlaceID)
	stop.GooglePlaceID = trimStringPtr(stop.GooglePlaceID)
	stop.Address = trimStringPtr(stop.Address)
	stop.Neighborhood = trimStringPtr(stop.Neighborhood)
	stop.ImageURL = trimStringPtr(stop.ImageURL)
	stop.Tags = normalizeStringSlice(stop.Tags)

	if stop.Latitude != nil && (*stop.Latitude < -90 || *stop.Latitude > 90) {
		return stop, &errs.Error{Code: errs.InvalidArgument, Message: "latitude must be between -90 and 90"}
	}
	if stop.Longitude != nil && (*stop.Longitude < -180 || *stop.Longitude > 180) {
		return stop, &errs.Error{Code: errs.InvalidArgument, Message: "longitude must be between -180 and 180"}
	}
	return stop, nil
}

func normalizePlanSlug(raw string) string {
	return collapseSlug(strings.TrimSpace(strings.ToLower(raw)))
}

func slugFromTitle(title string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '/':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func collapseSlug(slug string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || unicode.IsSpace(r) || r == '_' || r == '/':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func stringSliceOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func isUniqueViolation(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraintName
}
