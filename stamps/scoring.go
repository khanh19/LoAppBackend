package stamps

import (
	"math"
	"sort"
)

const (
	VerdictMustRepeat = "must-repeat"
	VerdictGoodPick   = "good-pick"
	VerdictMid        = "mid"
	VerdictNotForMe   = "not-for-me"

	QualityOutstanding = "outstanding"
	QualityGood        = "good"
	QualityOkay        = "okay"
	QualityNotGood     = "not-good"

	VerifyBooked    = "booked"
	VerifyGPS       = "gps"
	VerifyRepeat    = "repeat"
	VerifyFirstTime = "first_time"
)

type BandBounds struct {
	Lo float64
	Hi float64
}

type PlacementConfig struct {
	SMin float64
	SMax float64
	KMin int
}

type ShrinkageConfig struct {
	M float64
	C map[string]float64
}

type CompositeWeights struct {
	Sentiment              float64
	Quality                float64
	Pairwise               float64
	PairwiseMinComparisons int
}

type ComparabilityWeights struct {
	Bisection float64
	Geo       float64
	Price     float64
	Vibe      float64
	Recency   float64
}

var defaultBands = map[string]BandBounds{
	VerdictMustRepeat: {Lo: 7.0, Hi: 10.0},
	VerdictGoodPick:   {Lo: 4.5, Hi: 7.0},
	VerdictMid:        {Lo: 2.5, Hi: 4.5},
	VerdictNotForMe:   {Lo: 0.0, Hi: 2.5},
}

var defaultVerdictValues = map[string]float64{
	VerdictMustRepeat: 9.0,
	VerdictGoodPick:   6.5,
	VerdictMid:        4.0,
	VerdictNotForMe:   1.5,
}

var defaultQualityValues = map[string]float64{
	QualityOutstanding: 9.5,
	QualityGood:        7.0,
	QualityOkay:        4.5,
	QualityNotGood:     1.5,
}

var defaultVerificationWeights = map[string]float64{
	VerifyBooked:    1.0,
	VerifyGPS:       0.8,
	VerifyRepeat:    0.5,
	VerifyFirstTime: 0.3,
}

var defaultPlacement = PlacementConfig{SMin: 0.40, SMax: 1.00, KMin: 8}

var defaultShrinkage = ShrinkageConfig{
	M: 5,
	C: map[string]float64{
		"restaurant": 6.5,
		"cafe":       6.5,
		"bar":        6.5,
		"club":       6.5,
		"other":      6.5,
	},
}

var defaultComposite = CompositeWeights{
	Sentiment:              0.50,
	Quality:                0.35,
	Pairwise:               0.15,
	PairwiseMinComparisons: 5,
}

var defaultComparability = ComparabilityWeights{
	Bisection: 0.40,
	Geo:       0.20,
	Price:     0.15,
	Vibe:      0.15,
	Recency:   0.10,
}

func BandForVerdict(verdict string) (BandBounds, bool) {
	b, ok := defaultBands[verdict]
	return b, ok
}

func VerdictValue(verdict string) float64 {
	if v, ok := defaultVerdictValues[verdict]; ok {
		return v
	}
	return 4.0
}

func QualityValue(quality string) float64 {
	if v, ok := defaultQualityValues[quality]; ok {
		return v
	}
	return 4.5
}

func VerificationWeight(level string) float64 {
	if v, ok := defaultVerificationWeights[level]; ok {
		return v
	}
	return 0.3
}

func AdjacentBands(band string) []string {
	switch band {
	case VerdictMustRepeat:
		return []string{VerdictGoodPick}
	case VerdictGoodPick:
		return []string{VerdictMustRepeat, VerdictMid}
	case VerdictMid:
		return []string{VerdictGoodPick, VerdictNotForMe}
	case VerdictNotForMe:
		return []string{VerdictMid}
	default:
		return nil
	}
}

// PlacementEvidenceRatio returns s in [sMin, sMax].
func PlacementEvidenceRatio(eBand float64, k int, cfg PlacementConfig) float64 {
	if cfg.SMin == 0 && cfg.SMax == 0 {
		cfg = defaultPlacement
	}
	if k < 1 {
		k = 1
	}
	needed := float64(k) * math.Log2(float64(k)+1)
	if needed <= 0 {
		return cfg.SMin
	}
	s := eBand / needed
	if s < cfg.SMin {
		return cfg.SMin
	}
	if s > cfg.SMax {
		return cfg.SMax
	}
	return s
}

