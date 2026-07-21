package places

import (
	"context"
	"errors"
	"strings"

	"encore.app/internal/dbgen"
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

	return &GetPlaceResponse{Place: *place}, nil
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

	if len(row.PhotoNames) > index && row.PhotoNames[index] != "" {
		photoURL, err := s.google.fetchPhotoURL(ctx, row.PhotoNames[index], maxWidth)
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to fetch place photo")
		}
		return &GetPlacePhotoResponse{PhotoURL: photoURL}, nil
	}

	if row.CoverImageUrl != nil && *row.CoverImageUrl != "" {
		return &GetPlacePhotoResponse{PhotoURL: *row.CoverImageUrl}, nil
	}

	return nil, &errs.Error{Code: errs.NotFound, Message: "no photo available for place"}
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
