package places

import (
	"context"
	"math"
	"math/rand"
	"strings"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	nearbyMinRadiusM     = 500
	nearbyMaxRadiusM     = 15000
	nearbyDefaultRadiusM = 2000
	nearbyDeckLimit      = 20
	nearbyGoogleFillAt   = 8
	nearbyMaxCompanions  = 3
)

var nearbyCategories = map[string]string{
	"restaurant": "restaurant",
	"cafe":       "cafe",
	"bar":        "bar",
	"bars":       "bar",
	"club":       "club",
	"pub":        "pub",
}

//encore:api public method=POST path=/places/nearby
func (s *Service) NearbyPlaces(ctx context.Context, req *NearbyRequest) (*NearbyResponse, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	if !isValidLatLng(req.Latitude, req.Longitude) {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "latitude and longitude are required"}
	}

	radiusM := req.RadiusM
	if radiusM <= 0 {
		radiusM = nearbyDefaultRadiusM
	}
	if radiusM < nearbyMinRadiusM {
		radiusM = nearbyMinRadiusM
	}
	if radiusM > nearbyMaxRadiusM {
		radiusM = nearbyMaxRadiusM
	}

	category, err := normalizeNearbyCategory(req.Category)
	if err != nil {
		return nil, err
	}

	excludeIDs, err := parseExcludeIDs(req.ExcludeIDs)
	if err != nil {
		return nil, err
	}

	rows, err := s.listNearbyFromDB(ctx, req.Latitude, req.Longitude, float64(radiusM), category, excludeIDs, nearbyDeckLimit)
	if err != nil {
		return nil, err
	}

	if len(rows) < nearbyGoogleFillAt {
		if err := s.fillNearbyFromGoogle(ctx, req.Latitude, req.Longitude, float64(radiusM), category); err != nil {
			// Soft-fail Google fill: still return whatever we already have in DB.
			if len(rows) == 0 {
				return nil, errs.WrapCode(err, errs.Internal, "nearby search failed")
			}
		} else {
			rows, err = s.listNearbyFromDB(ctx, req.Latitude, req.Longitude, float64(radiusM), category, excludeIDs, nearbyDeckLimit)
			if err != nil {
				return nil, err
			}
		}
	}

	places, err := s.enrichNearbyPlaces(ctx, rows)
	if err != nil {
		return nil, err
	}
	rand.Shuffle(len(places), func(i, j int) {
		places[i], places[j] = places[j], places[i]
	})
	return &NearbyResponse{Places: places}, nil
}

func normalizeNearbyCategory(raw string) (*string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return nil, nil
	}
	mapped, ok := nearbyCategories[trimmed]
	if !ok {
		return nil, &errs.Error{
			Code:    errs.InvalidArgument,
			Message: "category must be restaurant, cafe, bar, club, or pub",
		}
	}
	return &mapped, nil
}

func parseExcludeIDs(raw []string) ([]pgtype.UUID, error) {
	if len(raw) == 0 {
		return []pgtype.UUID{}, nil
	}
	out := make([]pgtype.UUID, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, id := range raw {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		uid, err := uuidFromString(id)
		if err != nil {
			return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid exclude id"}
		}
		seen[id] = struct{}{}
		out = append(out, uid)
	}
	return out, nil
}

func isValidLatLng(lat, lng float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return false
	}
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && !(lat == 0 && lng == 0)
}

