package places

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode"
)

func mapGooglePlace(place *googlePlace, includePhotos bool) *googleResolvedPlace {
	address := place.ShortFormattedAddress
	if address == "" {
		address = place.FormattedAddress
	}

	resolved := &googleResolvedPlace{
		GooglePlaceID: place.ID,
		Name:          place.DisplayName.Text,
		Address:       address,
		Neighborhood:  parseNeighborhood(place.AddressComponents),
		Latitude:      place.Location.Latitude,
		Longitude:     place.Location.Longitude,
		Rating:        place.Rating,
		PriceLevel:    mapPriceLevel(place.PriceLevel),
		Phone:         place.NationalPhoneNumber,
		Website:       place.WebsiteURI,
		BusinessStatus: place.BusinessStatus,
		Tags:          buildPlaceTags(place),
	}
	if place.UserRatingCount != nil {
		v := int32(*place.UserRatingCount)
		resolved.UserRatingCount = &v
	}
	if place.RegularOpeningHours != nil {
		raw, err := json.Marshal(place.RegularOpeningHours)
		if err == nil {
			resolved.HoursJSON = raw
		}
	}
	if includePhotos && len(place.Photos) > 0 {
		names := make([]string, 0, len(place.Photos))
		for _, photo := range place.Photos {
			if photo.Name != "" {
				names = append(names, photo.Name)
			}
		}
		resolved.PhotoNames = names
		if len(names) > 0 {
			resolved.PhotoName = names[0]
		}
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

func buildPhotoMediaURL(photoName string, maxWidth int) string {
	photoName = strings.TrimPrefix(photoName, "/")
	if maxWidth <= 0 {
		maxWidth = 800
	}
	return googlePhotoMediaURL + photoName + "/media?maxWidthPx=" + strconv.Itoa(maxWidth) + "&skipHttpRedirect=true"
}
