package stamps

type CreateStampRequest struct {
	PlaceID     string   `json:"place_id"`
	Verdict     string   `json:"verdict"`
	Quality     string   `json:"quality"`
	Companions  []string `json:"companions"`
	Vibes       []string `json:"vibes"`
	Note        string   `json:"note"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}

type PairwiseCandidate struct {
	PlaceID      string   `json:"place_id"`
	Name         string   `json:"name"`
	Neighborhood *string  `json:"neighborhood"`
	CoverImageURL *string `json:"cover_image_url"`
	PriceLevel   *int16   `json:"price_level"`
	Archetype    *string  `json:"venue_archetype"`
	Comparability float64 `json:"comparability"`
	Tier         int      `json:"tier"`
	Reasons      []string `json:"reasons"`
	IsProbe      bool     `json:"is_probe"`
}

type CreateStampResponse struct {
	StampID          string              `json:"stamp_id"`
	PersonalScore    float64             `json:"personal_score"`
	Band             string              `json:"band"`
	SkipPairwise     bool                `json:"skip_pairwise"`
	Candidates       []PairwiseCandidate `json:"candidates"`
	PlacementSession *string             `json:"placement_session_id,omitempty"`
}

type SubmitPairwiseRequest struct {
	OpponentPlaceID string `json:"opponent_place_id"`
	Winner          string `json:"winner"` // this | other | tie
}

type SubmitPairwiseResponse struct {
	PersonalScore       float64             `json:"personal_score"`
	PlacementConfidence float64             `json:"placement_confidence"`
	IsComplete          bool                `json:"is_complete"`
	NextCandidates      []PairwiseCandidate `json:"next_candidates,omitempty"`
}

type PoolEntry struct {
	PlaceID             string  `json:"place_id"`
	Name                string  `json:"name"`
	Band                string  `json:"band"`
	Score               float64 `json:"score"`
	PlacementConfidence float64 `json:"placement_confidence"`
	IsProvisional       bool    `json:"is_provisional"`
	Neighborhood        *string `json:"neighborhood"`
	CoverImageURL       *string `json:"cover_image_url"`
	PriceLevel          *int16  `json:"price_level"`
}

type GetPoolResponse struct {
	Category string      `json:"category"`
	Entries  []PoolEntry `json:"entries"`
}

type FamiliarityRequest struct {
	PlaceID string `json:"place_id"`
	Status  string `json:"status"` // been | not_been
}

type FamiliarityResponse struct {
	PlaceID string `json:"place_id"`
	Status  string `json:"status"`
}

type NextPairwiseResponse struct {
	Skip       bool                `json:"skip"`
	PlaceID    string              `json:"place_id,omitempty"`
	PlaceName  string              `json:"place_name,omitempty"`
	Band       string              `json:"band,omitempty"`
	Candidates []PairwiseCandidate `json:"candidates,omitempty"`
	StampID    *string             `json:"stamp_id,omitempty"`
}

type StandalonePairwiseRequest struct {
	PlaceID         string `json:"place_id"`
	OpponentPlaceID string `json:"opponent_place_id"`
	Winner          string `json:"winner"`
	StampID         string `json:"stamp_id,omitempty"`
}

type PhotoUploadURLRequest struct {
	ContentType string `json:"content_type"`
	Label       string `json:"label"`
}

type PhotoUploadURLResponse struct {
	UploadURL   string `json:"upload_url"`
	StoragePath string `json:"storage_path"`
	Token       string `json:"token,omitempty"`
}

type ConfirmPhotoRequest struct {
	StoragePath string `json:"storage_path"`
	Label       string `json:"label"`
	SortOrder   int    `json:"sort_order"`
}

type ConfirmPhotoResponse struct {
	PhotoID     string `json:"photo_id"`
	StoragePath string `json:"storage_path"`
	Label       string `json:"label"`
}

type StampSummary struct {
	StampCount        int                `json:"stamp_count"`
	FinalScore        *float64           `json:"final_score"`
	DisplayState      string             `json:"display_state"`
	SentimentScore    *float64           `json:"sentiment_score"`
	QualityScore      *float64           `json:"quality_score"`
	VibeConflict      bool               `json:"vibe_conflict"`
	QualityConflict   bool               `json:"quality_conflict"`
	ConflictNote      *string            `json:"conflict_note"`
	Companions        []TagAggregate     `json:"companions"`
	Vibes             []TagAggregate     `json:"vibes"`
}

type TagAggregate struct {
	Slug       string  `json:"slug"`
	Percentage float64 `json:"percentage"`
	RawCount   int     `json:"raw_count"`
}
