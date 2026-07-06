package lists

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	_ "embed"
	"fmt"
	"sort"
)

//go:embed seeds/seed-lists.json
var seedListsJSON []byte

type seedFile struct {
	Lists []seedList `json:"lists"`
}

type seedList struct {
	ListID    string     `json:"list_id"`
	Title     string     `json:"title"`
	Category  string     `json:"category"`
	Area      string     `json:"area"`
	Occasions []string   `json:"occasions"`
	Items     []seedItem `json:"items"`
}

type seedItem struct {
	Order         int    `json:"order"`
	SeedName      string `json:"seed_name"`
	CityHint      string `json:"city_hint"`
	DistrictHint  string `json:"district_hint"`
	AddressHint   string `json:"address_hint"`
	Note          string `json:"note"`
	PlaceID       any    `json:"place_id"`
	Resolved      bool   `json:"resolved"`
}

func loadSeedFile() (*seedFile, error) {
	var file seedFile
	if err := json.Unmarshal(seedListsJSON, &file); err != nil {
		return nil, fmt.Errorf("parse seed lists: %w", err)
	}
	return &file, nil
}

func hashSeedList(list seedList) string {
	payload := struct {
		ListID    string     `json:"list_id"`
		Title     string     `json:"title"`
		Category  string     `json:"category"`
		Area      string     `json:"area"`
		Occasions []string   `json:"occasions"`
		Items     []seedItem `json:"items"`
	}{
		ListID:    list.ListID,
		Title:     list.Title,
		Category:  list.Category,
		Area:      list.Area,
		Occasions: sortedCopy(list.Occasions),
		Items:     normalizeSeedItems(list.Items),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		b = []byte(list.ListID)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func normalizeSeedItems(items []seedItem) []seedItem {
	out := make([]seedItem, len(items))
	copy(out, items)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].SeedName < out[j].SeedName
		}
		return out[i].Order < out[j].Order
	})
	for i := range out {
		out[i].PlaceID = nil
		out[i].Resolved = false
	}
	return out
}

func sortedCopy(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func (item seedItem) locationHint() string {
	if item.AddressHint != "" {
		return item.AddressHint
	}
	return item.DistrictHint
}

func buildSearchQuery(item seedItem) string {
	parts := []string{item.SeedName}
	if hint := item.locationHint(); hint != "" {
		parts = append(parts, hint)
	}
	if item.CityHint != "" {
		parts = append(parts, item.CityHint)
	}
	return stringsJoin(parts, ", ")
}

func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += sep + part
	}
	return out
}
