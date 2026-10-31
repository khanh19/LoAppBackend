package places

import (
	"context"
	"errors"
	"strings"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
)

// cityConfig is the geography Lo keeps per supported city: the center used to
// bias Google searches, the bounding box used to scope them, and the radius.
type cityConfig struct {
	ID            string
	Slug          string
	Name          string
	Latitude      float64
	Longitude     float64
	SwLat         float64
	SwLng         float64
	NeLat         float64
	NeLng         float64
	SearchRadiusM int32
}

func (c cityConfig) hasBounds() bool {
	return c.SwLat < c.NeLat && c.SwLng < c.NeLng
}

func (c cityConfig) contains(lat, lng float64) bool {
	if !c.hasBounds() {
		return false
	}
	return lat >= c.SwLat && lat <= c.NeLat && lng >= c.SwLng && lng <= c.NeLng
}

// searchArea builds the Google search scope for this city. The user's own
// position is used as the bias center when it falls inside the city.
func (c cityConfig) searchArea(lat, lng float64) searchArea {
	return c.searchAreaWithRadius(lat, lng, 0)
}

// searchAreaWithRadius is like searchArea, but a positive radiusM replaces the
// city default and drops the bounding-box restriction so the circle radius is
// what Google actually uses (Google rejects bias + restriction together).
func (c cityConfig) searchAreaWithRadius(lat, lng, radiusM float64) searchArea {
	area := searchArea{
		Center:  center{Latitude: c.Latitude, Longitude: c.Longitude},
		RadiusM: float64(c.SearchRadiusM),
	}
	if c.contains(lat, lng) {
		area.Center = center{Latitude: lat, Longitude: lng}
	}
	if radiusM > 0 {
		area.RadiusM = radiusM
		return area
	}
	if c.hasBounds() {
		area.Restrict = &rectangle{
			Low:  center{Latitude: c.SwLat, Longitude: c.SwLng},
			High: center{Latitude: c.NeLat, Longitude: c.NeLng},
		}
	}
	return area
}

func cityFromHintRow(row dbgen.GetCityByHintRow) cityConfig {
	return cityConfig{
		ID:            row.ID,
		Slug:          row.Slug,
		Name:          row.Name,
		Latitude:      numericToFloat(row.Latitude),
		Longitude:     numericToFloat(row.Longitude),
		SwLat:         numericToFloat(row.SwLat),
		SwLng:         numericToFloat(row.SwLng),
		NeLat:         numericToFloat(row.NeLat),
		NeLng:         numericToFloat(row.NeLng),
		SearchRadiusM: row.SearchRadiusM,
	}
}

func cityFromPointRow(row dbgen.GetCityByPointRow) cityConfig {
	return cityConfig{
		ID:            row.ID,
		Slug:          row.Slug,
		Name:          row.Name,
		Latitude:      numericToFloat(row.Latitude),
		Longitude:     numericToFloat(row.Longitude),
		SwLat:         numericToFloat(row.SwLat),
		SwLng:         numericToFloat(row.SwLng),
		NeLat:         numericToFloat(row.NeLat),
		NeLng:         numericToFloat(row.NeLng),
		SearchRadiusM: row.SearchRadiusM,
	}
}

func cityByHint(ctx context.Context, q *dbgen.Queries, hint string) (cityConfig, bool, error) {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return cityConfig{}, false, nil
	}
	row, err := q.GetCityByHint(ctx, dbgen.GetCityByHintParams{
		CityHint:    hint,
		CityHintAlt: cityHintAlt(hint),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cityConfig{}, false, nil
		}
		return cityConfig{}, false, errs.WrapCode(err, errs.Internal, "failed to resolve city")
	}
	return cityFromHintRow(row), true, nil
}

func cityByPoint(ctx context.Context, q *dbgen.Queries, lat, lng float64) (cityConfig, bool, error) {
	if lat == 0 && lng == 0 {
		return cityConfig{}, false, nil
	}
	row, err := q.GetCityByPoint(ctx, dbgen.GetCityByPointParams{
		Lat: numericFromFloat(lat),
		Lng: numericFromFloat(lng),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cityConfig{}, false, nil
		}
		return cityConfig{}, false, errs.WrapCode(err, errs.Internal, "failed to resolve city by coordinates")
	}
	return cityFromPointRow(row), true, nil
}

// Google Autocomplete accepts at most five includedPrimaryTypes.
const maxIncludedPrimaryTypes = 5

func clampPrimaryTypes(types []string) []string {
	out := make([]string, 0, maxIncludedPrimaryTypes)
	seen := make(map[string]struct{}, maxIncludedPrimaryTypes)
	for _, raw := range types {
		t := strings.TrimSpace(raw)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) >= maxIncludedPrimaryTypes {
			break
		}
	}
	return out
}

// resolveCity picks the city in order of trustworthiness: the slug the client
// asked for, then the bounds containing the coordinates, then the locality
// Google reported for the place. There is no default city.
func resolveCity(ctx context.Context, q *dbgen.Queries, citySlug string, lat, lng float64, googleCityHint string) (cityConfig, error) {
	if city, ok, err := cityByHint(ctx, q, citySlug); err != nil || ok {
		return city, err
	}
	if city, ok, err := cityByPoint(ctx, q, lat, lng); err != nil || ok {
		return city, err
	}
	if city, ok, err := cityByHint(ctx, q, googleCityHint); err != nil || ok {
		return city, err
	}
	return cityConfig{}, &errs.Error{
		Code:    errs.InvalidArgument,
		Message: "could not resolve an active city from city_slug or coordinates",
	}
}
