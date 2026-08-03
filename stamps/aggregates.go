package stamps

import (
	"context"
	"errors"
	"math"
	"strings"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func recomputeVenueAggregateTx(ctx context.Context, q *dbgen.Queries, placeID pgtype.UUID, category string) error {
	rows, err := q.ListPlaceStampsForAggregate(ctx, placeID)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to list stamps for aggregate")
	}

	stampCount := len(rows)
	sentVals := make([]float64, 0, stampCount)
	qualVals := make([]float64, 0, stampCount)
	weights := make([]float64, 0, stampCount)
	var weightedSum float64
	for _, row := range rows {
		w := numericToFloat(row.Weight)
		if w <= 0 {
			w = 0.3
		}
		sentVals = append(sentVals, VerdictValue(row.Verdict))
		qualVals = append(qualVals, QualityValue(row.Quality))
		weights = append(weights, w)
		weightedSum += w
	}

	var sentiment, quality *float64
	if stampCount > 0 {
		s := WeightedMean(sentVals, weights)
		qScore := WeightedMean(qualVals, weights)
		sentiment = &s
		quality = &qScore
	}

	pairCount, err := q.CountPlacePairwise(ctx, placeID)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to count pairwise")
	}
	winCount, err := q.CountPlacePairwiseWins(ctx, placeID)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to count wins")
	}
	var pairwiseStrength *float64
	ps := 0.0
	if pairCount > 0 {
		ps = float64(winCount) / float64(pairCount)
		pairwiseStrength = &ps
	}

	var composite, final *float64
	if stampCount > 0 && sentiment != nil && quality != nil {
		c := CommunityComposite(*sentiment, *quality, ps, int(pairCount), defaultComposite)
		f := BayesianShrink(c, weightedSum, category, defaultShrinkage)
		composite = &c
		final = &f
	}

	// Tag aggregates + conflict detection
	tagVotes, err := q.ListPlaceTagVotes(ctx, placeID)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to list tag votes")
	}
	type tagAcc struct {
		weighted float64
		raw      int
		custom   bool
	}
	byType := map[string]map[string]*tagAcc{
		"companion": {},
		"vibe":      {},
	}
	typeWeightSum := map[string]float64{"companion": 0, "vibe": 0}
	for _, tv := range tagVotes {
		if tv.IsCustom {
			continue // exclude custom from vibe DNA until promotion
		}
		m := byType[tv.TagType]
		if m == nil {
			continue
		}
		acc := m[tv.TagSlug]
		if acc == nil {
			acc = &tagAcc{}
			m[tv.TagSlug] = acc
		}
		w := numericToFloat(tv.Weight)
		acc.weighted += w
		acc.raw++
		typeWeightSum[tv.TagType] += w
	}

	if err := q.DeleteVenueTagAggregates(ctx, placeID); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to clear tag aggregates")
	}
	vibePct := map[string]float64{}
	for tagType, m := range byType {
		total := typeWeightSum[tagType]
		for slug, acc := range m {
			pct := 0.0
			if total > 0 {
				pct = (acc.weighted / total) * 100
			}
			if tagType == "vibe" {
				vibePct[slug] = pct
			}
			if err := q.UpsertVenueTagAggregate(ctx, dbgen.UpsertVenueTagAggregateParams{
				PlaceID:       placeID,
				TagType:       tagType,
				TagSlug:       slug,
				WeightedCount: numericFromFloat(acc.weighted),
				RawCount:      int32(acc.raw),
				Percentage:    numericFromFloat(math.Round(pct*100) / 100),
			}); err != nil {
				return errs.WrapCode(err, errs.Internal, "failed to upsert tag aggregate")
			}
		}
	}

	vibeConflict := false
	conflictNote := (*string)(nil)
	opposites := [][2]string{
		{"chill", "lively"},
		{"cozy", "lively"},
		{"romantic", "lively"},
	}
	for _, pair := range opposites {
		if vibePct[pair[0]] > 30 && vibePct[pair[1]] > 30 {
			vibeConflict = true
			note := "Vibe varies depending on the night"
			conflictNote = &note
			break
		}
	}

	qualityConflict := false
	if stampCount > 0 && sentiment != nil && quality != nil {
		// High sentiment (>= ~ Must Repeat heavy) vs low quality
		mustRepeatWeight := 0.0
		notWorthWeight := 0.0
		for i, row := range rows {
			w := weights[i]
			if row.Verdict == VerdictMustRepeat {
				mustRepeatWeight += w
			}
			if row.Quality == QualityNotGood {
				notWorthWeight += w
			}
		}
		if weightedSum > 0 {
			if mustRepeatWeight/weightedSum >= 0.60 && notWorthWeight/weightedSum >= 0.40 {
				qualityConflict = true
				note := "Loved for the vibe — mixed on the food"
				conflictNote = &note
			}
		}
	}

	display := DisplayState(stampCount)
	params := dbgen.UpsertVenueStampAggregateParams{
		PlaceID:          placeID,
		StampCount:       int32(stampCount),
		WeightedStampSum: numericFromFloat(weightedSum),
		PairwiseCount:    pairCount,
		DisplayState:     display,
		VibeConflict:     vibeConflict,
		QualityConflict:  qualityConflict,
		VelocityScore:    numericFromFloat(0),
	}
	if sentiment != nil {
		params.SentimentScore = numericFromFloat(*sentiment)
	}
	if quality != nil {
		params.QualityScore = numericFromFloat(*quality)
	}
	if pairwiseStrength != nil {
		params.PairwiseStrength = numericFromFloat(*pairwiseStrength)
	}
	if composite != nil {
		params.CompositeScore = numericFromFloat(*composite)
	}
	if final != nil {
		params.FinalScore = numericFromFloat(*final)
	}
	if conflictNote != nil {
		params.ConflictNote = conflictNote
	}

	if err := q.UpsertVenueStampAggregate(ctx, params); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to upsert venue aggregate")
	}
	return nil
}