func (s *Service) listNearbyFromDB(
	ctx context.Context,
	lat, lng, radiusM float64,
	category *string,
	excludeIDs []pgtype.UUID,
	limit int32,
) ([]dbgen.ListNearbyPlacesRow, error) {
	// ~1 degree latitude ≈ 111 km. Expand the box a bit so edges are not clipped.
	delta := (radiusM / 111000.0) * 1.2
	lngDelta := delta / math.Max(0.2, math.Cos(lat*math.Pi/180))

	rows, err := dbgen.New(s.db).ListNearbyPlaces(ctx, dbgen.ListNearbyPlacesParams{
		SwLat:      lat - delta,
		NeLat:      lat + delta,
		SwLng:      lng - lngDelta,
		NeLng:      lng + lngDelta,
		Category:   category,
		ExcludeIds: excludeIDs,
		Lat:        lat,
		Lng:        lng,
		RadiusM:    radiusM,
		LimitCount: limit,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list nearby places")
	}
	return rows, nil
}

// nearbyGoogleTypes maps a swipe category onto Google Places included types.
// Pubs are stored as bars, so the pub search asks Google for bars and the SQL
// filter keeps places tagged as pubs.
func nearbyGoogleTypes(category *string) []string {
	if category == nil {
		return []string{"restaurant", "cafe", "bar", "night_club"}
	}
	switch *category {
	case "club":
		return []string{"night_club"}
	case "pub":
		return []string{"bar"}
	default:
		return []string{*category}
	}
}

func (s *Service) fillNearbyFromGoogle(ctx context.Context, lat, lng, radiusM float64, category *string) error {
	types := nearbyGoogleTypes(category)

	resolved, err := s.google.searchNearby(ctx, lat, lng, radiusM, types, nearbyDeckLimit)
	if err != nil {
		return err
	}

	q := dbgen.New(s.db)
	for _, place := range resolved {
		if place == nil || place.GooglePlaceID == "" {
			continue
		}
		// Skip places Google returned that fall outside a Lo city.
		city, err := resolveCity(ctx, q, "", place.Latitude, place.Longitude, place.CityHint)
		if err != nil {
			continue
		}
		cityUUID, err := uuidFromString(city.ID)
		if err != nil {
			continue
		}
		_, _ = q.UpsertPlaceFromGoogle(ctx, upsertParamsFromResolved(cityUUID, place))
	}
	return nil
}

func (s *Service) enrichNearbyPlaces(ctx context.Context, rows []dbgen.ListNearbyPlacesRow) ([]NearbyPlace, error) {
	if len(rows) == 0 {
		return []NearbyPlace{}, nil
	}

	placeIDs := make([]pgtype.UUID, 0, len(rows))
	for _, row := range rows {
		uid, err := uuidFromString(row.ID)
		if err != nil {
			continue
		}
		placeIDs = append(placeIDs, uid)
	}

	q := dbgen.New(s.db)
	bookmarkCounts := map[string]int32{}
	if len(placeIDs) > 0 {
		counts, err := q.CountBookmarksForPlaces(ctx, placeIDs)
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to count bookmarks")
		}
		for _, row := range counts {
			bookmarkCounts[row.PlaceID] = row.BookmarkCount
		}
	}

	companions := map[string][]string{}
	if len(placeIDs) > 0 {
		tags, err := q.ListTopCompanionsForPlaces(ctx, placeIDs)
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to load companions")
		}
		for _, tag := range tags {
			list := companions[tag.PlaceID]
			if len(list) >= nearbyMaxCompanions {
				continue
			}
			label := strings.TrimSpace(tag.Label)
			if label == "" {
				continue
			}
			companions[tag.PlaceID] = append(list, label)
		}
	}

	out := make([]NearbyPlace, 0, len(rows))
	for _, row := range rows {
		cover := row.CoverImageUrl
		if cover != nil && isGoogleUserContentURL(*cover) {
			cover = nil
		}
		labels := companions[row.ID]
		if labels == nil {
			labels = []string{}
		}
		out = append(out, NearbyPlace{
			ID:            row.ID,
			Name:          row.Name,
			Address:       row.Address,
			Ward:          row.Ward,
			Latitude:      numericToFloat(row.Latitude),
			Longitude:     numericToFloat(row.Longitude),
			PriceLevel:    row.PriceLevel,
			CoverImageURL: cover,
			VenueCategory: row.VenueCategory,
			CitySlug:      row.CitySlug,
			CityName:      row.CityName,
			BookmarkCount: bookmarkCounts[row.ID],
			BestForLabels: labels,
		})
	}
	return out, nil
}
