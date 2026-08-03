package stamps

import (
	"math"
	"testing"
)

func TestScoreAtPositionSingle(t *testing.T) {
	band, _ := BandForVerdict(VerdictMustRepeat)
	score := ScoreAtPosition(band, 1, 1, 0.40, defaultPlacement)
	if score != 8.5 {
		t.Fatalf("expected 8.5 for single must-repeat, got %v", score)
	}
}

func TestScoreAtPositionTwoWithFloor(t *testing.T) {
	band, _ := BandForVerdict(VerdictMustRepeat)
	s := PlacementEvidenceRatio(0.69, 2, defaultPlacement)
	if s != 0.40 {
		t.Fatalf("expected s floor 0.40, got %v", s)
	}
	low := ScoreAtPosition(band, 1, 2, s, defaultPlacement)
	high := ScoreAtPosition(band, 2, 2, s, defaultPlacement)
	if low != 8.4 {
		t.Fatalf("expected low 8.4, got %v", low)
	}
	if high != 8.6 {
		t.Fatalf("expected high 8.6, got %v", high)
	}
	if high <= low {
		t.Fatalf("expected high > low")
	}
}

func TestBayesianShrinkOneStamp(t *testing.T) {
	final := BayesianShrink(9.21, 0.8, "bar", defaultShrinkage)
	if final != 6.9 {
		t.Fatalf("expected 6.9, got %v", final)
	}
}

func TestHaversineNear(t *testing.T) {
	// ~111m per 0.001 deg latitude
	d := HaversineMeters(21.0285, 105.8542, 21.0295, 105.8542)
	if d < 100 || d > 130 {
		t.Fatalf("expected ~111m, got %v", d)
	}
}

func TestNarrowInterval(t *testing.T) {
	lo, hi := NarrowInterval(1, 3, 2, true)
	if lo != 3 || hi != 3 {
		t.Fatalf("prefer new: got lo=%d hi=%d", lo, hi)
	}
	lo, hi = NarrowInterval(1, 3, 2, false)
	if lo != 1 || hi != 2 {
		t.Fatalf("prefer other: got lo=%d hi=%d", lo, hi)
	}
}

func TestOrderedInsert(t *testing.T) {
	got := OrderedInsert([]string{"a", "b"}, "x", 2)
	if len(got) != 3 || got[0] != "a" || got[1] != "x" || got[2] != "b" {
		t.Fatalf("unexpected order: %v", got)
	}
}

func TestCommunityCompositeWithoutPairwise(t *testing.T) {
	c := CommunityComposite(9.0, 9.5, 0, 1, defaultComposite)
	expected := (0.50/0.85)*9.0 + (0.35/0.85)*9.5
	if math.Abs(c-expected) > 0.01 {
		t.Fatalf("expected ~%v, got %v", expected, c)
	}
}

func TestArchetypesCompatible(t *testing.T) {
	a := "cocktail_bar"
	b := "craft_beer"
	if ArchetypesCompatible(&a, &b) {
		t.Fatal("expected incompatible")
	}
	if !ArchetypesCompatible(&a, &a) {
		t.Fatal("expected compatible")
	}
	if ArchetypesCompatible(nil, &a) {
		t.Fatal("nil should be incompatible at MVP")
	}
}

func TestDisplayState(t *testing.T) {
	if DisplayState(1) != "new" || DisplayState(5) != "early" || DisplayState(15) != "full" {
		t.Fatal("display state thresholds wrong")
	}
}

func TestOrdinalRecoverySimulation(t *testing.T) {
	// Synthetic: 8 venues with true utilities 1..8. Simulate noisy pairwise
	// placement and check Spearman correlation after full binary search.
	band, _ := BandForVerdict(VerdictMustRepeat)
	trueOrder := []string{"v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8"}
	utilities := map[string]float64{
		"v1": 1, "v2": 2, "v3": 3, "v4": 4, "v5": 5, "v6": 6, "v7": 7, "v8": 8,
	}

	placed := []string{}
	comps := map[string]int{}
	for _, venue := range trueOrder {
		lo, hi := 1, len(placed)+1
		for lo < hi {
			mid := MidIndex(lo, hi)
			opp := placed[mid-1]
			preferNew := utilities[venue] > utilities[opp]
			lo, hi = NarrowInterval(lo, hi, mid, preferNew)
			comps[venue]++
			comps[opp]++
		}
		placed = OrderedInsert(placed, venue, lo)
	}

	scores := RecomputeBandScores(band, placed, 20, comps, defaultPlacement)
	// Spearman: rank by score vs true utility order
	var concordant, total int
	for i := 0; i < len(placed); i++ {
		for j := i + 1; j < len(placed); j++ {
			total++
			si, sj := scores[placed[i]].Score, scores[placed[j]].Score
			ui, uj := utilities[placed[i]], utilities[placed[j]]
			if (si-sj)*(ui-uj) > 0 {
				concordant++
			}
		}
	}
	rho := float64(concordant) / float64(total)
	if rho < 0.99 {
		t.Fatalf("expected near-perfect recovery with noiseless answers, got rho=%v placed=%v", rho, placed)
	}
}
