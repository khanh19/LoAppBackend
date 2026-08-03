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
		GoogleTypes:   append([]string(nil), place.Types...),
	}
	cat, arch := inferVenueCategoryAndArchetype(place.Types, resolved.PriceLevel)
	resolved.VenueCategory = cat
	resolved.VenueArchetype = arch
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

func inferVenueCategoryAndArchetype(types []string, priceLevel *int16) (*string, *string) {
	joined := strings.ToLower(strings.Join(types, " "))
	var category string
	switch {
	case strings.Contains(joined, "cafe") || strings.Contains(joined, "coffee") || strings.Contains(joined, "bakery"):
		category = "cafe"
	case strings.Contains(joined, "night_club") || strings.Contains(joined, "nightclub"):
		category = "club"
	case strings.Contains(joined, "bar") || strings.Contains(joined, "pub") || strings.Contains(joined, "wine_bar"):
		category = "bar"
	case strings.Contains(joined, "restaurant") || strings.Contains(joined, "meal_takeaway") || strings.Contains(joined, "food"):
		category = "restaurant"
	default:
		return nil, nil
	}

	archetype := ""
	switch category {
	case "restaurant":
		switch {
		case priceLevel != nil && *priceLevel == 1:
			archetype = "street_food"
		case priceLevel != nil && *priceLevel == 2:
			archetype = "casual"
		case priceLevel != nil && *priceLevel == 3:
			archetype = "midrange"
		case priceLevel != nil && *priceLevel >= 4:
			archetype = "fine_dining"
		case strings.Contains(joined, "meal_takeaway") || strings.Contains(joined, "fast_food"):
			archetype = "street_food"
		default:
			archetype = "casual"
		}
	case "cafe":
		if priceLevel != nil && *priceLevel >= 3 {
			archetype = "chain"
		} else {
			archetype = "specialty_coffee"
		}
	case "bar":
		switch {
		case strings.Contains(joined, "wine"):
			archetype = "cocktail_bar"
		case priceLevel != nil && *priceLevel >= 3:
			archetype = "cocktail_bar"
		case priceLevel != nil && *priceLevel == 1:
			archetype = "pub"
		default:
			archetype = "craft_beer"
		}
	case "club":
		archetype = "nightclub"
	}

	return &category, &archetype
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
