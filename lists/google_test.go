package lists

import "testing"

func TestParseNeighborhood(t *testing.T) {
	components := []addressComponent{
		{
			LongText:  "District 1",
			ShortText: "Q1",
			Types:     []string{"sublocality_level_1", "political"},
		},
	}
	got := parseNeighborhood(components)
	if got != "District 1" {
		t.Fatalf("got %q, want District 1", got)
	}
}

func TestBuildPlaceTags(t *testing.T) {
	place := &googlePlace{
		PrimaryTypeDisplayName: localizedText{Text: "Cocktail Bar"},
		Types:                  []string{"bar", "point_of_interest", "establishment"},
	}
	tags := buildPlaceTags(place)
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", tags)
	}
}

func TestMapPriceLevel(t *testing.T) {
	level := mapPriceLevel("PRICE_LEVEL_MODERATE")
	if level == nil || *level != 2 {
		t.Fatalf("expected price level 2, got %v", level)
	}
}

func TestCityHintAlt(t *testing.T) {
	if cityHintAlt("Ho Chi Minh City") != "hcmc" {
		t.Fatal("expected hcmc alt hint")
	}
}
