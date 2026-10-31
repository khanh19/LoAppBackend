package places

import "testing"

var hanoi = cityConfig{
	Slug:          "hanoi",
	Name:          "Hanoi",
	Latitude:      21.0285,
	Longitude:     105.8542,
	SwLat:         20.83,
	SwLng:         105.6,
	NeLat:         21.23,
	NeLng:         106.06,
	SearchRadiusM: 25000,
}

func TestCityContains(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		city     cityConfig
		lat, lng float64
		want     bool
	}{
		{name: "inside", city: hanoi, lat: 21.0278, lng: 105.8342, want: true},
		{name: "on the corner", city: hanoi, lat: 20.83, lng: 105.6, want: true},
		{name: "another city", city: hanoi, lat: 10.7769, lng: 106.7009, want: false},
		{name: "no coordinates", city: hanoi, want: false},
		{name: "city without bounds", city: cityConfig{Slug: "hue"}, lat: 16.46, lng: 107.59, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.city.contains(tc.lat, tc.lng); got != tc.want {
				t.Fatalf("contains(%v, %v) = %v, want %v", tc.lat, tc.lng, got, tc.want)
			}
		})
	}
}

func TestCitySearchAreaBiasesToTheUserInsideTheCity(t *testing.T) {
	t.Parallel()

	area := hanoi.searchArea(21.0278, 105.8342)
	if area.Center.Latitude != 21.0278 || area.Center.Longitude != 105.8342 {
		t.Fatalf("expected the user's position as the bias center, got %+v", area.Center)
	}
	if area.RadiusM != 25000 {
		t.Fatalf("expected the city radius, got %v", area.RadiusM)
	}
	if area.Restrict == nil || area.Restrict.Low.Latitude != hanoi.SwLat || area.Restrict.High.Longitude != hanoi.NeLng {
		t.Fatalf("expected the city bounds as the restriction, got %+v", area.Restrict)
	}
}

func TestCitySearchAreaFallsBackToTheCityCenter(t *testing.T) {
	t.Parallel()

	area := hanoi.searchArea(10.7769, 106.7009)
	if area.Center.Latitude != hanoi.Latitude || area.Center.Longitude != hanoi.Longitude {
		t.Fatalf("expected the city center as the bias center, got %+v", area.Center)
	}
}

func TestCitySearchAreaWithoutBoundsHasNoRestriction(t *testing.T) {
	t.Parallel()

	city := cityConfig{Slug: "hue", Latitude: 16.46, Longitude: 107.59, SearchRadiusM: 15000}
	area := city.searchArea(0, 0)
	if area.Restrict != nil {
		t.Fatalf("expected no restriction, got %+v", area.Restrict)
	}
	if area.RadiusM != 15000 {
		t.Fatalf("expected the city radius, got %v", area.RadiusM)
	}
}

func TestCitySearchAreaWithRadiusOverrideDropsRestriction(t *testing.T) {
	t.Parallel()

	area := hanoi.searchAreaWithRadius(21.0278, 105.8342, 5000)
	if area.Restrict != nil {
		t.Fatalf("expected no city rectangle when radius is overridden, got %+v", area.Restrict)
	}
	if area.RadiusM != 5000 {
		t.Fatalf("expected override radius 5000, got %v", area.RadiusM)
	}
	if area.Center.Latitude != 21.0278 || area.Center.Longitude != 105.8342 {
		t.Fatalf("expected user center, got %+v", area.Center)
	}
}

func TestClampPrimaryTypes(t *testing.T) {
	t.Parallel()

	got := clampPrimaryTypes([]string{" restaurant ", "cafe", "restaurant", "", "bar", "bakery", "night_club", "park"})
	if len(got) != 5 {
		t.Fatalf("expected 5 types, got %v", got)
	}
	if got[0] != "restaurant" || got[1] != "cafe" || got[4] != "night_club" {
		t.Fatalf("unexpected order/values: %v", got)
	}
}
