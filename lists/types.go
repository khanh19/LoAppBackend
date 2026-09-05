package lists

type PlaceListSummary struct {
	ID            string   `json:"id"`
	Slug          string   `json:"slug"`
	Title         string   `json:"title"`
	Category      string   `json:"category"`
	Area          string   `json:"area"`
	Occasions     []string `json:"occasions"`
	CitySlug      string   `json:"city_slug"`
	CityName      string   `json:"city_name"`
	EntryCount    int32    `json:"entry_count"`
	CoverPlaceID  *string  `json:"cover_place_id"`
	CoverImageURL *string  `json:"cover_image_url"`
}

type PlaceListEntry struct {
	StopOrder     int32    `json:"stop_order"`
	SeedName      string   `json:"seed_name"`
	Note          string   `json:"note"`
	TimeLabel     string   `json:"time_label"`
	ActivityType  string   `json:"activity_type"`
	ImageURL      *string  `json:"image_url"`
	PlaceID       string   `json:"place_id"`
	Name          string   `json:"name"`
	GooglePlaceID *string  `json:"google_place_id"`
	Address       *string  `json:"address"`
	Neighborhood  *string  `json:"neighborhood"`
	Latitude      float64  `json:"latitude"`
	Longitude     float64  `json:"longitude"`
	Rating        float64  `json:"rating"`
	PriceLevel    *int16   `json:"price_level"`
	CoverImageURL *string  `json:"cover_image_url"`
	Tags          []string `json:"tags"`
}

type PlaceListCreator struct {
	UserID      string  `json:"user_id"`
	DisplayName *string `json:"display_name"`
	Username    string  `json:"username"`
	AvatarURL   *string `json:"avatar_url"`
}

type PlaceListDetail struct {
	ID         string            `json:"id"`
	Slug       string            `json:"slug"`
	Title      string            `json:"title"`
	Subtitle   string            `json:"subtitle"`
	Category   string            `json:"category"`
	Area       string            `json:"area"`
	Occasions  []string          `json:"occasions"`
	ListType   string            `json:"list_type"`
	Visibility string            `json:"visibility"`
	SavesCount int32             `json:"saves_count"`
	StopsCount int32             `json:"stops_count"`
	CitySlug   string            `json:"city_slug"`
	CityName   string            `json:"city_name"`
	Creator    *PlaceListCreator `json:"creator,omitempty"`
	Entries    []PlaceListEntry  `json:"entries"`
}

type ListPlaceListsResponse struct {
	Lists []PlaceListSummary `json:"lists"`
}

type GetPlaceListResponse struct {
	List PlaceListDetail `json:"list"`
}

type CreatePlanRequest struct {
	Slug       string                  `json:"slug"`
	Title      string                  `json:"title"`
	Subtitle   string                  `json:"subtitle"`
	CitySlug   string                  `json:"city_slug"`
	Category   string                  `json:"category"`
	Area       string                  `json:"area"`
	Occasions  []string                `json:"occasions"`
	Visibility string                  `json:"visibility"`
	SavesCount int32                   `json:"saves_count"`
	Stops      []CreatePlanStopRequest `json:"stops"`
}

type CreatePlanStopRequest struct {
	StopOrder     int32    `json:"stop_order"`
	TimeLabel     string   `json:"time_label"`
	ActivityType  string   `json:"activity_type"`
	PlaceID       *string  `json:"place_id"`
	PlaceName     string   `json:"place_name"`
	GooglePlaceID *string  `json:"google_place_id"`
	Address       *string  `json:"address"`
	Neighborhood  *string  `json:"neighborhood"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	Note          string   `json:"note"`
	ImageURL      *string  `json:"image_url"`
	Tags          []string `json:"tags"`
}

type CreatePlanResponse struct {
	Plan PlaceListDetail `json:"plan"`
}

type ListPlansParams struct {
	Limit    int    `query:"limit"`
	CitySlug string `query:"city_slug"`
}

type PlanSummary struct {
	ID                 string  `json:"id"`
	Slug               string  `json:"slug"`
	Title              string  `json:"title"`
	Subtitle           string  `json:"subtitle"`
	Category           string  `json:"category"`
	Area               string  `json:"area"`
	CitySlug           string  `json:"city_slug"`
	CityName           string  `json:"city_name"`
	SavesCount         int32   `json:"saves_count"`
	StopsCount         int32   `json:"stops_count"`
	CoverPlaceID       *string `json:"cover_place_id"`
	CoverImageURL      *string `json:"cover_image_url"`
	CreatorDisplayName *string `json:"creator_display_name"`
}

type ListPlansResponse struct {
	Plans []PlanSummary `json:"plans"`
}

type SyncPlaceListsResponse struct {
	RunID          string `json:"run_id"`
	Status         string `json:"status"`
	EntriesAdded   int32  `json:"entries_added"`
	EntriesSkipped int32  `json:"entries_skipped"`
	EntriesFailed  int32  `json:"entries_failed"`
}

type ListSyncRunsParams struct {
	Limit int `query:"limit"`
}

type SyncRunSummary struct {
	ID             string            `json:"id"`
	StartedAt      string            `json:"started_at"`
	FinishedAt     string            `json:"finished_at"`
	Status         string            `json:"status"`
	EntriesAdded   int32             `json:"entries_added"`
	EntriesSkipped int32             `json:"entries_skipped"`
	EntriesFailed  int32             `json:"entries_failed"`
	Details        []SyncEntryResult `json:"details"`
}

type ListSyncRunsResponse struct {
	Runs []SyncRunSummary `json:"runs"`
}

type SyncEntryResult struct {
	ListSlug string `json:"list_slug"`
	SeedName string `json:"seed_name"`
	Status   string `json:"status"`
	PlaceID  string `json:"place_id,omitempty"`
	Error    string `json:"error,omitempty"`
}
