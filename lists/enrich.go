package lists

import (
	"context"
	"strings"
	"sync"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
)

const enrichPlanPlaceWorkers = 3

type EnrichPlanPlacesParams struct {
	Limit int `query:"limit"`
}

type EnrichPlanPlaceResult struct {
	PlaceID string `json:"place_id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

type EnrichPlanPlacesResponse struct {
	Enriched int                     `json:"enriched"`
	Failed   int                     `json:"failed"`
	Skipped  int                     `json:"skipped"`
	Results  []EnrichPlanPlaceResult `json:"results"`
}

func (s *Service) enrichPlanPlacesMissingCover(ctx context.Context, limit int) (*EnrichPlanPlacesResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	q := dbgen.New(s.db)
	rows, err := q.ListPlanPlacesMissingCover(ctx, int32(limit))
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list plan places missing cover")
	}
	if len(rows) == 0 {
		return &EnrichPlanPlacesResponse{Results: []EnrichPlanPlaceResult{}}, nil
	}

	type job struct {
		row dbgen.ListPlanPlacesMissingCoverRow
	}
	type result struct {
		item EnrichPlanPlaceResult
	}

	jobs := make(chan job)
	results := make(chan result)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			item := EnrichPlanPlaceResult{
				PlaceID: j.row.ID,
				Name:    j.row.Name,
			}
			if err := s.enrichOnePlanPlace(ctx, q, j.row); err != nil {
				item.Status = "failed"
				item.Error = err.Error()
			} else {
				item.Status = "enriched"
			}
			results <- result{item: item}
		}
	}

	for i := 0; i < enrichPlanPlaceWorkers; i++ {
		wg.Add(1)
		go worker()
	}
	go func() {
		for _, row := range rows {
			jobs <- job{row: row}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	resp := &EnrichPlanPlacesResponse{
		Results: make([]EnrichPlanPlaceResult, 0, len(rows)),
	}
	for r := range results {
		resp.Results = append(resp.Results, r.item)
		switch r.item.Status {
		case "enriched":
			resp.Enriched++
		case "failed":
			resp.Failed++
		default:
			resp.Skipped++
		}
	}
	return resp, nil
}

func (s *Service) enrichOnePlanPlace(ctx context.Context, q *dbgen.Queries, row dbgen.ListPlanPlacesMissingCoverRow) error {
	lat := numericToFloat(row.Latitude)
	lng := numericToFloat(row.Longitude)
	if lat == 0 && lng == 0 {
		lat, lng = cityCoords(dbgen.GetCityByHintRow{Slug: row.CitySlug})
	}

	query := strings.TrimSpace(row.Name)
	if row.CitySlug != "" {
		query = query + " " + row.CitySlug
	}

	resolved, err := s.google.resolvePlace(ctx, query, lat, lng)
	if err != nil {
		return err
	}

	placeUUID, err := uuidFromString(row.ID)
	if err != nil {
		return err
	}

	photoNames := []string{}
	if resolved.PhotoName != "" {
		photoNames = []string{resolved.PhotoName}
	}

	err = q.UpdatePlaceFromGoogleResolve(ctx, dbgen.UpdatePlaceFromGoogleResolveParams{
		GooglePlaceID: stringPtr(resolved.GooglePlaceID),
		Name:          resolved.Name,
		Neighborhood:  stringPtr(resolved.Neighborhood),
		Address:       stringPtr(resolved.Address),
		Latitude:      numericFromFloat(resolved.Latitude),
		Longitude:     numericFromFloat(resolved.Longitude),
		PriceLevel:    resolved.PriceLevel,
		RatingCached:  numericFromOptionalFloat(resolved.Rating),
		CoverImageUrl: stringPtr(resolved.CoverImageURL),
		PhotoNames:    photoNames,
		Tags:          resolved.Tags,
		ID:            placeUUID,
	})
	if err != nil {
		// Another place already owns this google_place_id — upsert that row instead.
		if isUniqueViolation(err, "places_google_place_id_key") {
			cityUUID, parseErr := uuidFromString(row.CityID)
			if parseErr != nil {
				return err
			}
			_, upsertErr := q.UpsertGooglePlace(ctx, dbgen.UpsertGooglePlaceParams{
				CityID:        cityUUID,
				GooglePlaceID: stringPtr(resolved.GooglePlaceID),
				Name:          resolved.Name,
				Neighborhood:  stringPtr(resolved.Neighborhood),
				Address:       stringPtr(resolved.Address),
				Latitude:      numericFromFloat(resolved.Latitude),
				Longitude:     numericFromFloat(resolved.Longitude),
				PriceLevel:    resolved.PriceLevel,
				RatingCached:  numericFromOptionalFloat(resolved.Rating),
				CoverImageUrl: stringPtr(resolved.CoverImageURL),
				PhotoNames:    photoNames,
				Tags:          resolved.Tags,
			})
			return upsertErr
		}
		return err
	}
	return nil
}
