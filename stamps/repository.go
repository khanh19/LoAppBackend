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

func (s *Service) createStamp(ctx context.Context, userID string, req *CreateStampRequest) (*CreateStampResponse, error) {
	if err := validateCreateStamp(req); err != nil {
		return nil, err
	}

	userUUID, err := uuidFromString(userID)
	if err != nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "invalid user"}
	}
	placeUUID, err := uuidFromString(req.PlaceID)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid place_id"}
	}

	q := dbgen.New(s.db)
	place, err := q.GetStampPlaceContext(ctx, placeUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "place not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to load place")
	}
	if place.VenueCategory == nil || strings.TrimSpace(*place.VenueCategory) == "" {
		return nil, &errs.Error{Code: errs.FailedPrecondition, Message: "place has no venue_category; cannot stamp yet"}
	}
	category := *place.VenueCategory

	visitCount := 1
	existing, err := q.GetActiveStampByUserPlace(ctx, dbgen.GetActiveStampByUserPlaceParams{
		UserID:  userUUID,
		PlaceID: placeUUID,
	})
	if err == nil {
		visitCount = int(existing.VisitCount) + 1
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.WrapCode(err, errs.Internal, "failed to load existing stamp")
	}

	level, weight := ResolveVerification(
		visitCount,
		req.Latitude,
		req.Longitude,
		numericToFloat(place.Latitude),
		numericToFloat(place.Longitude),
		200,
	)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to begin transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := dbgen.New(tx)

	stamp, err := tq.UpsertStamp(ctx, dbgen.UpsertStampParams{
		UserID:            userUUID,
		PlaceID:           placeUUID,
		VenueCategory:     category,
		Verdict:           req.Verdict,
		Quality:           req.Quality,
		Note:              strings.TrimSpace(req.Note),
		VerificationLevel: level,
		Weight:            numericFromFloat(weight),
		DeviceLatitude:    numericFromOptionalFloat(req.Latitude),
		DeviceLongitude:   numericFromOptionalFloat(req.Longitude),
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to upsert stamp")
	}
	stampUUID, err := uuidFromString(stamp.ID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "invalid stamp id")
	}

	if err := tq.DeleteStampTags(ctx, stampUUID); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to clear stamp tags")
	}
	for i, slug := range req.Companions {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		if err := tq.InsertStampTag(ctx, dbgen.InsertStampTagParams{
			StampID:  stampUUID,
			TagType:  "companion",
			TagSlug:  slug,
			Position: int32(i),
			IsCustom: strings.HasPrefix(slug, "custom-"),
		}); err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to insert companion tag")
		}
	}
	for i, slug := range req.Vibes {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		if err := tq.InsertStampTag(ctx, dbgen.InsertStampTagParams{
			StampID:  stampUUID,
			TagType:  "vibe",
			TagSlug:  slug,
			Position: int32(i),
			IsCustom: strings.HasPrefix(slug, "custom-"),
		}); err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to insert vibe tag")
		}
	}

	band := req.Verdict
	bandBounds, ok := BandForVerdict(band)
	if !ok {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid verdict"}
	}

	poolRows, err := tq.ListUserPoolBand(ctx, dbgen.ListUserPoolBandParams{
		UserID:         userUUID,
		VenueCategory:  category,
		Band:           band,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to load pool band")
	}

	orderedIDs := make([]string, 0, len(poolRows))
	comps := map[string]int{}
	for _, row := range poolRows {
		if row.PlaceID == req.PlaceID {
			continue
		}
		orderedIDs = append(orderedIDs, row.PlaceID)
		comps[row.PlaceID] = int(row.ComparisonsCount)
	}

	// Provisional placement at band midpoint until pairwise resolves.
	insertPos := len(orderedIDs)/2 + 1
	orderedWithNew := OrderedInsert(orderedIDs, req.PlaceID, insertPos)
	comps[req.PlaceID] = 0

	eBand, err := sumBandComparability(ctx, tq, userUUID, category, band)
	if err != nil {
		return nil, err
	}
	scores := RecomputeBandScores(bandBounds, orderedWithNew, eBand, comps, defaultPlacement)

	for placeID, sc := range scores {
		pid, err := uuidFromString(placeID)
		if err != nil {
			return nil, err
		}
		isProv := placeID != req.PlaceID && false
		entry, getErr := tq.GetUserPoolEntry(ctx, dbgen.GetUserPoolEntryParams{
			UserID:  userUUID,
			PlaceID: pid,
		})
		contradiction := 0
		if getErr == nil {
			contradiction = int(entry.ContradictionCount)
			isProv = entry.IsProvisional && placeID != req.PlaceID
		}
		if _, err := tq.UpsertUserPoolEntry(ctx, dbgen.UpsertUserPoolEntryParams{
			UserID:              userUUID,
			PlaceID:             pid,
			VenueCategory:       category,
			Band:                band,
			RankKey:             numericFromFloat(sc.RankKey),
			Score:               numericFromFloat(sc.Score),
			PlacementConfidence: numericFromFloat(sc.PlacementConfidence),
			ComparisonsCount:    int32(sc.ComparisonsCount),
			ContradictionCount:  int32(contradiction),
			IsProvisional:       isProv,
		}); err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to upsert pool entry")
		}
	}

	if err := recomputeVenueAggregateTx(ctx, tq, placeUUID, category); err != nil {
		return nil, err
	}

	lo, hi := 1, len(orderedIDs)+1
	session, err := tq.CreatePlacementSession(ctx, dbgen.CreatePlacementSessionParams{
		UserID:          userUUID,
		StampID:         stampUUID,
		PlaceID:         placeUUID,
		VenueCategory:   category,
		Band:            band,
		LoIndex:         int32(lo),
		HiIndex:         int32(hi),
		ComparisonsDone: 0,
		IsComplete:      len(orderedIDs) == 0,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create placement session")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to commit stamp")
	}

	personal := scores[req.PlaceID].Score
	candidates, err := s.selectPairwiseCandidates(ctx, s.db, userID, req.PlaceID, category, band, place, orderedIDs, lo, hi)
	if err != nil {
		return nil, err
	}

	resp := &CreateStampResponse{
		StampID:       stamp.ID,
		PersonalScore: personal,
		Band:          band,
		SkipPairwise:  len(candidates) == 0,
		Candidates:    candidates,
	}
	if !resp.SkipPairwise {
		resp.PlacementSession = &session.ID
	}
	return resp, nil
}

