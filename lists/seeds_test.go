package lists

import (
	"testing"
)

func TestHashSeedListStable(t *testing.T) {
	list := seedList{
		ListID:   "d1-cocktail-circuit",
		Title:    "The D1 Cocktail Circuit",
		Category: "Cocktail bars",
		Area:     "District 1",
		Occasions: []string{"Date", "Squad"},
		Items: []seedItem{
			{Order: 1, SeedName: "Summer Experiment", CityHint: "Ho Chi Minh City", Note: "Farm-to-bar"},
		},
	}

	first := hashSeedList(list)
	second := hashSeedList(list)
	if first != second {
		t.Fatalf("expected stable hash, got %q and %q", first, second)
	}

	list.Items[0].PlaceID = "ignored"
	list.Items[0].Resolved = true
	third := hashSeedList(list)
	if first != third {
		t.Fatalf("expected hash to ignore place_id/resolved, got %q and %q", first, third)
	}
}

func TestBuildSearchQuery(t *testing.T) {
	item := seedItem{
		SeedName:    "Phở Hòa Pasteur",
		AddressHint: "260C Pasteur",
		CityHint:    "Ho Chi Minh City",
	}
	got := buildSearchQuery(item)
	want := "Phở Hòa Pasteur, 260C Pasteur, Ho Chi Minh City"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLoadSeedFile(t *testing.T) {
	file, err := loadSeedFile()
	if err != nil {
		t.Fatalf("loadSeedFile: %v", err)
	}
	if len(file.Lists) == 0 {
		t.Fatal("expected seeded lists")
	}
}
