package stamps

import (
	"context"
	"errors"
	"strings"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (s *Service) selectPairwiseCandidates(
	ctx context.Context,
	db *pgxpool.Pool,
	userID, placeID, category, band string,
	place dbgen.GetStampPlaceContextRow,
	sameBandIDs []string,
	lo, hi int,
) ([]PairwiseCandidate, error) {
	userUUID, err := uuidFromString(userID)
	if err != nil {
		return nil, err
	}
	q := dbgen.New(db)

	// Tier 1: stamped same band
	if len(sameBandIDs) > 0 {
		cands, err := s.scorePoolCandidates(ctx, q, userUUID, place, category, band, sameBandIDs, lo, hi, 1)
		if err != nil {
			return nil, err
		}
		if len(cands) > 0 {
			return cands[:min(3, len(cands))], nil
		}
	}

	// Tier 2: adjacent band (calibration only — still return for optional UI, but marked tier 2)
	for _, adj := range AdjacentBands(band) {
		rows, err := q.ListUserPoolBand(ctx, dbgen.ListUserPoolBandParams{
			UserID:        userUUID,
			VenueCategory: category,
			Band:          adj,
		})
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to list adjacent band")
		}
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.PlaceID)
		}
		if len(ids) == 0 {
			continue
		}
		cands, err := s.scorePoolCandidates(ctx, q, userUUID, place, category, band, ids, 1, len(ids)+1, 2)
		if err != nil {
			return nil, err
		}
		if len(cands) > 0 {
			return cands[:min(2, len(cands))], nil
		}
	}

	// Tier 3: onboarding likes
	liked, err := q.ListLikedPlacesForPairwise(ctx, dbgen.ListLikedPlacesForPairwiseParams{
		UserID:         userUUID,
		VenueCategory:  &category,
		CityID:         mustUUID(place.CityID),
		ExcludePlaceID: mustUUID(placeID),
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list liked places")
	}
	if len(liked) > 0 {
		cands := make([]PairwiseCandidate, 0, len(liked))
		for _, row := range liked {
			if !passesHardGates(place, row.VenueArchetype, row.PriceLevel, row.VenueCategory) {
				continue
			}
			in := ComparabilityInput{
				BisectionScore:   0.5,
				SameNeighborhood: eqPtr(place.Neighborhood, row.Neighborhood),
				SameCity:         true,
				PriceA:           place.PriceLevel,
				PriceB:           row.PriceLevel,
				VibesA:           place.Tags,
				VibesB:           row.Tags,
				RecencyScore:     0.5,
			}
			score := ScoreComparability(in, defaultComparability)
			arch := ""
			if row.VenueArchetype != nil {
				arch = *row.VenueArchetype
			}
			cands = append(cands, PairwiseCandidate{
				PlaceID:       row.ID,
				Name:          row.Name,
				Neighborhood:  row.Neighborhood,
				CoverImageURL: row.CoverImageUrl,
				PriceLevel:    row.PriceLevel,
				Archetype:     row.VenueArchetype,
				Comparability: score,
				Tier:          3,
				Reasons:       BuildComparabilityReasons(in, arch),
			})
		}
		sortCandidates(cands)
		if len(cands) > 0 {
			return cands[:min(3, len(cands))], nil
		}
	}

	// Tier 4: curated been-here probe
	probes, err := q.ListCuratedPlacesForProbe(ctx, dbgen.ListCuratedPlacesForProbeParams{
		VenueCategory:  &category,
		CityID:         mustUUID(place.CityID),
		ExcludePlaceID: mustUUID(placeID),
		UserID:         userUUID,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list probe places")
	}
	cands := make([]PairwiseCandidate, 0, len(probes))
	for _, row := range probes {
		if !passesHardGates(place, row.VenueArchetype, row.PriceLevel, row.VenueCategory) {
			continue
		}
		in := ComparabilityInput{
			BisectionScore:   0.5,
			SameNeighborhood: eqPtr(place.Neighborhood, row.Neighborhood),
			SameCity:         true,
			PriceA:           place.PriceLevel,
			PriceB:           row.PriceLevel,
			VibesA:           place.Tags,
			VibesB:           row.Tags,
			RecencyScore:     0.3,
		}
		score := ScoreComparability(in, defaultComparability)
		arch := ""
		if row.VenueArchetype != nil {
			arch = *row.VenueArchetype
		}
		cands = append(cands, PairwiseCandidate{
			PlaceID:       row.ID,
			Name:          row.Name,
			Neighborhood:  row.Neighborhood,
			CoverImageURL: row.CoverImageUrl,
			PriceLevel:    row.PriceLevel,
			Archetype:     row.VenueArchetype,
			Comparability: score,
			Tier:          4,
			Reasons:       BuildComparabilityReasons(in, arch),
			IsProbe:       true,
		})
	}
	sortCandidates(cands)
	if len(cands) > 0 {
		return cands[:min(2, len(cands))], nil
	}

	// Tier 5: empty
	return nil, nil
}

func (s *Service) scorePoolCandidates(
	ctx context.Context,
	q *dbgen.Queries,
	userUUID pgtype.UUID,
	place dbgen.GetStampPlaceContextRow,
	category, band string,
	candidateIDs []string,
	lo, hi, tier int,
) ([]PairwiseCandidate, error) {
	mid := MidIndex(lo, hi)
	cands := make([]PairwiseCandidate, 0, len(candidateIDs))
	for idx, id := range candidateIDs {
		pid, err := uuidFromString(id)
		if err != nil {
			continue
		}
		row, err := q.GetStampPlaceContext(ctx, pid)
		if err != nil {
			continue
		}
		if !passesHardGates(place, row.VenueArchetype, row.PriceLevel, row.VenueCategory) {
			continue
		}
		// skip recent pairs
		count, err := q.CountRecentPairwisePair(ctx, dbgen.CountRecentPairwisePairParams{
			UserID:   userUUID,
			PlaceAID: mustUUID(place.ID),
			PlaceBID: pid,
		})
		if err == nil && count > 0 && tier == 1 {
			continue
		}
		bisection := 1.0
		if len(candidateIDs) > 1 {
			dist := idx + 1 - mid
			if dist < 0 {
				dist = -dist
			}
			bisection = 1.0 - float64(dist)/float64(len(candidateIDs))
			if bisection < 0 {
				bisection = 0
			}
		}
		in := ComparabilityInput{
			BisectionScore:   bisection,
			SameNeighborhood: eqPtr(place.Neighborhood, row.Neighborhood),
			SameCity:         place.CityID == row.CityID,
			PriceA:           place.PriceLevel,
			PriceB:           row.PriceLevel,
			VibesA:           place.Tags,
			VibesB:           row.Tags,
			RecencyScore:     0.8,
		}
		score := ScoreComparability(in, defaultComparability)
		arch := ""
		if row.VenueArchetype != nil {
			arch = *row.VenueArchetype
		}
		cands = append(cands, PairwiseCandidate{
			PlaceID:       row.ID,
			Name:          row.Name,
			Neighborhood:  row.Neighborhood,
			CoverImageURL: row.CoverImageUrl,
			PriceLevel:    row.PriceLevel,
			Archetype:     row.VenueArchetype,
			Comparability: score,
			Tier:          tier,
			Reasons:       BuildComparabilityReasons(in, arch),
		})
		_ = category
		_ = band
	}
	sortCandidates(cands)
	return cands, nil
}

func passesHardGates(place dbgen.GetStampPlaceContextRow, candArchetype *string, candPrice *int16, candCategory *string) bool {
	if candCategory == nil || place.VenueCategory == nil || *candCategory != *place.VenueCategory {
		return false
	}
	// Strict archetype: both must be set and equal
	if !ArchetypesCompatible(place.VenueArchetype, candArchetype) {
		// Allow only when both missing AND price within one — still strict for mismatch
		if place.VenueArchetype != nil && candArchetype != nil {
			return false
		}
		if place.VenueArchetype == nil || candArchetype == nil {
			if !PriceWithinOne(place.PriceLevel, candPrice) {
				return false
			}
			// if either price missing and archetypes missing, skip
			if place.PriceLevel == nil || candPrice == nil {
				return false
			}
			d := int(*place.PriceLevel) - int(*candPrice)
			if d < 0 {
				d = -d
			}
			if d > 1 {
				return false
			}
		}
	}
	if !PriceWithinOne(place.PriceLevel, candPrice) {
		return false
	}
	return true
}

func (s *Service) submitPairwise(ctx context.Context, userID, stampID string, req *SubmitPairwiseRequest) (*SubmitPairwiseResponse, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	winner := strings.TrimSpace(req.Winner)
	if winner != "this" && winner != "other" && winner != "tie" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "winner must be this, other, or tie"}
	}

	userUUID, err := uuidFromString(userID)
	if err != nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "invalid user"}
	}
	stampUUID, err := uuidFromString(stampID)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid stamp id"}
	}
	oppUUID, err := uuidFromString(strings.TrimSpace(req.OpponentPlaceID))
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid opponent_place_id"}
	}

	q := dbgen.New(s.db)
	stamp, err := q.GetStampByID(ctx, stampUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "stamp not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to load stamp")
	}
	if stamp.UserID != userID {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "stamp does not belong to user"}
	}

	placeUUID, err := uuidFromString(stamp.PlaceID)
	if err != nil {
		return nil, err
	}
	place, err := q.GetStampPlaceContext(ctx, placeUUID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to load place")
	}
	opp, err := q.GetStampPlaceContext(ctx, oppUUID)
	if err != nil {
		return nil, &errs.Error{Code: errs.NotFound, Message: "opponent place not found"}
	}

	session, err := q.GetActivePlacementSession(ctx, stampUUID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.WrapCode(err, errs.Internal, "failed to load placement session")
	}
	hasSession := err == nil

	in := ComparabilityInput{
		BisectionScore:   0.8,
		SameNeighborhood: eqPtr(place.Neighborhood, opp.Neighborhood),
		SameCity:         place.CityID == opp.CityID,
		PriceA:           place.PriceLevel,
		PriceB:           opp.PriceLevel,
		VibesA:           place.Tags,
		VibesB:           opp.Tags,
		RecencyScore:     0.8,
	}
	comparability := ScoreComparability(in, defaultComparability)

	var winnerID *string
	switch winner {
	case "this":
		winnerID = &stamp.PlaceID
	case "other":
		winnerID = &req.OpponentPlaceID
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "begin tx")
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := dbgen.New(tx)

	var winnerUUID pgtype.UUID
	if winnerID != nil {
		winnerUUID, err = uuidFromString(*winnerID)
		if err != nil {
			return nil, err
		}
	}

	tier := int32(1)
	surface := "stamp_flow"
	if hasSession {
		// detect if opponent is adjacent band → tier 2 calibration
		oppEntry, e := tq.GetUserPoolEntry(ctx, dbgen.GetUserPoolEntryParams{
			UserID:  userUUID,
			PlaceID: oppUUID,
		})
		if e == nil && oppEntry.Band != stamp.Verdict {
			tier = 2
			surface = "calibration"
		}
	}

	stampIDArg := stampUUID
	_, err = tq.InsertPairwiseComparison(ctx, dbgen.InsertPairwiseComparisonParams{
		UserID:         userUUID,
		StampID:        stampIDArg,
		PlaceAID:       placeUUID,
		PlaceBID:       oppUUID,
		WinnerPlaceID:  winnerUUID,
		VenueCategory:  stamp.VenueCategory,
		Band:           stamp.Verdict,
		Tier:           tier,
		Comparability:  numericFromFloat(comparability),
		Surface:        surface,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to insert pairwise")
	}

	// Bootstrap provisional pool entry for opponent if missing (tier 3/4)
	_, poolErr := tq.GetUserPoolEntry(ctx, dbgen.GetUserPoolEntryParams{
		UserID:  userUUID,
		PlaceID: oppUUID,
	})
	if errors.Is(poolErr, pgx.ErrNoRows) && tier != 2 {
		bandBounds, _ := BandForVerdict(stamp.Verdict)
		_, err = tq.UpsertUserPoolEntry(ctx, dbgen.UpsertUserPoolEntryParams{
			UserID:              userUUID,
			PlaceID:             oppUUID,
			VenueCategory:       stamp.VenueCategory,
			Band:                stamp.Verdict,
			RankKey:             numericFromFloat(1),
			Score:               numericFromFloat((bandBounds.Lo + bandBounds.Hi) / 2),
			PlacementConfidence: numericFromFloat(0),
			ComparisonsCount:    1,
			ContradictionCount:  0,
			IsProvisional:       true,
		})
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to bootstrap opponent")
		}
	}

	personalScore := 0.0
	confidence := 0.0
	complete := winner == "tie" || tier == 2

	if hasSession && tier == 1 && winner != "tie" {
		lo := int(session.LoIndex)
		hi := int(session.HiIndex)
		mid := MidIndex(lo, hi)

		bandRows, err := tq.ListUserPoolBand(ctx, dbgen.ListUserPoolBandParams{
			UserID:        userUUID,
			VenueCategory: stamp.VenueCategory,
			Band:          stamp.Verdict,
		})
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to list band")
		}
		// Build ordered list excluding the stamp place
		ordered := make([]string, 0, len(bandRows))
		comps := map[string]int{}
		for _, row := range bandRows {
			comps[row.PlaceID] = int(row.ComparisonsCount)
			if row.PlaceID == stamp.PlaceID {
				continue
			}
			ordered = append(ordered, row.PlaceID)
		}
		// Ensure opponent is in ordered list for mid lookup
		if mid >= 1 && mid <= len(ordered) {
			preferNew := winner == "this"
			lo, hi = NarrowInterval(lo, hi, mid, preferNew)
		} else {
			// opponent may not be at mid if list changed; treat prefer against opponent index
			oppIdx := indexOf(ordered, req.OpponentPlaceID)
			if oppIdx >= 0 {
				preferNew := winner == "this"
				lo, hi = NarrowInterval(lo, hi, oppIdx+1, preferNew)
			} else {
				preferNew := winner == "this"
				if preferNew {
					lo = hi
				} else {
					hi = lo
				}
			}
		}

		comps[stamp.PlaceID] = comps[stamp.PlaceID] + 1
		comps[req.OpponentPlaceID] = comps[req.OpponentPlaceID] + 1

		resolved := false
		insertPos := 0
		if lo >= hi {
			insertPos = lo
			resolved = true
			complete = true
		} else {
			// Place at midpoint of remaining interval for this flow step
			insertPos = MidIndex(lo, hi)
			if insertPos < 1 {
				insertPos = 1
			}
			if insertPos > len(ordered)+1 {
				insertPos = len(ordered) + 1
			}
		}

		finalOrder := OrderedInsert(ordered, stamp.PlaceID, insertPos)
		bandBounds, _ := BandForVerdict(stamp.Verdict)
		eBand, err := sumBandComparability(ctx, tq, userUUID, stamp.VenueCategory, stamp.Verdict)
		if err != nil {
			return nil, err
		}
		scores := RecomputeBandScores(bandBounds, finalOrder, eBand, comps, defaultPlacement)
		for pid, sc := range scores {
			id, err := uuidFromString(pid)
			if err != nil {
				return nil, err
			}
			entry, e := tq.GetUserPoolEntry(ctx, dbgen.GetUserPoolEntryParams{UserID: userUUID, PlaceID: id})
			contradiction := 0
			isProv := false
			if e == nil {
				contradiction = int(entry.ContradictionCount)
				isProv = entry.IsProvisional && pid != stamp.PlaceID
			}
			if _, err := tq.UpsertUserPoolEntry(ctx, dbgen.UpsertUserPoolEntryParams{
				UserID:              userUUID,
				PlaceID:             id,
				VenueCategory:       stamp.VenueCategory,
				Band:                stamp.Verdict,
				RankKey:             numericFromFloat(sc.RankKey),
				Score:               numericFromFloat(sc.Score),
				PlacementConfidence: numericFromFloat(sc.PlacementConfidence),
				ComparisonsCount:    int32(sc.ComparisonsCount),
				ContradictionCount:  int32(contradiction),
				IsProvisional:       isProv,
			}); err != nil {
				return nil, errs.WrapCode(err, errs.Internal, "failed to update pool scores")
			}
		}
		personalScore = scores[stamp.PlaceID].Score
		confidence = scores[stamp.PlaceID].PlacementConfidence

		sessionUUID, err := uuidFromString(session.ID)
		if err != nil {
			return nil, err
		}
		_, err = tq.UpdatePlacementSession(ctx, dbgen.UpdatePlacementSessionParams{
			ID:              sessionUUID,
			LoIndex:         int32(lo),
			HiIndex:         int32(hi),
			ComparisonsDone: session.ComparisonsDone + 1,
			IsComplete:      complete || resolved,
		})
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to update session")
		}
	} else {
		entry, e := tq.GetUserPoolEntry(ctx, dbgen.GetUserPoolEntryParams{
			UserID:  userUUID,
			PlaceID: placeUUID,
		})
		if e == nil {
			personalScore = numericToFloat(entry.Score)
			confidence = numericToFloat(entry.PlacementConfidence)
		}
		complete = true
	}

	if err := recomputeVenueAggregateTx(ctx, tq, placeUUID, stamp.VenueCategory); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "commit pairwise")
	}

	return &SubmitPairwiseResponse{
		PersonalScore:       personalScore,
		PlacementConfidence: confidence,
		IsComplete:          complete,
	}, nil
}