// ScoreAtPosition computes the personal score for the i-th of k venues (1-indexed, worst to best).
func ScoreAtPosition(band BandBounds, i, k int, s float64, cfg PlacementConfig) float64 {
	if cfg.KMin <= 0 {
		cfg = defaultPlacement
	}
	if k < 1 {
		k = 1
	}
	if i < 1 {
		i = 1
	}
	if i > k {
		i = k
	}
	denom := float64(k)
	if denom < float64(cfg.KMin) {
		denom = float64(cfg.KMin)
	}
	center := float64(k+1) / 2
	inside := 0.5 + s*((float64(i)-center)/denom)
	rangeWidth := band.Hi - band.Lo
	score := band.Lo + rangeWidth*inside
	return math.Round(score*10) / 10
}

// RecomputeBandScores assigns rank keys and scores for an ordered band (worst→best).
func RecomputeBandScores(band BandBounds, placeIDs []string, eBand float64, comparisonsPerPlace map[string]int, cfg PlacementConfig) map[string]PoolScore {
	k := len(placeIDs)
	s := PlacementEvidenceRatio(eBand, k, cfg)
	neededComps := int(math.Ceil(math.Log2(float64(k) + 1)))
	if neededComps < 1 {
		neededComps = 1
	}

	out := make(map[string]PoolScore, k)
	for idx, placeID := range placeIDs {
		i := idx + 1
		rankKey := float64(i)
		if k == 1 {
			rankKey = 1
		}
		comps := comparisonsPerPlace[placeID]
		conf := float64(comps) / float64(neededComps)
		if conf > 1 {
			conf = 1
		}
		out[placeID] = PoolScore{
			RankKey:              rankKey,
			Score:                ScoreAtPosition(band, i, k, s, cfg),
			PlacementConfidence:  conf,
			ComparisonsCount:     comps,
			EvidenceRatio:        s,
		}
	}
	return out
}

type PoolScore struct {
	RankKey             float64
	Score               float64
	PlacementConfidence float64
	ComparisonsCount    int
	EvidenceRatio       float64
}

// InsertPosition after a prefer-new or prefer-other answer.
// preferNew=true means the new venue beats the opponent at mid.
func NarrowInterval(lo, hi, mid int, preferNew bool) (newLo, newHi int) {
	if preferNew {
		return mid + 1, hi
	}
	return lo, mid
}

// ResolvedIndex returns the insert position when lo==hi.
func ResolvedIndex(lo, hi int) (int, bool) {
	if lo >= hi {
		return lo, true
	}
	return 0, false
}

func MidIndex(lo, hi int) int {
	return (lo + hi) / 2
}

