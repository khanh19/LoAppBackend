package events

type Event struct {
	ID             string  `json:"id"`
	EventName      string  `json:"event_name"`
	EventURL       string  `json:"event_url"`
	VenueName      *string `json:"venue_name"`
	VenueURL       *string `json:"venue_url"`
	ScheduleText   *string `json:"schedule_text"`
	MusicText      *string `json:"music_text"`
	Description    *string `json:"description"`
	PosterImageURL *string `json:"poster_image_url"`
	ImportedAt     string  `json:"imported_at"`
	CitySlug       *string `json:"city_slug"`
	CityName       *string `json:"city_name"`
	SourceSlug     string  `json:"source_slug"`
	SourceName     string  `json:"source_name"`
}

type ListUpcomingEventsParams struct {
	Limit    int    `query:"limit"`
	CitySlug string `query:"city_slug"`
}

type ListUpcomingEventsResponse struct {
	Events []Event `json:"events"`
}

type ListEventsParams struct {
	Limit    int    `query:"limit"`
	Offset   int    `query:"offset"`
	CitySlug string `query:"city_slug"`
}

type ListEventsResponse struct {
	Events  []Event `json:"events"`
	Limit   int     `json:"limit"`
	Offset  int     `json:"offset"`
	HasMore bool    `json:"has_more"`
}
