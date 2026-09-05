package places

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"encore.app/internal/dbgen"
	"encore.dev"
)

// GetPlacePhotoImage exposes a stable app URL for place images. It resolves a
// fresh Google photo URI per request and redirects without exposing the API key.
//
//encore:api public raw method=GET path=/places/:id/photo/image
func (s *Service) GetPlacePhotoImage(w http.ResponseWriter, req *http.Request) {
	current := encore.CurrentRequest()
	if current == nil {
		http.Error(w, "request metadata unavailable", http.StatusInternalServerError)
		return
	}

	id := strings.TrimSpace(current.PathParams.Get("id"))
	placeUUID, err := uuidFromString(id)
	if err != nil {
		http.Error(w, "invalid place id", http.StatusBadRequest)
		return
	}

	index := queryInt(req, "index", 0)
	maxWidth := queryInt(req, "w", 800)
	if index < 0 {
		index = 0
	}
	if maxWidth <= 0 {
		maxWidth = 800
	}
	if maxWidth > 1600 {
		maxWidth = 1600
	}

	row, err := dbgen.New(s.db).GetPlaceByID(req.Context(), placeUUID)
	if err != nil {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}
	photoURL, err := s.resolvePlacePhotoURL(req.Context(), id, row, index, maxWidth)
	if err != nil {
		http.Error(w, "failed to resolve place photo", http.StatusBadGateway)
		return
	}
	if photoURL == "" {
		http.Error(w, "no photo available", http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, req, photoURL, http.StatusFound)
}

func queryInt(req *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(req.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (s *Service) resolvePlacePhotoURL(
	ctx context.Context,
	placeID string,
	row dbgen.GetPlaceByIDRow,
	index, maxWidth int,
) (string, error) {
	if url, err := s.photoURLFromNames(ctx, row.PhotoNames, index, maxWidth); err != nil {
		// Google photo names can expire. Refresh Place Details below and retry
		// with the newly issued resource name.
	} else if url != "" {
		return url, nil
	}

	row, err := s.refreshPhotoNames(ctx, placeID, row)
	if err != nil {
		return "", err
	}

	if url, err := s.photoURLFromNames(ctx, row.PhotoNames, index, maxWidth); err != nil {
		return "", err
	} else if url != "" {
		return url, nil
	}

	if row.CoverImageUrl != nil && !isGoogleUserContentURL(*row.CoverImageUrl) {
		return *row.CoverImageUrl, nil
	}
	return "", nil
}

func (s *Service) refreshPhotoNames(
	ctx context.Context,
	placeID string,
	row dbgen.GetPlaceByIDRow,
) (dbgen.GetPlaceByIDRow, error) {
	googlePlaceID := ""
	if row.GooglePlaceID != nil {
		googlePlaceID = strings.TrimSpace(*row.GooglePlaceID)
	}
	if googlePlaceID == "" {
		return row, nil
	}

	resolved, err := s.google.getPlaceDetails(ctx, googlePlaceID, "", true)
	if err != nil {
		return row, err
	}

	cityUUID, err := uuidFromString(row.CityID)
	if err != nil {
		return row, err
	}
	if _, err := dbgen.New(s.db).UpsertPlaceFromGoogle(ctx, upsertParamsFromResolved(cityUUID, resolved)); err != nil {
		return row, err
	}

	placeUUID, err := uuidFromString(placeID)
	if err != nil {
		return row, err
	}
	refreshed, err := dbgen.New(s.db).GetPlaceByID(ctx, placeUUID)
	if err != nil {
		return row, err
	}

	return refreshed, nil
}

func (s *Service) photoURLFromNames(
	ctx context.Context,
	photoNames []string,
	index, maxWidth int,
) (string, error) {
	if !hasPhotoName(photoNames, index) {
		return "", nil
	}
	return s.google.fetchPhotoURL(ctx, photoNames[index], maxWidth)
}

func hasPhotoName(photoNames []string, index int) bool {
	return len(photoNames) > index && strings.TrimSpace(photoNames[index]) != ""
}

func isGoogleUserContentURL(url string) bool {
	return strings.Contains(url, "googleusercontent.com")
}
