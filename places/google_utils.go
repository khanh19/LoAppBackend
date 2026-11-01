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
		GooglePlaceID:  strings.TrimPrefix(place.ID, "places/"),
		Name:           place.DisplayName.Text,
		Address:        address,
		Ward:           parseWard(place.AddressComponents),
		CityHint:       parseCityHint(place.AddressComponents),
		Latitude:       place.Location.Latitude,
		Longitude:      place.Location.Longitude,
		Rating:         place.Rating,
		PriceLevel:     mapPriceLevel(place.PriceLevel),
		Phone:          place.NationalPhoneNumber,
		Website:        place.WebsiteURI,
		BusinessStatus: place.BusinessStatus,
		Tags:           buildPlaceTags(place),
		GoogleTypes:    append([]string(nil), place.Types...),
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

// parseCityHint returns the Google name for the city a place sits in, used as
// the last resort when resolving a place to one of Lo's cities.
func parseCityHint(components []addressComponent) string {
	fallback := ""
	for _, component := range components {
		text := componentText(component)
		if text == "" {
			continue
		}
		if hasComponentType(component, "locality") {
			return text
		}
		if fallback == "" && hasComponentType(component, "administrative_area_level_1") {
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

func inferVenueCategoryAndArchetype(types []string, priceLevel *int16) (*string, *string) {
	category := stampCategory(types)
	if category == "" {
		return nil, nil
	}
	joined := strings.ToLower(strings.Join(types, " "))

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

// stampCategory maps Google place types onto the venue categories that can be
// stamped. Museums, parks, malls, and shops stay uncategorized so they can be
// added to a plan without becoming stampable. Matching is by whole type, so a
// barber shop is not treated as a bar.
func stampCategory(types []string) string {
	best := ""
	bestRank := len(stampCategoryOrder) + 1
	for _, raw := range types {
		category := categoryForPlaceType(strings.ToLower(strings.TrimSpace(raw)))
		if category == "" {
			continue
		}
		rank := stampCategoryRank(category)
		if rank < bestRank {
			best = category
			bestRank = rank
		}
	}
	return best
}

func categoryForPlaceType(placeType string) string {
	switch placeType {
	case "cafe", "coffee_shop", "bakery":
		return "cafe"
	case "night_club", "nightclub", "club":
		return "club"
	case "bar", "pub", "wine_bar":
		return "bar"
	case "restaurant", "food", "meal_takeaway", "meal_delivery", "food_court":
		return "restaurant"
	}
	switch {
	case strings.HasSuffix(placeType, "_cafe"), strings.HasSuffix(placeType, "_bakery"):
		return "cafe"
	case strings.HasSuffix(placeType, "_club"):
		return "club"
	case strings.HasSuffix(placeType, "_bar"):
		return "bar"
	case strings.HasSuffix(placeType, "_restaurant"):
		return "restaurant"
	default:
		return ""
	}
}

var stampCategoryOrder = []string{"cafe", "club", "bar", "restaurant"}

func stampCategoryRank(category string) int {
	for i, item := range stampCategoryOrder {
		if item == category {
			return i
		}
	}
	return len(stampCategoryOrder)
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
