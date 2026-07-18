package lists

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizePlansLimit(t *testing.T) {
	tests := []struct {
		limit int
		want  int
	}{
		{limit: 0, want: defaultPlansLimit},
		{limit: -1, want: defaultPlansLimit},
		{limit: 5, want: 5},
		{limit: 10, want: 10},
		{limit: 100, want: maxPlansLimit},
	}
	for _, tt := range tests {
		if got := normalizePlansLimit(tt.limit); got != tt.want {
			t.Fatalf("normalizePlansLimit(%d) = %d, want %d", tt.limit, got, tt.want)
		}
	}
}

func TestOptionalCitySlug(t *testing.T) {
	if got := optionalCitySlug(""); got != nil {
		t.Fatalf("empty slug = %#v, want nil", got)
	}
	got := optionalCitySlug("hcmc")
	if got == nil || *got != "hcmc" {
		t.Fatalf("hcmc slug = %#v, want hcmc", got)
	}
}

func TestPlanSummaryJSONFields(t *testing.T) {
	cover := "https://example.com/cover.jpg"
	creator := "Minh Nguyen"
	data, err := json.Marshal(PlanSummary{
		ID:                 "id-1",
		Slug:               "sunset-tour",
		Title:              "Sunset Tour",
		Subtitle:           "An evening walk",
		Category:           "Classic",
		Area:               "District 1",
		CitySlug:           "hcmc",
		CityName:           "Ho Chi Minh City",
		SavesCount:         42,
		StopsCount:         4,
		CoverImageURL:      &cover,
		CreatorDisplayName: &creator,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`"slug":"sunset-tour"`,
		`"saves_count":42`,
		`"stops_count":4`,
		`"cover_image_url":"https://example.com/cover.jpg"`,
		`"creator_display_name":"Minh Nguyen"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("JSON = %s, missing %s", got, want)
		}
	}
}

func TestListPlansResponseShape(t *testing.T) {
	data, err := json.Marshal(ListPlansResponse{Plans: []PlanSummary{}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `{"plans":[]}` {
		t.Fatalf("JSON = %s, want {\"plans\":[]}", data)
	}
}
