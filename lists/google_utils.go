package lists

import (
	"net/url"
	"strings"
	"unicode"
)

func mapGooglePlace(place *googlePlace) *googleResolvedPlace {
	resolved := &googleResolvedPlace{
		GooglePlaceID: place.ID,
		Name:          place.DisplayName.Text,
		Address:       place.ShortFormattedAddress,
		Neighborhood:  parseNeighborhood(place.AddressComponents),
		Latitude:      place.Location.Latitude,
		Longitude:     place.Location.Longitude,
		Rating:        place.Rating,
		PriceLevel:    mapPriceLevel(place.PriceLevel),
		Tags:          buildPlaceTags(place),
	}
	if len(place.Photos) > 0 {
		resolved.PhotoName = place.Photos[0].Name
	}
	return resolved
}

func parseNeighborhood(components []addressComponent) string {
	for _, component := range components {
		for _, t := range component.Types {
			if t == "sublocality_level_1" || t == "administrative_area_level_2" {
				if component.LongText != "" {
					return component.LongText
				}
				return component.ShortText
			}
		}
	}
	return ""
}

func mapPriceLevel(level string) *int16 {
	switch level {
	case "PRICE_LEVEL_INEXPENSIVE":
		v := int16(1)
		return &v
	case "PRICE_LEVEL_MODERATE":
		v := int16(2)
		return &v
	case "PRICE_LEVEL_EXPENSIVE":
		v := int16(3)
		return &v
	case "PRICE_LEVEL_VERY_EXPENSIVE":
		v := int16(4)
		return &v
	default:
		return nil
	}
}

func buildPlaceTags(place *googlePlace) []string {
	seen := make(map[string]struct{})
	tags := make([]string, 0, len(place.Types)+1)

	if text := strings.TrimSpace(place.PrimaryTypeDisplayName.Text); text != "" {
		seen[strings.ToLower(text)] = struct{}{}
		tags = append(tags, text)
	}

	for _, t := range place.Types {
		if isIgnoredPlaceType(t) {
			continue
		}
		label := humanizePlaceType(t)
		key := strings.ToLower(label)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tags = append(tags, label)
	}
	return tags
}

func isIgnoredPlaceType(t string) bool {
	switch t {
	case "point_of_interest", "establishment", "food", "store", "premise":
		return true
	default:
		return false
	}
}

func humanizePlaceType(t string) string {
	parts := strings.Split(t, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func cityHintAlt(cityHint string) string {
	switch strings.ToLower(strings.TrimSpace(cityHint)) {
	case "ho chi minh city", "hcmc", "saigon":
		return "hcmc"
	case "hanoi", "ha noi":
		return "hanoi"
	case "da nang", "da nang city", "danang":
		return "danang"
	default:
		return cityHint
	}
}

// Used only in tests to build photo URLs.
func buildPhotoMediaURL(photoName string) (*url.URL, error) {
	photoName = strings.TrimPrefix(photoName, "/")
	return url.Parse(googlePhotoMediaURL + photoName + "/media?maxWidthPx=800&skipHttpRedirect=true")
}
