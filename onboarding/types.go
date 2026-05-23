package onboarding

type City struct {
	ID            string  `json:"id"`
	Slug          string  `json:"slug"`
	Name          string  `json:"name"`
	CountryCode   string  `json:"country_code"`
	CoverImageURL *string `json:"cover_image_url"`
}

type PurposeCategory struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	IconName string `json:"icon_name"`
}

type Vibe struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Venue struct {
	ID       string   `json:"id"`
	Slug     *string  `json:"slug"`
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Rating   float64  `json:"rating"`
	Price    string   `json:"price"`
	Location string   `json:"location"`
	Tags     []string `json:"tags"`
	Image    *string  `json:"image"`
}

type ListCitiesResponse struct {
	Cities []City `json:"cities"`
}

type ListCategoriesResponse struct {
	Categories []PurposeCategory `json:"categories"`
}

type ListVibesResponse struct {
	Vibes []Vibe `json:"vibes"`
}

type ListVenuesParams struct {
	CityID string `query:"city_id"`
}

type ListVenuesResponse struct {
	Venues []Venue `json:"venues"`
}

type SaveStepParams struct {
	Step int `path:"step"`
}

type SaveStepRequest struct {
	CityIDs     []string `json:"cityIds"`
	CategoryIDs []string `json:"categoryIds"`
	VibeIDs     []string `json:"vibeIds"`
	PlaceIDs    []string `json:"placeIds"`
}

type SaveStepResponse struct {
	Step      int  `json:"step"`
	Completed bool `json:"completed"`
}

type CompleteOnboardingRequest struct {
	Skip bool `json:"skip"`
}

type CompleteOnboardingResponse struct {
	OnboardingStatus string `json:"onboarding_status"`
}
