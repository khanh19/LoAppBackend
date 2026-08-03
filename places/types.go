package places

import "encoding/json"

type PlaceDetail struct {
	ID              string          `json:"id"`
	GooglePlaceID   *string         `json:"google_place_id"`
	Name            string          `json:"name"`
	Address         *string         `json:"address"`
	Neighborhood    *string         `json:"neighborhood"`
	Latitude        float64         `json:"latitude"`
	Longitude       float64         `json:"longitude"`
	Rating          float64         `json:"rating"`
	UserRatingCount *int32          `json:"user_rating_count"`
	PriceLevel      *int16          `json:"price_level"`
	Hours           string          `json:"hours"`
	HoursJSON       json.RawMessage `json:"hours_json,omitempty"`
	Phone           *string         `json:"phone"`
	Website         *string         `json:"website"`
	BusinessStatus  *string         `json:"business_status"`
	CoverImageURL   *string         `json:"cover_image_url"`
	PhotoURLs       []string        `json:"photo_urls"`
	Tags            []string        `json:"tags"`
	VenueCategory   *string         `json:"venue_category"`
	VenueArchetype  *string         `json:"venue_archetype"`
	CitySlug        string          `json:"city_slug"`
	CityName        string          `json:"city_name"`
	LastSyncedAt    string          `json:"last_synced_at"`
	IsStale         bool            `json:"is_stale"`
	StampSummary    *StampSummary   `json:"stamp_summary,omitempty"`
}

// StampSummary is embedded on place detail; mirrors stamps.StampSummary fields.
type StampSummary struct {
	StampCount      int            `json:"stamp_count"`
	FinalScore      *float64       `json:"final_score"`
	DisplayState    string         `json:"display_state"`
	SentimentScore  *float64       `json:"sentiment_score"`
	QualityScore    *float64       `json:"quality_score"`
	VibeConflict    bool           `json:"vibe_conflict"`
	QualityConflict bool           `json:"quality_conflict"`
	ConflictNote    *string        `json:"conflict_note"`
	Companions      []TagAggregate `json:"companions"`
	Vibes           []TagAggregate `json:"vibes"`
}

type TagAggregate struct {
	Slug       string  `json:"slug"`
	Percentage float64 `json:"percentage"`
	RawCount   int     `json:"raw_count"`
}

type GetPlaceResponse struct {
	Place PlaceDetail `json:"place"`
}

type GetPlacePhotoParams struct {
	Index int `query:"index"`
	W     int `query:"w"`
}

type GetPlacePhotoResponse struct {
	PhotoURL string `json:"photo_url"`
}

type AutocompleteRequest struct {
	Input        string  `json:"input"`
	SessionToken string  `json:"session_token"`
	CitySlug     string  `json:"city_slug"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
}

type AutocompletePrediction struct {
	GooglePlaceID string `json:"google_place_id"`
	Label         string `json:"label"`
	Secondary     string `json:"secondary"`
}

type AutocompleteResponse struct {
	Predictions []AutocompletePrediction `json:"predictions"`
}

type ResolvePlaceRequest struct {
	GooglePlaceID string  `json:"google_place_id"`
	SessionToken  string  `json:"session_token"`
	CitySlug      string  `json:"city_slug"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
}

type ResolvePlaceForPlanRequest struct {
	GooglePlaceID string `json:"google_place_id"`
	CitySlug      string `json:"city_slug"`
}

type ResolvePlaceForPlanResponse struct {
	PlaceID string `json:"place_id"`
}

type ResolvePlaceResponse struct {
	PlaceID string      `json:"place_id"`
	Place   PlaceDetail `json:"place"`
}
