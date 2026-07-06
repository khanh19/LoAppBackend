package lists

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
	googlePlacesSearchURL = "https://places.googleapis.com/v1/places:searchText"
	googlePhotoMediaURL   = "https://places.googleapis.com/v1/"
	searchFieldMask       = "places.id,places.displayName,places.shortFormattedAddress,places.addressComponents,places.location,places.rating,places.userRatingCount,places.priceLevel,places.primaryTypeDisplayName,places.types,places.photos"
)

type googlePlacesClient struct {
	httpClient *http.Client
}

func newGooglePlacesClient() *googlePlacesClient {
	return &googlePlacesClient{
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *googlePlacesClient) resolvePlace(ctx context.Context, query string, lat, lng float64) (*googleResolvedPlace, error) {
	place, err := c.searchText(ctx, query, lat, lng)
	if err != nil {
		return nil, err
	}

	resolved := mapGooglePlace(place)
	if resolved.PhotoName != "" {
		photoURL, err := c.fetchPhotoURL(ctx, resolved.PhotoName)
		if err == nil && photoURL != "" {
			resolved.CoverImageURL = photoURL
		}
	}
	return resolved, nil
}

func (c *googlePlacesClient) searchText(ctx context.Context, query string, lat, lng float64) (*googlePlace, error) {
	reqBody := textSearchRequest{
		TextQuery:    query,
		PageSize:     1,
		LanguageCode: "en",
		RegionCode:   "VN",
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

	var resp textSearchResponse
	err = c.doJSONWithRetry(ctx, http.MethodPost, googlePlacesSearchURL, searchFieldMask, body, &resp)
	if err != nil {
		return nil, err
	}
	if len(resp.Places) == 0 {
		return nil, fmt.Errorf("no places found for query %q", query)
	}
	return &resp.Places[0], nil
}

func (c *googlePlacesClient) fetchPhotoURL(ctx context.Context, photoName string) (string, error) {
	photoName = strings.TrimPrefix(photoName, "/")
	endpoint := googlePhotoMediaURL + photoName + "/media?maxWidthPx=800&skipHttpRedirect=true"

	var resp photoMediaResponse
	if err := c.doJSONWithRetry(ctx, http.MethodGet, endpoint, "", nil, &resp); err != nil {
		return "", err
	}
	return resp.PhotoURI, nil
}

func (c *googlePlacesClient) doJSONWithRetry(ctx context.Context, method, endpoint, fieldMask string, body []byte, out any) error {
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

		lastErr = c.doJSON(ctx, method, endpoint, fieldMask, body, out)
		if lastErr == nil {
			return nil
		}
		if !isRetryableGoogleError(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func (c *googlePlacesClient) doJSON(ctx context.Context, method, endpoint, fieldMask string, body []byte, out any) error {
	if secrets.GooglePlacesAPIKey == "" {
		return fmt.Errorf("google places api key is not configured")
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", secrets.GooglePlacesAPIKey)
	if fieldMask != "" {
		req.Header.Set("X-Goog-FieldMask", fieldMask)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("google places api %s %s: %s", method, endpoint, strings.TrimSpace(string(respBody)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode google places response: %w", err)
	}
	return nil
}

func isRetryableGoogleError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "500") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "504") ||
		strings.Contains(msg, "resource_exhausted")
}
