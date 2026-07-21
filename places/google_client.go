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
	googlePlacesSearchURL      = "https://places.googleapis.com/v1/places:searchText"
	googlePlacesAutocompleteURL = "https://places.googleapis.com/v1/places:autocomplete"
	googlePlacesDetailsURL     = "https://places.googleapis.com/v1/places/"
	googlePhotoMediaURL        = "https://places.googleapis.com/v1/"

	detailsProFieldMask = "id,displayName,formattedAddress,shortFormattedAddress,addressComponents,location,rating,userRatingCount,priceLevel,regularOpeningHours,nationalPhoneNumber,websiteUri,businessStatus,types,primaryTypeDisplayName"
	detailsPhotosFieldMask = detailsProFieldMask + ",photos"

	autocompletePrimaryTypes = "restaurant,cafe,bar,bakery,meal_takeaway,meal_delivery,food"
)

type googlePlacesClient struct {
	httpClient *http.Client
}

func newGooglePlacesClient() *googlePlacesClient {
	return &googlePlacesClient{
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *googlePlacesClient) autocomplete(ctx context.Context, input, sessionToken string, lat, lng float64) ([]AutocompletePrediction, error) {
	reqBody := autocompleteRequest{
		Input:                input,
		SessionToken:         sessionToken,
		IncludedPrimaryTypes: strings.Split(autocompletePrimaryTypes, ","),
		LanguageCode:         "en",
		RegionCode:           "VN",
	}
	if lat != 0 || lng != 0 {
		reqBody.LocationBias = &locationBias{
			Circle: circle{
				Center: center{Latitude: lat, Longitude: lng},
				Radius: 15000,
			},
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
		label := pred.Text.Text
		if label == "" {
			label = pred.StructuredFormat.MainText.Text
		}
		out = append(out, AutocompletePrediction{
			GooglePlaceID: pred.PlaceID,
			Label:         label,
			Secondary:     pred.StructuredFormat.SecondaryText.Text,
		})
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
	if includePhotos && resolved.PhotoName != "" {
		photoURL, err := c.fetchPhotoURL(ctx, resolved.PhotoName, 800)
		if err == nil && photoURL != "" {
			resolved.CoverImageURL = photoURL
		}
	}
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
