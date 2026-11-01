package places

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var secrets struct {
	GooglePlacesAPIKey string
}

const (
	googlePlacesSearchURL       = "https://places.googleapis.com/v1/places:searchText"
	googlePlacesNearbyURL       = "https://places.googleapis.com/v1/places:searchNearby"
	googlePlacesAutocompleteURL = "https://places.googleapis.com/v1/places:autocomplete"
	googlePlacesDetailsURL      = "https://places.googleapis.com/v1/places/"
	googlePhotoMediaURL         = "https://places.googleapis.com/v1/"

	detailsProFieldMask    = "id,displayName,formattedAddress,shortFormattedAddress,addressComponents,location,rating,userRatingCount,priceLevel,regularOpeningHours,nationalPhoneNumber,websiteUri,businessStatus,types,primaryTypeDisplayName"
	detailsPhotosFieldMask = detailsProFieldMask + ",photos"
	nearbySearchFieldMask  = "places.id,places.displayName,places.formattedAddress,places.shortFormattedAddress,places.addressComponents,places.location,places.rating,places.userRatingCount,places.priceLevel,places.regularOpeningHours,places.nationalPhoneNumber,places.websiteUri,places.businessStatus,places.types,places.primaryTypeDisplayName,places.photos"
)

type googlePlacesClient struct {
	httpClient *http.Client
}

func newGooglePlacesClient() *googlePlacesClient {
	return &googlePlacesClient{
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *googlePlacesClient) autocomplete(ctx context.Context, input, sessionToken string, area searchArea, includedTypes []string) ([]AutocompletePrediction, error) {
	reqBody := autocompleteRequest{
		Input:        input,
		SessionToken: sessionToken,
		LanguageCode: "en",
		RegionCode:   "VN",
	}
	if len(includedTypes) > 0 {
		reqBody.IncludedPrimaryTypes = includedTypes
	}
	// Google rejects a request that carries both a bias and a restriction, so
	// the city rectangle wins when we have one: it keeps results in the city.
	// A client radius override clears Restrict so the circle radius is used.
	switch {
	case area.Restrict != nil:
		reqBody.LocationRestriction = &locationRestriction{Rectangle: *area.Restrict}
	case area.RadiusM > 0:
		reqBody.LocationBias = &locationBias{
			Circle: circle{Center: area.Center, Radius: area.RadiusM},
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var resp autocompleteResponse
	if err := c.doJSONWithRetry(ctx, http.MethodPost, googlePlacesAutocompleteURL, "suggestions.placePrediction.placeId,suggestions.placePrediction.text,suggestions.placePrediction.structuredFormat", body, &resp, sessionToken); err != nil {
		return nil, err
	}

	out := make([]AutocompletePrediction, 0, len(resp.Suggestions))
	for _, suggestion := range resp.Suggestions {
		pred := suggestion.PlacePrediction
		if pred.PlaceID == "" {
			continue
		}
		// Prefer mainText so the title is the place name only (not "Name, Address").
		label := pred.StructuredFormat.MainText.Text
		if label == "" {
			label = pred.Text.Text
		}
		out = append(out, AutocompletePrediction{
			GooglePlaceID: pred.PlaceID,
			Label:         label,
			Secondary:     pred.StructuredFormat.SecondaryText.Text,
		})
	}
	return out, nil
}

func (c *googlePlacesClient) searchNearby(ctx context.Context, lat, lng, radiusM float64, includedTypes []string, maxResults int) ([]*googleResolvedPlace, error) {
	if maxResults <= 0 {
		maxResults = 20
	}
	if maxResults > 20 {
		maxResults = 20
	}
	reqBody := nearbySearchRequest{
		IncludedTypes:  includedTypes,
		MaxResultCount: maxResults,
		LanguageCode:   "en",
		RegionCode:     "VN",
		RankPreference: "DISTANCE",
		LocationRestriction: nearbyLocationRestriction{
			Circle: circle{
				Center: center{Latitude: lat, Longitude: lng},
				Radius: radiusM,
			},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	var resp nearbySearchResponse
	if err := c.doJSONWithRetry(ctx, http.MethodPost, googlePlacesNearbyURL, nearbySearchFieldMask, body, &resp, ""); err != nil {
		return nil, err
	}

	out := make([]*googleResolvedPlace, 0, len(resp.Places))
	for i := range resp.Places {
		place := resp.Places[i]
		if place.ID == "" {
			continue
		}
		resolved := mapGooglePlace(&place, true)
		out = append(out, resolved)
	}
	return out, nil
}

func (c *googlePlacesClient) getPlaceDetails(ctx context.Context, googlePlaceID, sessionToken string, includePhotos bool) (*googleResolvedPlace, error) {
	fieldMask := detailsProFieldMask
	if includePhotos {
		fieldMask = detailsPhotosFieldMask
	}

	endpoint := googlePlacesDetailsURL + strings.TrimPrefix(googlePlaceID, "places/")
	var place googlePlace
	if err := c.doJSONWithRetry(ctx, http.MethodGet, endpoint, fieldMask, nil, &place, sessionToken); err != nil {
		return nil, err
	}

	resolved := mapGooglePlace(&place, includePhotos)
	return resolved, nil
}

func (c *googlePlacesClient) fetchPhotoURL(ctx context.Context, photoName string, maxWidth int) (string, error) {
	endpoint := buildPhotoMediaURL(photoName, maxWidth)
	var resp photoMediaResponse
	if err := c.doJSONWithRetry(ctx, http.MethodGet, endpoint, "", nil, &resp, ""); err != nil {
		return "", err
	}
	return resp.PhotoURI, nil
}

func (c *googlePlacesClient) doJSONWithRetry(ctx context.Context, method, endpoint, fieldMask string, body []byte, out any, sessionToken string) error {
	backoff := []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second}
	var lastErr error

	for _, wait := range backoff {
		if wait > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}

		lastErr = c.doJSON(ctx, method, endpoint, fieldMask, body, out, sessionToken)
		if lastErr == nil {
			return nil
		}
		if !isRetryableGoogleError(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func (c *googlePlacesClient) doJSON(ctx context.Context, method, endpoint, fieldMask string, body []byte, out any, sessionToken string) error {
	if secrets.GooglePlacesAPIKey == "" {
		return fmt.Errorf("google places api key is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, requestBodyReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", secrets.GooglePlacesAPIKey)
	if fieldMask != "" {
		req.Header.Set("X-Goog-FieldMask", fieldMask)
	}
	if sessionToken != "" {
		req.Header.Set("X-Goog-Session-Token", sessionToken)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("google places api error: status=%d body=%s", res.StatusCode, string(payload))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(payload, out)
}

func isRetryableGoogleError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "status=429") ||
		strings.Contains(msg, "status=500") ||
		strings.Contains(msg, "status=502") ||
		strings.Contains(msg, "status=503") ||
		strings.Contains(msg, "status=504")
}

func requestBodyReader(body []byte) io.Reader {
	if len(body) == 0 {
		return nil
	}
	return bytes.NewReader(body)
}
