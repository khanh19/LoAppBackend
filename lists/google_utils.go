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
		Ward:          parseWard(place.AddressComponents),
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

// parseWard picks the ward, the lowest administrative unit left after
// Vietnam's 2025 reform removed districts. Components that still name a
// district are ignored rather than used as a fallback.
func parseWard(components []addressComponent) string {
	fallback := ""
	for _, component := range components {
		text := componentText(component)
		if text == "" || looksLikeDistrict(text) {
			continue
		}
		if looksLikeWard(text) {
			return text
		}
		if fallback == "" && hasComponentType(component, "administrative_area_level_3", "sublocality_level_1") {
			fallback = text
		}
	}
	return fallback
}

var wardPrefixes = []string{"phường", "ward", "xã", "commune"}

func looksLikeWard(text string) bool {
	lower := strings.ToLower(text)
	for _, prefix := range wardPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

var districtPrefixes = []string{"quận", "quan ", "huyện", "huyen ", "thị xã", "thi xa"}

func looksLikeDistrict(text string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "district") {
		return true
	}
	for _, prefix := range districtPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func componentText(component addressComponent) string {
	if text := strings.TrimSpace(component.LongText); text != "" {
		return text
	}
	return strings.TrimSpace(component.ShortText)
}

func hasComponentType(component addressComponent, types ...string) bool {
	for _, t := range component.Types {
		for _, want := range types {
			if t == want {
				return true
			}
		}
	}
	return false
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
