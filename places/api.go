package places

import (
	"context"
	"errors"
	"strings"

	"encore.app/internal/dbgen"
	"encore.app/stamps"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
)

//encore:api public method=GET path=/places/:id
func (s *Service) GetPlace(ctx context.Context, id string) (*GetPlaceResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "place id is required"}
	}

	place, err := getPlaceByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}

	if place.IsStale {
		s.maybeRefreshPlace(ctx, id)
	}

	if summary, err := stamps.GetStampSummary(ctx, &stamps.GetStampSummaryRequest{PlaceID: id}); err == nil && summary != nil {
		place.StampSummary = mapStampSummary(summary)
	}

	return &GetPlaceResponse{Place: *place}, nil
}

func mapStampSummary(summary *stamps.StampSummary) *StampSummary {
	if summary == nil {
		return nil
	}
	out := &StampSummary{
		StampCount:      summary.StampCount,
		FinalScore:      summary.FinalScore,
		DisplayState:    summary.DisplayState,
		SentimentScore:  summary.SentimentScore,
		QualityScore:    summary.QualityScore,
		VibeConflict:    summary.VibeConflict,
		QualityConflict: summary.QualityConflict,
		ConflictNote:    summary.ConflictNote,
		Companions:      make([]TagAggregate, 0, len(summary.Companions)),
		Vibes:           make([]TagAggregate, 0, len(summary.Vibes)),
	}
	for _, c := range summary.Companions {
		out.Companions = append(out.Companions, TagAggregate{
			Slug: c.Slug, Percentage: c.Percentage, RawCount: c.RawCount,
		})
	}
	for _, v := range summary.Vibes {
		out.Vibes = append(out.Vibes, TagAggregate{
			Slug: v.Slug, Percentage: v.Percentage, RawCount: v.RawCount,
		})
	}
	return out
}

//encore:api public method=GET path=/places/:id/photo
func (s *Service) GetPlacePhoto(ctx context.Context, id string, params *GetPlacePhotoParams) (*GetPlacePhotoResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "place id is required"}
	}

	index := 0
	maxWidth := 800
	if params != nil {
		if params.Index > 0 {
			index = params.Index
		}
		if params.W > 0 {
			maxWidth = params.W
		}
	}
	if maxWidth > 1600 {
		maxWidth = 1600
	}

	placeUUID, err := uuidFromString(id)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid place id"}
	}

	row, err := dbgen.New(s.db).GetPlaceByID(ctx, placeUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "place not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to load place")
	}

	photoURL, err := s.resolvePlacePhotoURL(ctx, id, row, index, maxWidth)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to fetch place photo")
	}
	if photoURL == "" {
		return nil, &errs.Error{Code: errs.NotFound, Message: "no photo available for place"}
	}
	return &GetPlacePhotoResponse{PhotoURL: photoURL}, nil
}

//encore:api public method=POST path=/places/autocomplete
func (s *Service) AutocompletePlaces(ctx context.Context, req *AutocompleteRequest) (*AutocompleteResponse, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	input := strings.TrimSpace(req.Input)
	if len(input) < 2 {
		return &AutocompleteResponse{Predictions: []AutocompletePrediction{}}, nil
	}
	sessionToken := strings.TrimSpace(req.SessionToken)
	if sessionToken == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "session_token is required"}
	}

	predictions, err := s.google.autocomplete(ctx, input, sessionToken, req.Latitude, req.Longitude)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "autocomplete failed")
	}
	return &AutocompleteResponse{Predictions: predictions}, nil
}

//encore:api public method=POST path=/places/resolve
func (s *Service) ResolvePlace(ctx context.Context, req *ResolvePlaceRequest) (*ResolvePlaceResponse, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	googlePlaceID := strings.TrimSpace(req.GooglePlaceID)
	if googlePlaceID == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "google_place_id is required"}
	}

	placeID, err := s.resolveAndUpsertPlace(
		ctx,
		s.db,
		googlePlaceID,
		strings.TrimSpace(req.SessionToken),
		req.CitySlug,
		req.Latitude,
		req.Longitude,
	)
	if err != nil {
		return nil, err
	}

	place, err := getPlaceByID(ctx, s.db, placeID)
	if err != nil {
		return nil, err
	}
	return &ResolvePlaceResponse{PlaceID: placeID, Place: *place}, nil
}

//encore:api private
func (s *Service) ResolvePlaceForPlan(ctx context.Context, req *ResolvePlaceForPlanRequest) (*ResolvePlaceForPlanResponse, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	googlePlaceID := strings.TrimSpace(req.GooglePlaceID)
	if googlePlaceID == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "google_place_id is required"}
	}
	placeID, err := s.resolveAndUpsertPlace(ctx, s.db, googlePlaceID, "", req.CitySlug, 0, 0)
	if err != nil {
		return nil, err
	}
	return &ResolvePlaceForPlanResponse{PlaceID: placeID}, nil
}
