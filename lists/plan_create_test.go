package lists

import "testing"

func TestNormalizeCreatePlanRequestDefaults(t *testing.T) {
	req := &CreatePlanRequest{
		Title:      " Sunset Tour in D1 ",
		Subtitle:   " 10k + ppl saved - 4 stops ",
		CitySlug:   " hcmc ",
		Occasions:  []string{" Date ", "date", "", "Friends"},
		Visibility: "",
		Stops: []CreatePlanStopRequest{
			{
				PlaceName:    " Bitexco Financial Tower ",
				TimeLabel:    " 05:00 PM ",
				ActivityType: " Meetup ",
				Address:      strPtr(" 2 Hai Trieu "),
				Tags:         []string{" View ", "view"},
			},
			{
				PlaceName: "Nguyen Hue Walking Street",
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
	if normalized.Stops[0].Rank != 1 || normalized.Stops[1].Rank != 2 {
		t.Fatalf("ranks = %d/%d, want 1/2", normalized.Stops[0].Rank, normalized.Stops[1].Rank)
	}
	if normalized.Stops[0].Address == nil || *normalized.Stops[0].Address != "2 Hai Trieu" {
		t.Fatalf("address = %#v, want trimmed value", normalized.Stops[0].Address)
	}
	if got := normalized.Stops[0].Tags; len(got) != 1 || got[0] != "View" {
		t.Fatalf("tags = %#v, want View", got)
	}
}

func TestNormalizeCreatePlanRequestRejectsDuplicateRanks(t *testing.T) {
	_, err := normalizeCreatePlanRequest(&CreatePlanRequest{
		Title:    "Sunset Tour",
		CitySlug: "hcmc",
		Stops: []CreatePlanStopRequest{
			{Rank: 1, PlaceName: "Stop One"},
			{Rank: 1, PlaceName: "Stop Two"},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate rank error")
	}
}

func TestNormalizeCreatePlanRequestRejectsInvalidCoordinates(t *testing.T) {
	lat := 100.0
	_, err := normalizeCreatePlanRequest(&CreatePlanRequest{
		Title:    "Sunset Tour",
		CitySlug: "hcmc",
		Stops: []CreatePlanStopRequest{
			{PlaceName: "Stop One", Latitude: &lat},
		},
	})
	if err == nil {
		t.Fatal("expected invalid latitude error")
	}
}

func strPtr(v string) *string {
	return &v
}
