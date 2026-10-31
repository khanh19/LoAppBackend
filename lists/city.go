package lists

import (
	"context"
	"errors"
	"strings"

	"encore.app/internal/dbgen"
	"github.com/jackc/pgx/v5"
)

// cityAreaForSlug reads the Google search scope for a city slug. An unknown
// slug yields an empty area, which leaves the query unbiased rather than
// silently pointing it at another city.
func cityAreaForSlug(ctx context.Context, q *dbgen.Queries, citySlug string) (searchArea, error) {
	citySlug = strings.TrimSpace(citySlug)
	if citySlug == "" {
		return searchArea{}, nil
	}
	city, err := q.GetCityByHint(ctx, dbgen.GetCityByHintParams{
		CityHint:    citySlug,
		CityHintAlt: cityHintAlt(citySlug),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return searchArea{}, nil
		}
		return searchArea{}, err
	}
	return citySearchArea(city), nil
}
