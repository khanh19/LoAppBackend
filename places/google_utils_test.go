package places

import "testing"

func TestParseWard(t *testing.T) {
	t.Parallel()

	cases := []struct {
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
				{LongText: "Thị xã Sơn Tây", Types: []string{"administrative_area_level_3"}},
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := parseWard(tc.components); got != tc.want {
				t.Fatalf("parseWard() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseCityHint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		components []addressComponent
		want       string
	}{
		{
			name: "prefers the locality",
			components: []addressComponent{
				{LongText: "Hanoi", Types: []string{"locality", "political"}},
				{LongText: "Hà Nội", Types: []string{"administrative_area_level_1", "political"}},
			},
			want: "Hanoi",
		},
		{
			name: "falls back to the province",
			components: []addressComponent{
				{LongText: "Da Nang", Types: []string{"administrative_area_level_1", "political"}},
			},
			want: "Da Nang",
		},
		{
			name:       "no city component",
			components: []addressComponent{{LongText: "Vietnam", Types: []string{"country"}}},
			want:       "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := parseCityHint(tc.components); got != tc.want {
				t.Fatalf("parseCityHint() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStampCategory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		types []string
		want  string
	}{
		{name: "museum", types: []string{"museum", "tourist_attraction"}, want: ""},
		{name: "park", types: []string{"park"}, want: ""},
		{name: "mall", types: []string{"shopping_mall"}, want: ""},
		{name: "shop", types: []string{"clothing_store"}, want: ""},
		{name: "barber is not a bar", types: []string{"barber_shop"}, want: ""},
		{name: "restaurant", types: []string{"restaurant"}, want: "restaurant"},
		{name: "specific restaurant", types: []string{"italian_restaurant"}, want: "restaurant"},
		{name: "food", types: []string{"food"}, want: "restaurant"},
		{name: "cafe", types: []string{"cafe"}, want: "cafe"},
		{name: "bakery", types: []string{"bakery"}, want: "cafe"},
		{name: "bar", types: []string{"bar"}, want: "bar"},
		{name: "club", types: []string{"night_club"}, want: "club"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := stampCategory(tc.types)
			if got != tc.want {
				t.Fatalf("stampCategory(%v) = %q, want %q", tc.types, got, tc.want)
			}
		})
	}
}
