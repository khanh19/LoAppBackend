package places

type googleResolvedPlace struct {
	GooglePlaceID   string
	Name            string
	Address         string
	Neighborhood    string
	Latitude        float64
	Longitude       float64
	Rating          *float64
	UserRatingCount *int32
	PriceLevel      *int16
	Phone           string
	Website         string
	BusinessStatus  string
	HoursJSON       []byte
	Tags            []string
	PhotoNames      []string
	PhotoName       string
	CoverImageURL   string
}

type autocompleteRequest struct {
	Input                  string        `json:"input"`
	SessionToken           string        `json:"sessionToken,omitempty"`
	IncludedPrimaryTypes   []string      `json:"includedPrimaryTypes,omitempty"`
	LanguageCode           string        `json:"languageCode,omitempty"`
	RegionCode             string        `json:"regionCode,omitempty"`
	LocationBias           *locationBias `json:"locationBias,omitempty"`
}

type autocompleteResponse struct {
	Suggestions []autocompleteSuggestion `json:"suggestions"`
}

type autocompleteSuggestion struct {
	PlacePrediction placePrediction `json:"placePrediction"`
}

type placePrediction struct {
	PlaceID string        `json:"placeId"`
	Text    localizedText `json:"text"`
	StructuredFormat structuredFormat `json:"structuredFormat"`
}

type structuredFormat struct {
	MainText      localizedText `json:"mainText"`
	SecondaryText localizedText `json:"secondaryText"`
}

type textSearchRequest struct {
	TextQuery    string        `json:"textQuery"`
	PageSize     int           `json:"pageSize"`
	LocationBias *locationBias `json:"locationBias,omitempty"`
	LanguageCode string        `json:"languageCode,omitempty"`
	RegionCode   string        `json:"regionCode,omitempty"`
}

type locationBias struct {
	Circle circle `json:"circle"`
}

type circle struct {
	Center center  `json:"center"`
	Radius float64 `json:"radius"`
}

type center struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type googlePlace struct {
	ID                     string             `json:"id"`
	DisplayName            localizedText      `json:"displayName"`
	FormattedAddress       string             `json:"formattedAddress"`
	ShortFormattedAddress  string             `json:"shortFormattedAddress"`
	AddressComponents      []addressComponent `json:"addressComponents"`
	Location               latLng             `json:"location"`
	Rating                 *float64           `json:"rating"`
	UserRatingCount        *int               `json:"userRatingCount"`
	PriceLevel             string             `json:"priceLevel"`
	RegularOpeningHours    *openingHours      `json:"regularOpeningHours"`
	NationalPhoneNumber    string             `json:"nationalPhoneNumber"`
	WebsiteURI             string             `json:"websiteUri"`
	BusinessStatus         string             `json:"businessStatus"`
	PrimaryTypeDisplayName localizedText      `json:"primaryTypeDisplayName"`
	Types                  []string           `json:"types"`
	Photos                 []googlePhoto      `json:"photos"`
}

type openingHours struct {
	WeekdayDescriptions []string `json:"weekdayDescriptions"`
}

type localizedText struct {
	Text string `json:"text"`
}

type addressComponent struct {
	LongText  string   `json:"longText"`
	ShortText string   `json:"shortText"`
	Types     []string `json:"types"`
}

type latLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type googlePhoto struct {
	Name string `json:"name"`
}

type photoMediaResponse struct {
	PhotoURI string `json:"photoUri"`
}
