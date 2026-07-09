package lists

type PlaceListSummary struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Area        string   `json:"area"`
	Occasions   []string `json:"occasions"`
	CitySlug      string  `json:"city_slug"`
	CityName      string  `json:"city_name"`
	EntryCount    int32   `json:"entry_count"`
	CoverImageURL *string `json:"cover_image_url"`
}

type PlaceListEntry struct {
	Rank           int32    `json:"rank"`
	SeedName       string   `json:"seed_name"`
	Note           string   `json:"note"`
	PlaceID        string   `json:"place_id"`
	Name           string   `json:"name"`
	GooglePlaceID  *string  `json:"google_place_id"`
	Address        *string  `json:"address"`
	Neighborhood   *string  `json:"neighborhood"`
	Latitude       float64  `json:"latitude"`
	Longitude      float64  `json:"longitude"`
	Rating         float64  `json:"rating"`
	PriceLevel     *int16   `json:"price_level"`
	CoverImageURL  *string  `json:"cover_image_url"`
	Tags           []string `json:"tags"`
}

type PlaceListDetail struct {
	ID        string           `json:"id"`
	Slug      string           `json:"slug"`
	Title     string           `json:"title"`
	Category  string           `json:"category"`
	Area      string           `json:"area"`
	Occasions []string         `json:"occasions"`
	CitySlug  string           `json:"city_slug"`
	CityName  string           `json:"city_name"`
	Entries   []PlaceListEntry `json:"entries"`
}

type ListPlaceListsResponse struct {
	Lists []PlaceListSummary `json:"lists"`
}

type GetPlaceListResponse struct {
	List PlaceListDetail `json:"list"`
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
	ID             string `json:"id"`
	StartedAt      string `json:"started_at"`
	FinishedAt     string `json:"finished_at"`
	Status         string `json:"status"`
	EntriesAdded   int32  `json:"entries_added"`
	EntriesSkipped int32  `json:"entries_skipped"`
	EntriesFailed  int32              `json:"entries_failed"`
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
