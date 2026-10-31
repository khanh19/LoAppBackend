package lists

import "testing"

func TestParseWard(t *testing.T) {
	tests := []struct {
		name       string
		components []addressComponent
		want       string
	}{
		{
			name: "prefers a Phường component over the enclosing district",
			components: []addressComponent{
				{LongText: "Phường Bến Nghé", Types: []string{"administrative_area_level_3", "political"}},
				{LongText: "District 1", ShortText: "Q1", Types: []string{"sublocality_level_1", "political"}},
			},
			want: "Phường Bến Nghé",
		},
		{
			name: "accepts the English Ward spelling",
			components: []addressComponent{
				{LongText: "Ward 5", Types: []string{"political"}},
			},
			want: "Ward 5",
		},
		{
			name: "accepts communes outside the urban core",
			components: []addressComponent{
				{LongText: "Xã Vĩnh Ngọc", Types: []string{"administrative_area_level_3"}},
			},
			want: "Xã Vĩnh Ngọc",
		},
		{
			name: "falls back to a sublocality that is not a district",
			components: []addressComponent{
				{LongText: "Thảo Điền", Types: []string{"sublocality_level_1", "political"}},
			},
			want: "Thảo Điền",
		},
		{
			name: "ignores district-looking components entirely",
			components: []addressComponent{
				{LongText: "Quận Hoàn Kiếm", Types: []string{"administrative_area_level_3"}},
				{LongText: "District 1", Types: []string{"sublocality_level_1"}},
				{LongText: "Huyện Củ Chi", Types: []string{"administrative_area_level_3"}},
			},
			want: "",
		},
		{
			name: "ignores components that are neither ward nor sublocality",
			components: []addressComponent{
				{LongText: "Hanoi", Types: []string{"locality", "political"}},
				{LongText: "Vietnam", Types: []string{"country", "political"}},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseWard(tt.components); got != tt.want {
				t.Fatalf("parseWard() = %q, want %q", got, tt.want)
			}
		})
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
