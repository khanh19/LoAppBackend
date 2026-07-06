package lists

type googleResolvedPlace struct {
	GooglePlaceID string
	Name          string
	Address       string
	Neighborhood  string
	Latitude      float64
	Longitude     float64
	Rating        *float64
	PriceLevel    *int16
	Tags          []string
	PhotoName     string
	CoverImageURL string
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

type textSearchResponse struct {
	Places []googlePlace `json:"places"`
}

type googlePlace struct {
	ID                     string             `json:"id"`
	DisplayName            localizedText      `json:"displayName"`
	ShortFormattedAddress  string             `json:"shortFormattedAddress"`
	AddressComponents      []addressComponent `json:"addressComponents"`
	Location               latLng             `json:"location"`
	Rating                 *float64           `json:"rating"`
	UserRatingCount        *int               `json:"userRatingCount"`
	PriceLevel             string             `json:"priceLevel"`
	PrimaryTypeDisplayName localizedText      `json:"primaryTypeDisplayName"`
	Types                  []string           `json:"types"`
	Photos                 []googlePhoto      `json:"photos"`
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
