package lists

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStringSliceOrEmpty(t *testing.T) {
	if got := stringSliceOrEmpty(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil -> %#v, want empty non-nil slice", got)
	}
	in := []string{"a"}
	if got := stringSliceOrEmpty(in); len(got) != 1 || got[0] != "a" {
		t.Fatalf("non-nil -> %#v", got)
	}
}

func TestCreatePlanStopRequestUsesStopOrderJSONField(t *testing.T) {
	data, err := json.Marshal(CreatePlanStopRequest{StopOrder: 2})
	if err != nil {
		t.Fatalf("marshal stop: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"stop_order":2`) {
		t.Fatalf("JSON = %s, want stop_order", got)
	}
	if strings.Contains(got, `"order"`) || strings.Contains(got, `"rank"`) {
		t.Fatalf("JSON = %s, contains legacy order field", got)
	}
}

func TestNormalizeCreatePlanRequestDefaults(t *testing.T) {
	req := &CreatePlanRequest{
		Title:      " Sunset Tour in D1 ",
		Subtitle:   " 10k + ppl saved - 4 stops ",
		CitySlug:   " hcmc ",
		Occasions:  []string{" Date ", "date", "", "Friends"},
		Visibility: "",
		Stops: []CreatePlanStopRequest{
			{
				PlaceName:     " Bitexco Financial Tower ",
				GooglePlaceID: strPtr("google-place-bitexco"),
				TimeLabel:     " 05:00 PM ",
				ActivityType:  " Meetup ",
				Address:       strPtr(" 2 Hai Trieu "),
				Tags:          []string{" View ", "view"},
			},
			{
				PlaceName:     "Nguyen Hue Walking Street",
				GooglePlaceID: strPtr("google-place-nguyen-hue"),
			},
		},
	}

	normalized, err := normalizeCreatePlanRequest(req)
	if err != nil {
		t.Fatalf("normalizeCreatePlanRequest: %v", err)
	}
	if normalized.Slug != "sunset-tour-in-d1" {
		t.Fatalf("slug = %q, want sunset-tour-in-d1", normalized.Slug)
	}
	if normalized.Visibility != "public" {
		t.Fatalf("visibility = %q, want public", normalized.Visibility)
	}
	if got := normalized.Occasions; len(got) != 2 || got[0] != "Date" || got[1] != "Friends" {
		t.Fatalf("occasions = %#v, want Date/Friends", got)
	}
	if normalized.Stops[0].StopOrder != 1 || normalized.Stops[1].StopOrder != 2 {
		t.Fatalf("stop orders = %d/%d, want 1/2", normalized.Stops[0].StopOrder, normalized.Stops[1].StopOrder)
	}
	if normalized.Stops[0].Address == nil || *normalized.Stops[0].Address != "2 Hai Trieu" {
		t.Fatalf("address = %#v, want trimmed value", normalized.Stops[0].Address)
	}
	if got := normalized.Stops[0].Tags; len(got) != 1 || got[0] != "View" {
		t.Fatalf("tags = %#v, want View", got)
	}
}

func TestNormalizeCreatePlanRequestRejectsDuplicateStopOrders(t *testing.T) {
	_, err := normalizeCreatePlanRequest(&CreatePlanRequest{
		Title:    "Sunset Tour",
		CitySlug: "hcmc",
		Stops: []CreatePlanStopRequest{
			{StopOrder: 1, PlaceName: "Stop One", GooglePlaceID: strPtr("g1")},
			{StopOrder: 1, PlaceName: "Stop Two", GooglePlaceID: strPtr("g2")},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate stop_order error")
	}
}

func TestNormalizeCreatePlanRequestRejectsDuplicatePlaceNames(t *testing.T) {
	_, err := normalizeCreatePlanRequest(&CreatePlanRequest{
		Title:    "Sunset Tour",
		CitySlug: "hcmc",
		Stops: []CreatePlanStopRequest{
			{PlaceName: "Bitexco Financial Tower", GooglePlaceID: strPtr("g-bitexco")},
			{PlaceName: " bitexco financial tower ", GooglePlaceID: strPtr("g-bitexco-2")},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate place name error")
	}
}

func TestNormalizeCreatePlanRequestRejectsStopWithoutPlaceReference(t *testing.T) {
	_, err := normalizeCreatePlanRequest(&CreatePlanRequest{
		Title:    "Sunset Tour",
		CitySlug: "hcmc",
		Stops: []CreatePlanStopRequest{
			{PlaceName: "Stop One"},
		},
	})
	if err == nil {
		t.Fatal("expected missing place reference error")
	}
}

func TestNormalizeCreatePlanRequestRejectsInvalidCoordinates(t *testing.T) {
	lat := 100.0
	_, err := normalizeCreatePlanRequest(&CreatePlanRequest{
		Title:    "Sunset Tour",
		CitySlug: "hcmc",
		Stops: []CreatePlanStopRequest{
			{PlaceName: "Stop One", GooglePlaceID: strPtr("g-stop"), Latitude: &lat},
		},
	})
	if err == nil {
		t.Fatal("expected invalid latitude error")
	}
}

func strPtr(v string) *string {
	return &v
}