func (s *Service) loadStampSummary(ctx context.Context, placeID string) (*StampSummary, error) {
	placeUUID, err := uuidFromString(placeID)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid place id"}
	}
	q := dbgen.New(s.db)
	agg, err := q.GetVenueStampAggregate(ctx, placeUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &StampSummary{
				StampCount:   0,
				DisplayState: "new",
				Companions:   []TagAggregate{},
				Vibes:        []TagAggregate{},
			}, nil
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to load stamp aggregate")
	}
	tags, err := q.ListVenueTagAggregates(ctx, placeUUID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to load tag aggregates")
	}
	summary := &StampSummary{
		StampCount:      int(agg.StampCount),
		DisplayState:    agg.DisplayState,
		VibeConflict:    agg.VibeConflict,
		QualityConflict: agg.QualityConflict,
		ConflictNote:    agg.ConflictNote,
		Companions:      []TagAggregate{},
		Vibes:           []TagAggregate{},
	}
	if agg.FinalScore.Valid {
		f := numericToFloat(agg.FinalScore)
		summary.FinalScore = &f
	}
	if agg.SentimentScore.Valid {
		f := numericToFloat(agg.SentimentScore)
		summary.SentimentScore = &f
	}
	if agg.QualityScore.Valid {
		f := numericToFloat(agg.QualityScore)
		summary.QualityScore = &f
	}
	for _, tag := range tags {
		item := TagAggregate{
			Slug:       tag.TagSlug,
			Percentage: numericToFloat(tag.Percentage),
			RawCount:   int(tag.RawCount),
		}
		switch tag.TagType {
		case "companion":
			summary.Companions = append(summary.Companions, item)
		case "vibe":
			// early display: only show vibes with >= 3 raw
			if summary.DisplayState == "early" && item.RawCount < 3 {
				continue
			}
			summary.Vibes = append(summary.Vibes, item)
		}
	}
	_ = strings.TrimSpace
	return summary, nil
}

type GetStampSummaryRequest struct {
	PlaceID string `json:"place_id"`
}

//encore:api private
func (s *Service) GetStampSummary(ctx context.Context, req *GetStampSummaryRequest) (*StampSummary, error) {
	if req == nil || strings.TrimSpace(req.PlaceID) == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "place_id is required"}
	}
	return s.loadStampSummary(ctx, strings.TrimSpace(req.PlaceID))
}