func sumBandComparability(ctx context.Context, q *dbgen.Queries, userID pgtype.UUID, category, band string) (float64, error) {
	total, err := q.SumBandComparability(ctx, dbgen.SumBandComparabilityParams{
		UserID:        userID,
		VenueCategory: category,
		Band:          band,
	})
	if err != nil {
		return 0, errs.WrapCode(err, errs.Internal, "failed to sum band comparability")
	}
	return total, nil
}

func validateCreateStamp(req *CreateStampRequest) error {
	if req == nil {
		return &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	req.PlaceID = strings.TrimSpace(req.PlaceID)
	req.Verdict = strings.TrimSpace(req.Verdict)
	req.Quality = strings.TrimSpace(req.Quality)
	if req.PlaceID == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "place_id is required"}
	}
	if _, ok := BandForVerdict(req.Verdict); !ok {
		return &errs.Error{Code: errs.InvalidArgument, Message: "invalid verdict"}
	}
	if _, ok := defaultQualityValues[req.Quality]; !ok {
		return &errs.Error{Code: errs.InvalidArgument, Message: "invalid quality"}
	}
	if len(req.Companions) > 3 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "companions max is 3"}
	}
	if len(req.Note) > 100 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "note max is 100 characters"}
	}
	return nil
}

func (s *Service) getPool(ctx context.Context, userID, category string) (*GetPoolResponse, error) {
	userUUID, err := uuidFromString(userID)
	if err != nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "invalid user"}
	}
	category = strings.TrimSpace(category)
	rows, err := dbgen.New(s.db).ListUserPoolByCategory(ctx, dbgen.ListUserPoolByCategoryParams{
		UserID:        userUUID,
		VenueCategory: category,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list pool")
	}
	entries := make([]PoolEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, PoolEntry{
			PlaceID:             row.PlaceID,
			Name:                row.PlaceName,
			Band:                row.Band,
			Score:               numericToFloat(row.Score),
			PlacementConfidence: numericToFloat(row.PlacementConfidence),
			IsProvisional:       row.IsProvisional,
			Neighborhood:        row.Neighborhood,
			CoverImageURL:       row.CoverImageUrl,
			PriceLevel:          row.PriceLevel,
		})
	}
	return &GetPoolResponse{Category: category, Entries: entries}, nil
}

func (s *Service) setFamiliarity(ctx context.Context, userID string, req *FamiliarityRequest) (*FamiliarityResponse, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	status := strings.TrimSpace(req.Status)
	if status != "been" && status != "not_been" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "status must be been or not_been"}
	}
	userUUID, err := uuidFromString(userID)
	if err != nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "invalid user"}
	}
	placeUUID, err := uuidFromString(strings.TrimSpace(req.PlaceID))
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid place_id"}
	}
	if err := dbgen.New(s.db).UpsertFamiliarity(ctx, dbgen.UpsertFamiliarityParams{
		UserID:  userUUID,
		PlaceID: placeUUID,
		Status:  status,
	}); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to save familiarity")
	}
	return &FamiliarityResponse{PlaceID: req.PlaceID, Status: status}, nil
}

// ensure pool helper available for pairwise without unused import of pgxpool in this file
var _ *pgxpool.Pool