// HaversineMeters returns distance between two WGS84 points in meters.
func HaversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0
	toRad := math.Pi / 180
	dLat := (lat2 - lat1) * toRad
	dLon := (lon2 - lon1) * toRad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*toRad)*math.Cos(lat2*toRad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

func ResolveVerification(visitCount int, deviceLat, deviceLng *float64, placeLat, placeLng float64, gpsRadiusMeters float64) (level string, weight float64) {
	if gpsRadiusMeters <= 0 {
		gpsRadiusMeters = 200
	}
	if deviceLat != nil && deviceLng != nil && placeLat != 0 && placeLng != 0 {
		if HaversineMeters(*deviceLat, *deviceLng, placeLat, placeLng) <= gpsRadiusMeters {
			return VerifyGPS, VerificationWeight(VerifyGPS)
		}
	}
	if visitCount >= 2 {
		return VerifyRepeat, VerificationWeight(VerifyRepeat)
	}
	return VerifyFirstTime, VerificationWeight(VerifyFirstTime)
}

func CommunityComposite(sentiment, quality, pairwiseStrength float64, pairwiseCount int, weights CompositeWeights) float64 {
	if weights.Sentiment == 0 && weights.Quality == 0 {
		weights = defaultComposite
	}
	sentW, qualW, pairW := weights.Sentiment, weights.Quality, weights.Pairwise
	if pairwiseCount < weights.PairwiseMinComparisons {
		total := sentW + qualW
		if total <= 0 {
			return sentiment
		}
		sentW = sentW / total
		qualW = qualW / total
		pairW = 0
	}
	return sentW*sentiment + qualW*quality + pairW*pairwiseStrength
}

func BayesianShrink(composite, v float64, category string, cfg ShrinkageConfig) float64 {
	if cfg.M == 0 {
		cfg = defaultShrinkage
	}
	c := cfg.C[category]
	if c == 0 {
		c = 6.5
	}
	final := (v/(v+cfg.M))*composite + (cfg.M/(v+cfg.M))*c
	return math.Round(final*10) / 10
}

func DisplayState(stampCount int) string {
	switch {
	case stampCount >= 15:
		return "full"
	case stampCount >= 5:
		return "early"
	default:
		return "new"
	}
}

func WeightedMean(values, weights []float64) float64 {
	var sum, wSum float64
	for i := range values {
		sum += values[i] * weights[i]
		wSum += weights[i]
	}
	if wSum == 0 {
		return 0
	}
	return sum / wSum
}

func Jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	setA := make(map[string]struct{}, len(a))
	for _, x := range a {
		setA[x] = struct{}{}
	}
	inter := 0
	setB := make(map[string]struct{}, len(b))
	for _, x := range b {
		setB[x] = struct{}{}
		if _, ok := setA[x]; ok {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

type ComparabilityInput struct {
	BisectionScore float64 // 0..1
	SameNeighborhood bool
	SameCity         bool
	PriceA, PriceB   *int16
	VibesA, VibesB   []string
	RecencyScore     float64 // 0..1
}

func ScoreComparability(in ComparabilityInput, w ComparabilityWeights) float64 {
	if w.Bisection == 0 && w.Geo == 0 {
		w = defaultComparability
	}
	geo := 0.0
	if in.SameNeighborhood {
		geo = 1.0
	} else if in.SameCity {
		geo = 0.4
	}
	price := 0.0
	if in.PriceA != nil && in.PriceB != nil {
		d := int(*in.PriceA) - int(*in.PriceB)
		if d < 0 {
			d = -d
		}
		switch d {
		case 0:
			price = 1.0
		case 1:
			price = 0.5
		}
	}
	vibe := Jaccard(in.VibesA, in.VibesB)
	score := w.Bisection*in.BisectionScore +
		w.Geo*geo +
		w.Price*price +
		w.Vibe*vibe +
		w.Recency*in.RecencyScore
	return math.Round(score*10000) / 10000
}

func BuildComparabilityReasons(in ComparabilityInput, archetype string) []string {
	reasons := make([]string, 0, 4)
	if archetype != "" {
		reasons = append(reasons, humanArchetype(archetype))
	}
	if in.SameNeighborhood {
		reasons = append(reasons, "Same neighborhood")
	} else if in.SameCity {
		reasons = append(reasons, "Same city")
	}
	if in.PriceA != nil && in.PriceB != nil && *in.PriceA == *in.PriceB {
		reasons = append(reasons, "Same price range")
	} else if in.PriceA != nil && in.PriceB != nil {
		d := int(*in.PriceA) - int(*in.PriceB)
		if d < 0 {
			d = -d
		}
		if d == 1 {
			reasons = append(reasons, "Similar price")
		}
	}
	return reasons
}

func humanArchetype(a string) string {
	switch a {
	case "street_food":
		return "Both street food"
	case "casual":
		return "Both casual spots"
	case "midrange":
		return "Both mid-range"
	case "fine_dining":
		return "Both fine dining"
	case "cocktail_bar":
		return "Both cocktail bars"
	case "craft_beer":
		return "Both craft beer bars"
	case "pub":
		return "Both pubs"
	case "rooftop":
		return "Both rooftop bars"
	case "specialty_coffee":
		return "Both specialty cafés"
	case "study_cafe":
		return "Both study cafés"
	case "chain":
		return "Both café chains"
	case "nightclub":
		return "Both clubs"
	case "live_music":
		return "Both live music venues"
	case "lounge":
		return "Both lounges"
	default:
		return ""
	}
}

// OrderedInsert merges a new place into an ordered list of place IDs at 1-based position.
func OrderedInsert(existing []string, placeID string, position int) []string {
	if position < 1 {
		position = 1
	}
	if position > len(existing)+1 {
		position = len(existing) + 1
	}
	out := make([]string, 0, len(existing)+1)
	out = append(out, existing[:position-1]...)
	out = append(out, placeID)
	out = append(out, existing[position-1:]...)
	return out
}

// SortPlaceIDsByRankKey sorts place IDs ascending by rank key.
func SortPlaceIDsByRankKey(ids []string, rankKeys map[string]float64) []string {
	out := append([]string(nil), ids...)
	sort.SliceStable(out, func(i, j int) bool {
		return rankKeys[out[i]] < rankKeys[out[j]]
	})
	return out
}

func ArchetypesCompatible(a, b *string) bool {
	if a == nil || b == nil || *a == "" || *b == "" {
		return false
	}
	return *a == *b
}

func PriceWithinOne(a, b *int16) bool {
	if a == nil || b == nil {
		return true // permit null price but soft-penalize elsewhere
	}
	d := int(*a) - int(*b)
	if d < 0 {
		d = -d
	}
	return d <= 1
}
