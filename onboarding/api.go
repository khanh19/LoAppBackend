package onboarding

import (
	"context"
	"strings"

	"encore.dev/beta/errs"
)

//encore:api public method=GET path=/onboarding/cities
func (s *Service) ListCities(ctx context.Context) (*ListCitiesResponse, error) {
	cities, err := listCities(ctx, s.db)
	if err != nil {
		return nil, err
	}
	return &ListCitiesResponse{Cities: cities}, nil
}

//encore:api public method=GET path=/onboarding/categories
func (s *Service) ListCategories(ctx context.Context) (*ListCategoriesResponse, error) {
	categories, err := listCategories(ctx, s.db)
	if err != nil {
		return nil, err
	}
	return &ListCategoriesResponse{Categories: categories}, nil
}

//encore:api public method=GET path=/onboarding/vibes
func (s *Service) ListVibes(ctx context.Context) (*ListVibesResponse, error) {
	vibes, err := listVibes(ctx, s.db)
	if err != nil {
		return nil, err
	}
	return &ListVibesResponse{Vibes: vibes}, nil
}

//encore:api public method=GET path=/onboarding/venues
func (s *Service) ListVenues(ctx context.Context, params *ListVenuesParams) (*ListVenuesResponse, error) {
	if params == nil || strings.TrimSpace(params.CityID) == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "city_id is required"}
	}

	venues, err := listVenuesByCity(ctx, s.db, strings.TrimSpace(params.CityID))
	if err != nil {
		return nil, err
	}
	return &ListVenuesResponse{Venues: venues}, nil
}
