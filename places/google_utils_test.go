package places

import "testing"

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
