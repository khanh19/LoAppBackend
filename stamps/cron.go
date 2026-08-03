package stamps

import (
	"context"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5/pgtype"
)

// Cron disabled on Encore free tier: Cloud Scheduler job quota is exhausted
// (JOBS-per-project-per-region in us-east4). Venue aggregates still update on
// stamp/pairwise writes; this job only does nightly drift recompute + history.
//
// Re-add once the app has its own GCP / paid Encore cloud (or Scheduler quota):
//
//	import "encore.dev/cron"
//
//	var _ = cron.NewJob("stamps-nightly-recompute", cron.JobConfig{
//		Title:    "Recompute venue stamp aggregates and snapshot scores",
//		Every:    24 * cron.Hour,
//		Endpoint: NightlyRecompute,
//	})

//encore:api private
func (s *Service) NightlyRecompute(ctx context.Context) error {
	q := dbgen.New(s.db)
	placeIDs, err := q.ListAllActivePlaceIDsWithStamps(ctx)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to list stamped places")
	}
	for _, id := range placeIDs {
		placeUUID, err := uuidFromString(id)
		if err != nil {
			continue
		}
		place, err := q.GetStampPlaceContext(ctx, placeUUID)
		if err != nil {
			continue
		}
		category := "other"
		if place.VenueCategory != nil && *place.VenueCategory != "" {
			category = *place.VenueCategory
		}
		if err := recomputeVenueAggregateTx(ctx, q, placeUUID, category); err != nil {
			return err
		}
	}
	if err := q.InsertVenueScoreHistory(ctx); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to snapshot score history")
	}
	if err := s.renormalizeRankKeys(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Service) renormalizeRankKeys(ctx context.Context) error {
	// Soft renormalization: for each user/category/band, rewrite rank_key to 1..n.
	// Done via listing low-confidence is incomplete for all users; instead scan via
	// distinct pool entries using raw SQL through the pool.
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT user_id::text, venue_category, band
		FROM user_pool_entries
	`)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to list pools for renormalize")
	}
	defer rows.Close()

	type key struct{ userID, category, band string }
	var keys []key
	for rows.Next() {
		var k key
		if err := rows.Scan(&k.userID, &k.category, &k.band); err != nil {
			return err
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	q := dbgen.New(s.db)
	for _, k := range keys {
		userUUID, err := uuidFromString(k.userID)
		if err != nil {
			continue
		}
		bandRows, err := q.ListUserPoolBand(ctx, dbgen.ListUserPoolBandParams{
			UserID:        userUUID,
			VenueCategory: k.category,
			Band:          k.band,
		})
		if err != nil {
			return errs.WrapCode(err, errs.Internal, "failed to list band for renormalize")
		}
		for i, row := range bandRows {
			placeUUID, err := uuidFromString(row.PlaceID)
			if err != nil {
				continue
			}
			if err := q.UpdateUserPoolEntryScores(ctx, dbgen.UpdateUserPoolEntryScoresParams{
				RankKey:             numericFromFloat(float64(i + 1)),
				Score:               row.Score,
				PlacementConfidence: row.PlacementConfidence,
				ComparisonsCount:    row.ComparisonsCount,
				ContradictionCount:  row.ContradictionCount,
				UserID:              userUUID,
				PlaceID:             placeUUID,
			}); err != nil {
				return errs.WrapCode(err, errs.Internal, "failed to renormalize rank key")
			}
		}
		_ = pgtype.UUID{}
	}
	return nil
}
