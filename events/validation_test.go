package events

import "testing"

func TestNormalizeUpcomingLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "default", limit: 0, want: 4},
		{name: "custom", limit: 6, want: 6},
		{name: "max", limit: 99, want: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeUpcomingLimit(tt.limit); got != tt.want {
				t.Fatalf("normalizeUpcomingLimit(%d) = %d, want %d", tt.limit, got, tt.want)
			}
		})
	}
}

func TestNormalizeListLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "default", limit: 0, want: 50},
		{name: "custom", limit: 25, want: 25},
		{name: "max", limit: 200, want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeListLimit(tt.limit); got != tt.want {
				t.Fatalf("normalizeListLimit(%d) = %d, want %d", tt.limit, got, tt.want)
			}
		})
	}
}

func TestNormalizeOffset(t *testing.T) {
	if got := normalizeOffset(-5); got != 0 {
		t.Fatalf("normalizeOffset(-5) = %d, want 0", got)
	}
	if got := normalizeOffset(10); got != 10 {
		t.Fatalf("normalizeOffset(10) = %d, want 10", got)
	}
}