func (s *Service) nextPairwise(ctx context.Context, userID string) (*NextPairwiseResponse, error) {
	userUUID, err := uuidFromString(userID)
	if err != nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "invalid user"}
	}
	q := dbgen.New(s.db)
	rows, err := q.ListLowConfidencePoolEntries(ctx, dbgen.ListLowConfidencePoolEntriesParams{
		UserID:   userUUID,
		LimitVal: 1,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list low confidence")
	}
	if len(rows) == 0 {
		return &NextPairwiseResponse{Skip: true}, nil
	}
	row := rows[0]
	placeUUID, err := uuidFromString(row.PlaceID)
	if err != nil {
		return nil, err
	}
	place, err := q.GetStampPlaceContext(ctx, placeUUID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to load place")
	}
	bandRows, err := q.ListUserPoolBand(ctx, dbgen.ListUserPoolBandParams{
		UserID:        userUUID,
		VenueCategory: row.VenueCategory,
		Band:          row.Band,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(bandRows))
	for _, r := range bandRows {
		if r.PlaceID != row.PlaceID {
			ids = append(ids, r.PlaceID)
		}
	}
	cands, err := s.scorePoolCandidates(ctx, q, userUUID, place, row.VenueCategory, row.Band, ids, 1, len(ids)+1, 1)
	if err != nil {
		return nil, err
	}
	if len(cands) == 0 {
		return &NextPairwiseResponse{Skip: true}, nil
	}
	return &NextPairwiseResponse{
		Skip:       false,
		PlaceID:    row.PlaceID,
		PlaceName:  row.PlaceName,
		Band:       row.Band,
		Candidates: cands[:min(2, len(cands))],
	}, nil
}

func (s *Service) submitStandalonePairwise(ctx context.Context, userID string, req *StandalonePairwiseRequest) (*SubmitPairwiseResponse, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	// Reuse stamp pairwise when stamp_id provided; otherwise apply edge + recompute band.
	if strings.TrimSpace(req.StampID) != "" {
		return s.submitPairwise(ctx, userID, req.StampID, &SubmitPairwiseRequest{
			OpponentPlaceID: req.OpponentPlaceID,
			Winner:          req.Winner,
		})
	}

	userUUID, err := uuidFromString(userID)
	if err != nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "invalid user"}
	}
	placeUUID, err := uuidFromString(strings.TrimSpace(req.PlaceID))
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid place_id"}
	}
	oppUUID, err := uuidFromString(strings.TrimSpace(req.OpponentPlaceID))
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid opponent"}
	}
	winner := strings.TrimSpace(req.Winner)
	if winner != "this" && winner != "other" && winner != "tie" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid winner"}
	}

	q := dbgen.New(s.db)
	entry, err := q.GetUserPoolEntry(ctx, dbgen.GetUserPoolEntryParams{
		UserID:  userUUID,
		PlaceID: placeUUID,
	})
	if err != nil {
		return nil, &errs.Error{Code: errs.NotFound, Message: "place not in your pool"}
	}

	var winnerUUID pgtype.UUID
	if winner == "this" {
		winnerUUID = placeUUID
	} else if winner == "other" {
		winnerUUID = oppUUID
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "begin")
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := dbgen.New(tx)

	_, err = tq.InsertPairwiseComparison(ctx, dbgen.InsertPairwiseComparisonParams{
		UserID:        userUUID,
		PlaceAID:      placeUUID,
		PlaceBID:      oppUUID,
		WinnerPlaceID: winnerUUID,
		VenueCategory: entry.VenueCategory,
		Band:          entry.Band,
		Tier:          1,
		Comparability: numericFromFloat(0.75),
		Surface:       "standalone",
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "insert pairwise")
	}

	bandRows, err := tq.ListUserPoolBand(ctx, dbgen.ListUserPoolBandParams{
		UserID:        userUUID,
		VenueCategory: entry.VenueCategory,
		Band:          entry.Band,
	})
	if err != nil {
		return nil, err
	}
	ordered := make([]string, 0, len(bandRows))
	comps := map[string]int{}
	for _, r := range bandRows {
		comps[r.PlaceID] = int(r.ComparisonsCount)
		ordered = append(ordered, r.PlaceID)
	}
	comps[req.PlaceID]++
	comps[req.OpponentPlaceID]++

	// Simple reorder: if prefer this over other, ensure place is above opponent
	if winner != "tie" {
		pi := indexOf(ordered, req.PlaceID)
		oi := indexOf(ordered, req.OpponentPlaceID)
		if pi >= 0 && oi >= 0 {
			if winner == "this" && pi < oi {
				// already above (higher index = better); worst is index 0
				// our order is worst→best, so "above" means higher index
			}
			if winner == "this" && pi < oi {
				// place is worse than opponent; swap upward
				ordered = moveAfter(ordered, pi, oi)
			}
			if winner == "other" && pi > oi {
				ordered = moveAfter(ordered, oi, pi)
			}
		}
	}

	bandBounds, _ := BandForVerdict(entry.Band)
	eBand, err := sumBandComparability(ctx, tq, userUUID, entry.VenueCategory, entry.Band)
	if err != nil {
		return nil, err
	}
	scores := RecomputeBandScores(bandBounds, ordered, eBand, comps, defaultPlacement)
	for pid, sc := range scores {
		id, err := uuidFromString(pid)
		if err != nil {
			return nil, err
		}
		e, _ := tq.GetUserPoolEntry(ctx, dbgen.GetUserPoolEntryParams{UserID: userUUID, PlaceID: id})
		contradiction := 0
		isProv := false
		if e.PlaceID != "" {
			contradiction = int(e.ContradictionCount)
			isProv = e.IsProvisional
		}
		if _, err := tq.UpsertUserPoolEntry(ctx, dbgen.UpsertUserPoolEntryParams{
			UserID:              userUUID,
			PlaceID:             id,
			VenueCategory:       entry.VenueCategory,
			Band:                entry.Band,
			RankKey:             numericFromFloat(sc.RankKey),
			Score:               numericFromFloat(sc.Score),
			PlacementConfidence: numericFromFloat(sc.PlacementConfidence),
			ComparisonsCount:    int32(sc.ComparisonsCount),
			ContradictionCount:  int32(contradiction),
			IsProvisional:       isProv,
		}); err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "update scores")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "commit")
	}

	return &SubmitPairwiseResponse{
		PersonalScore:       scores[req.PlaceID].Score,
		PlacementConfidence: scores[req.PlaceID].PlacementConfidence,
		IsComplete:          true,
	}, nil
}

func sortCandidates(cands []PairwiseCandidate) {
	for i := 0; i < len(cands); i++ {
		for j := i + 1; j < len(cands); j++ {
			if cands[j].Comparability > cands[i].Comparability {
				cands[i], cands[j] = cands[j], cands[i]
			}
		}
	}
}

func eqPtr(a, b *string) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(*a), strings.TrimSpace(*b))
}

func mustUUID(s string) pgtype.UUID {
	id, err := uuidFromString(s)
	if err != nil {
		return pgtype.UUID{}
	}
	return id
}

func indexOf(xs []string, v string) int {
	for i, x := range xs {
		if x == v {
			return i
		}
	}
	return -1
}

func moveAfter(xs []string, from, after int) []string {
	if from < 0 || after < 0 || from >= len(xs) || after >= len(xs) {
		return xs
	}
	item := xs[from]
	out := append([]string{}, xs[:from]...)
	out = append(out, xs[from+1:]...)
	// adjust after index if needed
	if from < after {
		after--
	}
	insertAt := after + 1
	if insertAt > len(out) {
		insertAt = len(out)
	}
	res := append([]string{}, out[:insertAt]...)
	res = append(res, item)
	res = append(res, out[insertAt:]...)
	return res
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
